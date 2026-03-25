package tools

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ATMackay/agent/state"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

const (
	maxReadBytes     = 128 * 1024
	maxManifestBytes = 512 * 1024
	httpTimeout      = 90 * time.Second
)

type FetchRepoTreeConfig struct {
	WorkDir string
}

type FetchRepoTreeArgs struct {
	RepositoryURL string `json:"repository_url"`
	Ref           string `json:"ref,omitempty"`
	SubPath       string `json:"sub_path,omitempty"`
}

type FileEntry struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
	Size int64  `json:"size,omitempty"`
}

type FetchRepoTreeResult struct {
	FileCount int         `json:"file_count"`
	Manifest  []FileEntry `json:"manifest"`
}

// NewFetchRepoTool returns a fetch_repo_tree function tool.
func NewFetchRepoTreeTool(workDir string) (tool.Tool, error) {
	fetchRepoTreeTool, err := functiontool.New(
		functiontool.Config{
			Name:        "fetch_repo_tree",
			Description: "Download the GitHub repository to a local cache, build a source-file manifest, and store both in state.",
		},
		newFetchRepoTreeTool(workDir),
	)
	if err != nil {
		return nil, fmt.Errorf("create fetch_repo_tree tool: %w", err)
	}
	return fetchRepoTreeTool, nil
}

func newFetchRepoTreeTool(workDir string) func(tool.Context, FetchRepoTreeArgs) (FetchRepoTreeResult, error) {
	return func(ctx tool.Context, args FetchRepoTreeArgs) (FetchRepoTreeResult, error) {
		slog.Info("tool call", "function", "fetch_repo_tree", "args", toJSONString(args))
		localPath, manifest, err := fetchRepoManifest(args.RepositoryURL, args.Ref, args.SubPath, workDir)
		if err != nil {
			return FetchRepoTreeResult{}, err
		}

		raw, err := json.Marshal(manifest)
		if err != nil {
			return FetchRepoTreeResult{}, err
		}

		ctx.Actions().StateDelta[state.StateRepoURL] = args.RepositoryURL
		ctx.Actions().StateDelta[state.StateRepoRef] = args.Ref
		ctx.Actions().StateDelta[state.StateSubPath] = args.SubPath
		ctx.Actions().StateDelta[state.StateRepoManifest] = string(raw)
		ctx.Actions().StateDelta[state.StateRepoLocalPath] = localPath

		return FetchRepoTreeResult{
			FileCount: len(manifest),
			Manifest:  manifest,
		}, nil
	}
}

func fetchRepoManifest(repoURL, ref, subPath, workDir string) (string, []FileEntry, error) {
	if strings.TrimSpace(repoURL) == "" {
		return "", nil, fmt.Errorf("repository URL is required")
	}

	root, err := fetchRepository(repoURL, ref, workDir)
	if err != nil {
		return "", nil, err
	}

	root, err = resolveSubPath(root, subPath)
	if err != nil {
		return "", nil, err
	}

	manifest, err := buildManifest(root)
	if err != nil {
		return "", nil, err
	}

	return root, manifest, nil
}

func fetchRepository(repoURL, ref, workDir string) (string, error) {
	if workDir == "" {
		workDir = os.TempDir()
	}

	// Prefer fast HTTPS archive fetch for public GitHub repos.
	if owner, repo, ok := tryParseGitHubRepoURL(repoURL); ok {
		root, err := downloadAndExtractGitHubRepo(owner, repo, ref, workDir)
		if err == nil {
			return root, nil
		}
		// Fall through to git CLI fallback.
	}

	return cloneRepoWithGit(repoURL, ref, workDir)
}

func tryParseGitHubRepoURL(repoURL string) (owner, repo string, ok bool) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", "", false
	}
	host := strings.ToLower(u.Host)
	if host != "github.com" && host != "www.github.com" {
		return "", "", false
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return "", "", false
	}

	owner = parts[0]
	repo = strings.TrimSuffix(parts[1], ".git")
	if owner == "" || repo == "" {
		return "", "", false
	}
	return owner, repo, true
}

func downloadAndExtractGitHubRepo(owner, repo, ref, workDir string) (string, error) {
	dest, err := os.MkdirTemp(workDir, "repo-http-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	archiveURL := fmt.Sprintf("https://codeload.github.com/%s/%s/tar.gz", owner, repo)
	if strings.TrimSpace(ref) != "" {
		archiveURL = fmt.Sprintf("https://codeload.github.com/%s/%s/tar.gz/%s", owner, repo, url.PathEscape(ref))
	}

	req, err := http.NewRequest(http.MethodGet, archiveURL, nil)
	if err != nil {
		return "", fmt.Errorf("build archive request: %w", err)
	}
	req.Header.Set("User-Agent", "agent-documentor")

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download repository archive: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("error closing body", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download repository archive failed: %s", resp.Status)
	}

	if err := untarGzSafe(resp.Body, dest); err != nil {
		return "", fmt.Errorf("extract repository archive: %w", err)
	}

	root, err := firstSubdir(dest)
	if err != nil {
		return "", err
	}
	return root, nil
}

func cloneRepoWithGit(repoURL, ref, workDir string) (string, error) {
	dest, err := os.MkdirTemp(workDir, "repo-git-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	// Initialize empty repo so we can handle branch/tag/sha more flexibly.
	if err := runGit(dest, "init"); err != nil {
		return "", err
	}
	if err := runGit(dest, "remote", "add", "origin", repoURL); err != nil {
		return "", err
	}

	ref = strings.TrimSpace(ref)

	switch {
	case ref == "":
		// Default branch shallow fetch.
		if err := runGit(dest, "fetch", "--depth", "1", "origin"); err != nil {
			return "", fmt.Errorf("git fetch default branch: %w", err)
		}
		if err := runGit(dest, "checkout", "FETCH_HEAD"); err != nil {
			return "", fmt.Errorf("git checkout default branch: %w", err)
		}

	case looksLikeCommitish(ref):
		// Try exact commit-ish fetch.
		if err := runGit(dest, "fetch", "--depth", "1", "origin", ref); err == nil {
			if err := runGit(dest, "checkout", "FETCH_HEAD"); err != nil {
				return "", fmt.Errorf("git checkout fetched ref: %w", err)
			}
			return dest, nil
		}

		// Fallback: fetch all refs shallowly and checkout the requested ref.
		if err := runGit(dest, "fetch", "--depth", "1", "--tags", "origin"); err != nil {
			return "", fmt.Errorf("git fetch tags for ref %q: %w", ref, err)
		}
		if err := runGit(dest, "fetch", "--depth", "1", "origin", ref); err == nil {
			if err := runGit(dest, "checkout", "FETCH_HEAD"); err != nil {
				return "", fmt.Errorf("git checkout ref %q: %w", ref, err)
			}
			return dest, nil
		}
		if err := runGit(dest, "checkout", ref); err != nil {
			return "", fmt.Errorf("git checkout ref %q: %w", ref, err)
		}

	default:
		// Branch or tag name.
		if err := runGit(dest, "fetch", "--depth", "1", "--tags", "origin", ref); err == nil {
			if err := runGit(dest, "checkout", "FETCH_HEAD"); err != nil {
				return "", fmt.Errorf("git checkout ref %q: %w", ref, err)
			}
			return dest, nil
		}

		if err := runGit(dest, "fetch", "--depth", "1", "--tags", "origin"); err != nil {
			return "", fmt.Errorf("git fetch for ref %q: %w", ref, err)
		}
		if err := runGit(dest, "checkout", ref); err != nil {
			return "", fmt.Errorf("git checkout ref %q: %w", ref, err)
		}
	}

	return dest, nil
}

func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	// Avoid interactive prompts hanging the process.
	env := os.Environ()
	env = append(env, "GIT_TERMINAL_PROMPT=0")
	cmd.Env = env

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func looksLikeCommitish(ref string) bool {
	if len(ref) < 7 || len(ref) > 40 {
		return false
	}
	for _, r := range ref {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return false
		}
	}
	return true
}

func untarGzSafe(r io.Reader, dest string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer func() {
		if err := gzr.Close(); err != nil {
			slog.Error("error closing body", "error", err)
		}
	}()

	tr := tar.NewReader(gzr)
	cleanDest := filepath.Clean(dest)

	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		name := filepath.Clean(hdr.Name)
		if name == "." || name == "" {
			continue
		}

		target := filepath.Join(cleanDest, name)
		if !isWithinBase(cleanDest, target) {
			return fmt.Errorf("archive entry escapes destination: %q", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(f, tr)
			closeErr := f.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}

		case tar.TypeSymlink, tar.TypeLink:
			// Ignore links for safety/simplicity in v1.
			continue

		default:
			continue
		}
	}
}

func firstSubdir(root string) (string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", fmt.Errorf("read extracted repo dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			return filepath.Join(root, e.Name()), nil
		}
	}
	return "", fmt.Errorf("no extracted repository directory found")
}

func resolveSubPath(root, subPath string) (string, error) {
	root = filepath.Clean(root)
	if strings.TrimSpace(subPath) == "" {
		return root, nil
	}

	cleanSub := filepath.Clean(subPath)
	target := filepath.Join(root, cleanSub)
	if !isWithinBase(root, target) {
		return "", fmt.Errorf("invalid sub_path: %s", subPath)
	}

	info, err := os.Stat(target)
	if err != nil {
		return "", fmt.Errorf("sub_path not found: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("sub_path is not a directory: %s", subPath)
	}

	return target, nil
}

func buildManifest(root string) ([]FileEntry, error) {
	var manifest []FileEntry

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			if shouldSkipDir(rel) {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip symlinks entirely.
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		if !shouldIncludeFile(rel, info.Size()) {
			return nil
		}

		manifest = append(manifest, FileEntry{
			Path: rel,
			Kind: "file",
			Size: info.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(manifest, func(i, j int) bool {
		return manifest[i].Path < manifest[j].Path
	})

	return manifest, nil
}

func isWithinBase(base, target string) bool {
	base = filepath.Clean(base)
	target = filepath.Clean(target)

	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && rel != "..")
}

func shouldSkipDir(rel string) bool {
	switch filepath.Base(rel) {
	case ".git", ".github", "vendor", "node_modules", "dist", "build", "bin", "coverage", ".next", ".turbo":
		return true
	default:
		return false
	}
}

func shouldIncludeFile(rel string, size int64) bool {
	if size <= 0 || size > maxManifestBytes {
		return false
	}

	switch strings.ToLower(filepath.Ext(rel)) {
	case ".go", ".md", ".txt", ".yaml", ".yml", ".json", ".toml", ".proto", ".sql", ".sh", ".py", ".js", ".ts", ".tsx", ".jsx", ".java", ".rb", ".rs", ".c", ".h", ".cpp", ".hpp":
		return true
	default:
		return false
	}
}

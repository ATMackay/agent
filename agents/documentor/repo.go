package documentor

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxReadBytes = 128 * 1024

func fetchRepoManifest(repoURL, ref, subPath, workDir string) (string, []FileEntry, error) {
	owner, repo, err := parseGitHubRepoURL(repoURL)
	if err != nil {
		return "", nil, err
	}

	root, err := downloadAndExtractGitHubRepo(owner, repo, ref, workDir)
	if err != nil {
		return "", nil, err
	}

	if subPath != "" {
		root = filepath.Join(root, filepath.Clean(subPath))
		info, err := os.Stat(root)
		if err != nil {
			return "", nil, fmt.Errorf("sub_path not found: %w", err)
		}
		if !info.IsDir() {
			return "", nil, fmt.Errorf("sub_path is not a directory: %s", subPath)
		}
	}

	manifest, err := buildManifest(root)
	if err != nil {
		return "", nil, err
	}

	return root, manifest, nil
}

func parseGitHubRepoURL(repoURL string) (string, string, error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid repository URL: %w", err)
	}
	if !strings.EqualFold(u.Host, "github.com") && !strings.EqualFold(u.Host, "www.github.com") {
		return "", "", fmt.Errorf("only github.com repositories are supported")
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("repository URL must look like https://github.com/{owner}/{repo}")
	}

	owner := parts[0]
	repo := strings.TrimSuffix(parts[1], ".git")
	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("invalid GitHub repository URL")
	}
	return owner, repo, nil
}

func downloadAndExtractGitHubRepo(owner, repo, ref, workDir string) (string, error) {
	if workDir == "" {
		workDir = os.TempDir()
	}

	dest, err := os.MkdirTemp(workDir, "repo-*")
	if err != nil {
		return "", err
	}

	archiveURL := fmt.Sprintf("https://codeload.github.com/%s/%s/tar.gz", owner, repo)
	if strings.TrimSpace(ref) != "" {
		archiveURL = fmt.Sprintf("https://codeload.github.com/%s/%s/tar.gz/%s", owner, repo, url.PathEscape(ref))
	}

	req, err := http.NewRequest(http.MethodGet, archiveURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "agent-documentor")

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download repository archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download repository archive failed: %s", resp.Status)
	}

	if err := untarGz(resp.Body, dest); err != nil {
		return "", err
	}

	root, err := firstSubdir(dest)
	if err != nil {
		return "", err
	}
	return root, nil
}

func untarGz(r io.Reader, dest string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, filepath.Clean(hdr.Name))
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
			if _, err := io.Copy(f, tr); err != nil {
				_ = f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		}
	}
}

func firstSubdir(root string) (string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() {
			return filepath.Join(root, e.Name()), nil
		}
	}
	return "", fmt.Errorf("no extracted repository directory found")
}

func buildManifest(root string) ([]FileEntry, error) {
	var manifest []FileEntry

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		if info.IsDir() {
			if shouldSkipDir(rel) {
				return filepath.SkipDir
			}
			return nil
		}

		if !shouldIncludeFile(rel, info.Size()) {
			return nil
		}

		manifest = append(manifest, FileEntry{
			Path: filepath.ToSlash(rel),
			Kind: "file",
			Size: info.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return manifest, nil
}

func readRepoFileFromCachedCheckout(localRoot, relPath string) (string, error) {
	cleanRel := filepath.Clean(relPath)
	fullPath := filepath.Join(localRoot, cleanRel)

	if !strings.HasPrefix(fullPath, filepath.Clean(localRoot)+string(os.PathSeparator)) &&
		filepath.Clean(fullPath) != filepath.Clean(localRoot) {
		return "", fmt.Errorf("invalid repository path: %s", relPath)
	}

	b, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("read repository file %s: %w", relPath, err)
	}

	if len(b) > maxReadBytes {
		b = b[:maxReadBytes]
	}
	return string(b), nil
}

func shouldSkipDir(rel string) bool {
	switch filepath.Base(rel) {
	case ".git", ".github", "vendor", "node_modules", "dist", "build", "bin":
		return true
	default:
		return false
	}
}

func shouldIncludeFile(rel string, size int64) bool {
	if size <= 0 || size > 512*1024 {
		return false
	}

	switch strings.ToLower(filepath.Ext(rel)) {
	case ".go", ".md", ".txt", ".yaml", ".yml", ".json", ".toml", ".proto", ".sql", ".sh":
		return true
	default:
		return false
	}
}

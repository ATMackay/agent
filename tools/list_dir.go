package tools

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ATMackay/agent/state"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// ListDirArgs are the inputs to the list_dir tool.
type ListDirArgs struct {
	Path     string `json:"path"`
	MaxDepth int    `json:"max_depth,omitempty"`
}

// ListDirEntry describes a single file or directory.
type ListDirEntry struct {
	Path string `json:"path"`
	Kind string `json:"kind"` // "file" | "dir"
	Size int64  `json:"size,omitempty"`
}

// ListDirResult is returned by the list_dir tool.
type ListDirResult struct {
	Root       string         `json:"root"`
	EntryCount int            `json:"entry_count"`
	Entries    []ListDirEntry `json:"entries"`
}

// NewListDirTool returns a list_dir function tool.
func NewListDirTool() (tool.Tool, error) {
	t, err := functiontool.New(
		functiontool.Config{
			Name:        "list_dir",
			Description: "List the contents of a local directory up to the given depth. Returns paths, kinds (file/dir), and file sizes. Use this to explore the filesystem before reading files.",
		},
		newListDirTool(),
	)
	if err != nil {
		return nil, fmt.Errorf("create list_dir tool: %w", err)
	}
	return t, nil
}

func newListDirTool() func(tool.Context, ListDirArgs) (ListDirResult, error) {
	return func(ctx tool.Context, args ListDirArgs) (ListDirResult, error) {
		slog.Info("tool call", "function", "list_dir", "args", toJSONString(args))

		targetPath := resolveLocalPath(ctx, args.Path)

		info, err := os.Stat(targetPath)
		if err != nil {
			return ListDirResult{}, fmt.Errorf("stat %q: %w", args.Path, err)
		}
		if !info.IsDir() {
			return ListDirResult{}, fmt.Errorf("%q is not a directory", args.Path)
		}

		maxDepth := args.MaxDepth
		if maxDepth <= 0 {
			maxDepth = 3
		}
		if maxDepth > 10 {
			maxDepth = 10
		}

		var entries []ListDirEntry

		err = filepath.WalkDir(targetPath, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}

			rel, relErr := filepath.Rel(targetPath, path)
			if relErr != nil || rel == "." {
				return nil
			}

			depth := strings.Count(rel, string(os.PathSeparator)) + 1
			if depth > maxDepth {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			rel = filepath.ToSlash(rel)

			if d.IsDir() {
				if shouldSkipDir(rel) {
					return filepath.SkipDir
				}
				entries = append(entries, ListDirEntry{Path: rel, Kind: "dir"})
				return nil
			}

			fi, infoErr := d.Info()
			if infoErr != nil {
				return nil
			}

			entries = append(entries, ListDirEntry{
				Path: rel,
				Kind: "file",
				Size: fi.Size(),
			})
			return nil
		})
		if err != nil {
			return ListDirResult{}, fmt.Errorf("walk %q: %w", args.Path, err)
		}

		return ListDirResult{
			Root:       targetPath,
			EntryCount: len(entries),
			Entries:    entries,
		}, nil
	}
}

// resolveLocalPath resolves path relative to work_dir state, or returns it as-is if absolute.
func resolveLocalPath(ctx tool.Context, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	workDir := getWorkDir(ctx)
	if workDir != "" {
		return filepath.Clean(filepath.Join(workDir, path))
	}
	return filepath.Clean(path)
}

// getWorkDir retrieves the work_dir value from session state.
func getWorkDir(ctx tool.Context) string {
	v, err := ctx.State().Get(state.StateWorkDir)
	if err != nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

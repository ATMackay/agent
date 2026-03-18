package tools

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

type SearchRepoArgs struct {
	Query        string `json:"query"`
	PathPrefix   string `json:"path_prefix,omitempty"`
	MaxResults   int    `json:"max_results,omitempty"`
	ContextLines int    `json:"context_lines,omitempty"`
}

type SearchMatch struct {
	Path      string `json:"path"`
	Line      int    `json:"line"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Snippet   string `json:"snippet"`
}

type SearchRepoResult struct {
	Query      string        `json:"query"`
	MatchCount int           `json:"match_count"`
	Truncated  bool          `json:"truncated"`
	Matches    []SearchMatch `json:"matches"`
}

// NewSearchRepoTool returns a repo search tool
func NewSearchRepoTool() (tool.Tool, error) {
	searchRepoTool, err := functiontool.New(
		functiontool.Config{
			Name:        "search_repo",
			Description: "Search the cached repository for text matches and return matching file paths, line numbers, and short snippets. Use this before reading files to locate relevant symbols, functions, types, config keys, or strings.",
		},
		newSearchRepoTool(),
	)
	if err != nil {
		return nil, fmt.Errorf("create search_repo tool: %w", err)
	}
	return searchRepoTool, nil
}

func newSearchRepoTool() func(tool.Context, SearchRepoArgs) (SearchRepoResult, error) {
	return func(ctx tool.Context, args SearchRepoArgs) (SearchRepoResult, error) {
		slog.Info("tool call", "function", "search_repo", "args", toJSONString(args))

		if strings.TrimSpace(args.Query) == "" {
			return SearchRepoResult{}, fmt.Errorf("query is required")
		}

		// Sanitize tool args to prevent context overload
		if args.MaxResults <= 0 {
			args.MaxResults = 20
		}
		if args.MaxResults > 100 {
			args.MaxResults = 100
		}
		if args.ContextLines < 0 {
			args.ContextLines = 0
		}
		if args.ContextLines > 3 {
			args.ContextLines = 3
		}

		v, err := ctx.State().Get(StateRepoLocalPath)
		if err != nil {
			return SearchRepoResult{}, fmt.Errorf("read repo local path from state: %w", err)
		}

		localPath, ok := v.(string)
		if !ok || localPath == "" {
			return SearchRepoResult{}, fmt.Errorf("repository cache not initialized; call fetch_repo_tree first")
		}

		searchRoot := localPath
		if args.PathPrefix != "" {
			searchRoot = filepath.Join(localPath, filepath.Clean(args.PathPrefix))
		}

		var matches []SearchMatch
		truncated := false

		err = filepath.Walk(searchRoot, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			rel, relErr := filepath.Rel(localPath, path)
			if relErr != nil {
				return nil
			}

			if info.IsDir() {
				if shouldSkipDir(rel) {
					return filepath.SkipDir
				}
				return nil
			}

			if shouldSkipFile(rel, info) {
				return nil
			}

			fileMatches, err := searchFile(path, rel, args.Query, args.ContextLines, args.MaxResults-len(matches))
			if err != nil {
				return nil
			}

			matches = append(matches, fileMatches...)
			if len(matches) >= args.MaxResults {
				truncated = true
				return fmt.Errorf("search result limit reached")
			}
			return nil
		})

		// swallow the sentinel-ish stop condition
		if err != nil && !strings.Contains(err.Error(), "search result limit reached") {
			return SearchRepoResult{}, err
		}

		return SearchRepoResult{
			Query:      args.Query,
			MatchCount: len(matches),
			Truncated:  truncated,
			Matches:    matches,
		}, nil
	}
}

func searchFile(path, relPath, query string, contextLines, remaining int) ([]SearchMatch, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			slog.Error("error closing file", "error", err)
		}
	}()

	var lines []string
	scanner := bufio.NewScanner(f)

	// allow longer lines than bufio default
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	q := strings.ToLower(query)
	var matches []SearchMatch

	for i, line := range lines {
		if !strings.Contains(strings.ToLower(line), q) {
			continue
		}

		start := i - contextLines
		if start < 0 {
			start = 0
		}
		end := i + contextLines
		if end >= len(lines) {
			end = len(lines) - 1
		}

		snippet := strings.Join(lines[start:end+1], "\n")

		matches = append(matches, SearchMatch{
			Path:      relPath,
			Line:      i + 1,
			StartLine: start + 1,
			EndLine:   end + 1,
			Snippet:   snippet,
		})

		if len(matches) >= remaining {
			break
		}
	}

	return matches, nil
}

func shouldSkipFile(rel string, info os.FileInfo) bool {
	if info.Size() > 2*1024*1024 {
		return true
	}

	ext := strings.ToLower(filepath.Ext(rel))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".pdf", ".zip", ".gz", ".tar", ".jar", ".bin", ".exe", ".so", ".dll":
		return true
	}

	base := filepath.Base(rel)
	if strings.HasPrefix(base, ".") && base != ".env" {
		return true
	}

	return false
}

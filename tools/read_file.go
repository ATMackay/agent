package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

type ReadFileArgs struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
	MaxBytes  int    `json:"max_bytes,omitempty"`
	FullFile  bool   `json:"full_file,omitempty"`
}

type ReadFileResult struct {
	Path       string `json:"path"`
	StartLine  int    `json:"start_line"`
	EndLine    int    `json:"end_line"`
	TotalLines int    `json:"total_lines"`
	Truncated  bool   `json:"truncated"`
	Content    string `json:"content"`
}

type LoadedFileMeta struct {
	Path        string `json:"path"`
	StartLine   int    `json:"start_line"`
	EndLine     int    `json:"end_line"`
	BytesRead   int    `json:"bytes_read"`
	SnippetOnly bool   `json:"snippet_only"`
}

func newReadFileTool() func(tool.Context, ReadFileArgs) (ReadFileResult, error) {
	return func(ctx tool.Context, args ReadFileArgs) (ReadFileResult, error) {
		slog.Info("tool call", "function", "read_repo_file", "args", toJSONString(args))

		v, err := ctx.State().Get(StateRepoLocalPath)
		if err != nil {
			return ReadFileResult{}, fmt.Errorf("read repo local path from state: %w", err)
		}

		localPath, ok := v.(string)
		if !ok || localPath == "" {
			return ReadFileResult{}, fmt.Errorf("repository cache not initialized; call fetch_repo_tree first")
		}

		result, err := ReadFileSnippetFromCachedCheckout(localPath, args)
		if err != nil {
			return ReadFileResult{}, err
		}

		loaded := map[string]LoadedFileMeta{}
		existing, err := ctx.State().Get(StateLoadedFiles)
		if err == nil && existing != nil {
			if s, ok := existing.(string); ok && s != "" {
				_ = json.Unmarshal([]byte(s), &loaded)
			}
		}

		loaded[args.Path] = LoadedFileMeta{
			Path:        result.Path,
			StartLine:   result.StartLine,
			EndLine:     result.EndLine,
			BytesRead:   len(result.Content),
			SnippetOnly: !args.FullFile || result.StartLine != 1 || result.EndLine != result.TotalLines || result.Truncated,
		}

		raw, _ := json.Marshal(loaded)
		ctx.Actions().StateDelta[StateLoadedFiles] = string(raw)

		return result, nil
	}
}

func ReadFileSnippetFromCachedCheckout(localPath string, args ReadFileArgs) (ReadFileResult, error) {
	if strings.TrimSpace(args.Path) == "" {
		return ReadFileResult{}, fmt.Errorf("path is required")
	}

	cleanRel := filepath.Clean(args.Path)
	if cleanRel == "." || cleanRel == ".." || strings.HasPrefix(cleanRel, ".."+string(os.PathSeparator)) {
		return ReadFileResult{}, fmt.Errorf("invalid path %q", args.Path)
	}

	fullPath := filepath.Join(localPath, cleanRel)

	absRoot, err := filepath.Abs(localPath)
	if err != nil {
		return ReadFileResult{}, fmt.Errorf("resolve repo root: %w", err)
	}
	absFile, err := filepath.Abs(fullPath)
	if err != nil {
		return ReadFileResult{}, fmt.Errorf("resolve file path: %w", err)
	}
	if absFile != absRoot && !strings.HasPrefix(absFile, absRoot+string(os.PathSeparator)) {
		return ReadFileResult{}, fmt.Errorf("path escapes repository root: %q", args.Path)
	}

	info, err := os.Stat(absFile)
	if err != nil {
		return ReadFileResult{}, fmt.Errorf("stat file %s: %w", args.Path, err)
	}
	if info.IsDir() {
		return ReadFileResult{}, fmt.Errorf("path %q is a directory, not a file", args.Path)
	}

	lines, err := readFileLines(absFile)
	if err != nil {
		return ReadFileResult{}, fmt.Errorf("read file %s: %w", args.Path, err)
	}

	totalLines := len(lines)
	if totalLines == 0 {
		return ReadFileResult{
			Path:       args.Path,
			StartLine:  0,
			EndLine:    0,
			TotalLines: 0,
			Truncated:  false,
			Content:    "",
		}, nil
	}

	const (
		defaultSnippetLines = 120
		defaultMaxBytes     = 8_000
		hardMaxBytes        = 20_000
	)

	maxBytes := args.MaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultMaxBytes
	}
	if maxBytes > hardMaxBytes {
		maxBytes = hardMaxBytes
	}

	var startLine, endLine int

	switch {
	case args.FullFile:
		startLine = 1
		endLine = totalLines

	case args.StartLine == 0 && args.EndLine == 0:
		startLine = 1
		endLine = min(totalLines, defaultSnippetLines)

	default:
		startLine = args.StartLine
		endLine = args.EndLine

		if startLine <= 0 {
			startLine = 1
		}
		if endLine <= 0 {
			endLine = startLine + defaultSnippetLines - 1
		}
		if endLine < startLine {
			return ReadFileResult{}, fmt.Errorf("end_line must be >= start_line")
		}
		if startLine > totalLines {
			return ReadFileResult{}, fmt.Errorf("start_line %d is beyond file length %d", startLine, totalLines)
		}
		if endLine > totalLines {
			endLine = totalLines
		}
	}

	selected := lines[startLine-1 : endLine]
	content, actualEndLine, truncated := joinLinesWithinByteLimit(selected, startLine, maxBytes)

	return ReadFileResult{
		Path:       args.Path,
		StartLine:  startLine,
		EndLine:    actualEndLine,
		TotalLines: totalLines,
		Truncated:  truncated || actualEndLine < endLine,
		Content:    content,
	}, nil
}

func readFileLines(path string) ([]string, error) {
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

	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func joinLinesWithinByteLimit(lines []string, startLine, maxBytes int) (content string, actualEndLine int, truncated bool) {
	if len(lines) == 0 {
		return "", startLine - 1, false
	}

	var b strings.Builder
	actualEndLine = startLine - 1

	for i, line := range lines {
		addition := len(line)
		if i > 0 {
			addition++
		}

		if b.Len()+addition > maxBytes {
			truncated = true
			break
		}

		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(line)
		actualEndLine = startLine + i
	}

	return b.String(), actualEndLine, truncated
}

// NewFetchRepoTool returns a fetch_repo_tree function tool.
func NewReadFileTool() (tool.Tool, error) {
	ReadFileTool, err := functiontool.New(
		functiontool.Config{
			Name:        "read_repo_file",
			Description: "Read a repository file from the cached checkout and store it in state.",
		},
		newReadFileTool(),
	)
	if err != nil {
		return nil, fmt.Errorf("create read_repo_file tool: %w", err)
	}
	return ReadFileTool, nil
}

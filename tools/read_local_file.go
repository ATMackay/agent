package tools

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// ReadLocalFileArgs are the inputs to the read_local_file tool.
type ReadLocalFileArgs struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
	MaxBytes  int    `json:"max_bytes,omitempty"`
	FullFile  bool   `json:"full_file,omitempty"`
}

// NewReadLocalFileTool returns a read_local_file function tool.
func NewReadLocalFileTool() (tool.Tool, error) {
	t, err := functiontool.New(
		functiontool.Config{
			Name: "read_local_file",
			Description: "Read a local file from the filesystem and return its content. " +
				"Supports line ranges and byte limits. Paths are relative to work_dir or absolute. " +
				"Use this to read source code, text documents, configs, and any other text files.",
		},
		newReadLocalFileTool(),
	)
	if err != nil {
		return nil, fmt.Errorf("create read_local_file tool: %w", err)
	}
	return t, nil
}

func newReadLocalFileTool() func(tool.Context, ReadLocalFileArgs) (ReadFileResult, error) {
	return func(ctx tool.Context, args ReadLocalFileArgs) (ReadFileResult, error) {
		slog.Info("tool call", "function", "read_local_file", "args", toJSONString(args))

		if strings.TrimSpace(args.Path) == "" {
			return ReadFileResult{}, fmt.Errorf("path is required")
		}

		absPath := resolveLocalPath(ctx, args.Path)

		info, err := os.Stat(absPath)
		if err != nil {
			return ReadFileResult{}, fmt.Errorf("stat %q: %w", args.Path, err)
		}
		if info.IsDir() {
			return ReadFileResult{}, fmt.Errorf("%q is a directory, not a file", args.Path)
		}

		// Reuse the shared snippet reader, passing the resolved abs path directly.
		return readFileSnippet(absPath, args.Path, ReadFileArgs{
			StartLine: args.StartLine,
			EndLine:   args.EndLine,
			MaxBytes:  args.MaxBytes,
			FullFile:  args.FullFile,
		})
	}
}

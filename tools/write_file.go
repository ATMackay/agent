package tools

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

type WriteFileArgs struct {
	Markdown   string `json:"markdown"`
	OutputPath string `json:"output_path,omitempty"`
}

type WriteFileResult struct {
	Path string `json:"path"`
}

// NewWriteFileTool returns a write_output_file function tool.
func NewWriteFileTool() (tool.Tool, error) {
	WriteFileTool, err := functiontool.New(
		functiontool.Config{
			Name:        "write_output_file",
			Description: "Write markdown documentation to the requested output file.",
		},
		newWriteFileTool(),
	)
	if err != nil {
		return nil, fmt.Errorf("create write_file tool: %w", err)
	}
	return WriteFileTool, nil
}

func newWriteFileTool() func(tool.Context, WriteFileArgs) (WriteFileResult, error) {
	return func(ctx tool.Context, args WriteFileArgs) (WriteFileResult, error) {
		slog.Info("tool call", "function", string(WriteFile), "content_length", len(toJSONString(args)))
		out := args.OutputPath
		if out == "" {
			v, err := ctx.State().Get(StateOutputPath)
			if err == nil {
				if s, ok := v.(string); ok {
					out = s
				}
			}
		}
		if out == "" {
			return WriteFileResult{}, fmt.Errorf("output path is required")
		}

		if err := writeTextFile(out, args.Markdown); err != nil {
			return WriteFileResult{}, err
		}

		ctx.Actions().StateDelta[StateDocumentation] = args.Markdown
		return WriteFileResult{Path: out}, nil
	}
}

// writeTextFile creates parent directories as needed and writes content to path.
func writeTextFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

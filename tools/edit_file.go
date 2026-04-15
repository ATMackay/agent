package tools

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// EditFileArgs are the inputs to the edit_file tool.
type EditFileArgs struct {
	Path       string `json:"path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

// EditFileResult is returned by the edit_file tool.
type EditFileResult struct {
	Path     string `json:"path"`
	Replaced int    `json:"replaced"`
}

// NewEditFileTool returns an edit_file function tool.
func NewEditFileTool() (tool.Tool, error) {
	t, err := functiontool.New(
		functiontool.Config{
			Name: "edit_file",
			Description: "Edit a local file by replacing an exact string with a new string. " +
				"By default replaces only one occurrence and returns an error if old_string is not found " +
				"or matches more than once. Set replace_all to true to replace every occurrence. " +
				"Paths are relative to work_dir or absolute.",
		},
		newEditFileTool(),
	)
	if err != nil {
		return nil, fmt.Errorf("create edit_file tool: %w", err)
	}
	return t, nil
}

func newEditFileTool() func(tool.Context, EditFileArgs) (EditFileResult, error) {
	return func(ctx tool.Context, args EditFileArgs) (EditFileResult, error) {
		slog.Info("tool call", "function", "edit_file", "path", args.Path,
			"old_len", len(args.OldString), "new_len", len(args.NewString))

		if strings.TrimSpace(args.Path) == "" {
			return EditFileResult{}, fmt.Errorf("path is required")
		}
		if args.OldString == "" {
			return EditFileResult{}, fmt.Errorf("old_string is required")
		}

		absPath := resolveLocalPath(ctx, args.Path)

		data, err := os.ReadFile(absPath)
		if err != nil {
			return EditFileResult{}, fmt.Errorf("read %q: %w", args.Path, err)
		}

		content := string(data)
		count := strings.Count(content, args.OldString)

		if count == 0 {
			return EditFileResult{}, fmt.Errorf("old_string not found in %q", args.Path)
		}
		if !args.ReplaceAll && count > 1 {
			return EditFileResult{}, fmt.Errorf(
				"old_string matches %d times in %q; provide more context to make it unique, or set replace_all=true",
				count, args.Path,
			)
		}

		var updated string
		replaced := 0
		if args.ReplaceAll {
			updated = strings.ReplaceAll(content, args.OldString, args.NewString)
			replaced = count
		} else {
			updated = strings.Replace(content, args.OldString, args.NewString, 1)
			replaced = 1
		}

		info, err := os.Stat(absPath)
		if err != nil {
			return EditFileResult{}, fmt.Errorf("stat %q: %w", args.Path, err)
		}

		if err := os.WriteFile(absPath, []byte(updated), info.Mode()); err != nil {
			return EditFileResult{}, fmt.Errorf("write %q: %w", args.Path, err)
		}

		return EditFileResult{Path: args.Path, Replaced: replaced}, nil
	}
}

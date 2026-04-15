package tools

import (
	"encoding/json"
	"fmt"

	"google.golang.org/adk/tool"
)

type Kind string

const (
	// Documentor tools.
	FetchRepoTree Kind = "fetch_repo_tree"
	ReadFile      Kind = "read_file"
	SearchFiles   Kind = "search_repo"
	WriteFile     Kind = "write_file"

	// Analyzer tools.
	ListDir       Kind = "list_dir"
	ReadLocalFile Kind = "read_local_file"
	EditFile      Kind = "edit_file"
	ExecCommand   Kind = "exec_command"
)

// GetToolByEnum returns the tool.Tool for the given Kind, initialised with any
// required dependency configuration from deps.
func GetToolByEnum(kind Kind, deps *Deps) (tool.Tool, error) {
	switch kind {
	// Documentor tools
	case FetchRepoTree:
		cfg, err := getConfig[FetchRepoTreeConfig](kind, deps)
		if err != nil {
			return nil, err
		}
		if cfg.WorkDir == "" {
			return nil, fmt.Errorf("fetch_repo_tree requires WorkDir")
		}
		return NewFetchRepoTreeTool(cfg.WorkDir)
	case ReadFile:
		return NewReadFileTool()
	case SearchFiles:
		return NewSearchFilesTool()
	case WriteFile:
		return NewWriteFileTool()

	// Analyzer tools
	case ListDir:
		return NewListDirTool()
	case ReadLocalFile:
		return NewReadLocalFileTool()
	case EditFile:
		return NewEditFileTool()
	case ExecCommand:
		return NewExecCommandTool()

	default:
		return nil, fmt.Errorf("invalid tool kind: %q", kind)
	}
}

func GetTools(kinds []Kind, deps *Deps) ([]tool.Tool, error) {
	out := make([]tool.Tool, 0, len(kinds))
	for _, kind := range kinds {
		t, err := GetToolByEnum(kind, deps)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func toJSONString(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

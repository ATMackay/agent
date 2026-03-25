package tools

import (
	"encoding/json"
	"fmt"

	"google.golang.org/adk/tool"
)

type Kind string

const (
	FetchRepoTree Kind = "fetch_repo_tree"
	ReadFile      Kind = "read_file"
	SearchFiles   Kind = "search_repo"
	WriteFile     Kind = "write_file"
)

// GetToolByEnum
func GetToolByEnum(kind Kind, deps *Deps) (tool.Tool, error) {
	switch kind {
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

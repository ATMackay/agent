package documentor

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

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

func newFetchRepoTreeTool(cfg *Config) func(tool.Context, FetchRepoTreeArgs) (FetchRepoTreeResult, error) {
	return func(ctx tool.Context, args FetchRepoTreeArgs) (FetchRepoTreeResult, error) {
		slog.Info("tool call", "function", "fetch_repo_tree", "args", toJSONString(args))
		localPath, manifest, err := fetchRepoManifest(args.RepositoryURL, args.Ref, args.SubPath, cfg.WorkDir)
		if err != nil {
			return FetchRepoTreeResult{}, err
		}

		raw, err := json.Marshal(manifest)
		if err != nil {
			return FetchRepoTreeResult{}, err
		}

		ctx.Actions().StateDelta[StateRepoURL] = args.RepositoryURL
		ctx.Actions().StateDelta[StateRepoRef] = args.Ref
		ctx.Actions().StateDelta[StateSubPath] = args.SubPath
		ctx.Actions().StateDelta[StateRepoManifest] = string(raw)
		ctx.Actions().StateDelta[StateRepoLocalPath] = localPath

		return FetchRepoTreeResult{
			FileCount: len(manifest),
			Manifest:  manifest,
		}, nil
	}
}

// NewFetchRepoTool returns a fetch_repo_tree function tool.
func NewFetchRepoTreeTool(cfg *Config) (tool.Tool, error) {
	fetchRepoTreeTool, err := functiontool.New(
		functiontool.Config{
			Name:        "fetch_repo_tree",
			Description: "Download the GitHub repository to a local cache, build a source-file manifest, and store both in state.",
		},
		newFetchRepoTreeTool(cfg),
	)
	if err != nil {
		return nil, fmt.Errorf("create fetch_repo_tree tool: %w", err)
	}
	return fetchRepoTreeTool, nil
}

type ReadRepoFileArgs struct {
	Path string `json:"path"`
}

type ReadRepoFileResult struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func newReadRepoFileTool() func(tool.Context, ReadRepoFileArgs) (ReadRepoFileResult, error) {
	return func(ctx tool.Context, args ReadRepoFileArgs) (ReadRepoFileResult, error) {
		slog.Info("tool call", "function", "read_repo_file", "args", toJSONString(args))
		v, err := ctx.State().Get(StateRepoLocalPath)
		if err != nil {
			return ReadRepoFileResult{}, fmt.Errorf("read repo local path from state: %w", err)
		}

		localPath, ok := v.(string)
		if !ok || localPath == "" {
			return ReadRepoFileResult{}, fmt.Errorf("repository cache not initialized; call fetch_repo_tree first")
		}

		content, err := readRepoFileFromCachedCheckout(localPath, args.Path)
		if err != nil {
			return ReadRepoFileResult{}, err
		}

		loaded := map[string]string{}
		existing, err := ctx.State().Get(StateLoadedFiles)
		if err == nil && existing != nil {
			if s, ok := existing.(string); ok && s != "" {
				_ = json.Unmarshal([]byte(s), &loaded)
			}
		}

		loaded[args.Path] = content
		raw, _ := json.Marshal(loaded)
		ctx.Actions().StateDelta[StateLoadedFiles] = string(raw)

		return ReadRepoFileResult{
			Path:    args.Path,
			Content: content,
		}, nil
	}
}

// NewFetchRepoTool returns a fetch_repo_tree function tool.
func NewReadRepoFileTool(_ *Config) (tool.Tool, error) {
	readRepoFileTool, err := functiontool.New(
		functiontool.Config{
			Name:        "read_repo_file",
			Description: "Read a repository file from the cached checkout and store it in state.",
		},
		newReadRepoFileTool(),
	)
	if err != nil {
		return nil, fmt.Errorf("create read_repo_file tool: %w", err)
	}
	return readRepoFileTool, nil
}

type WriteOutputFileArgs struct {
	Markdown   string `json:"markdown"`
	OutputPath string `json:"output_path,omitempty"`
}

type WriteOutputFileResult struct {
	Path string `json:"path"`
}

func newWriteOutputFileTool() func(tool.Context, WriteOutputFileArgs) (WriteOutputFileResult, error) {
	return func(ctx tool.Context, args WriteOutputFileArgs) (WriteOutputFileResult, error) {
		slog.Info("tool call", "function", "write_output_file", "content_length", len(toJSONString(args)))
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
			return WriteOutputFileResult{}, fmt.Errorf("output path is required")
		}

		if err := writeTextFile(out, args.Markdown); err != nil {
			return WriteOutputFileResult{}, err
		}

		ctx.Actions().StateDelta[StateDocumentation] = args.Markdown
		return WriteOutputFileResult{Path: out}, nil
	}
}

// writeTextFile creates parent directories as needed and writes content to path.
func writeTextFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// NewWriteOutputTool returns a write_output_file function tool.
func NewWriteOutputTool(_ *Config) (tool.Tool, error) {
	writeOutputTool, err := functiontool.New(
		functiontool.Config{
			Name:        "write_output_file",
			Description: "Write markdown documentation to the requested output file.",
		},
		newWriteOutputFileTool(),
	)
	if err != nil {
		return nil, fmt.Errorf("create write_output_file tool: %w", err)
	}
	return writeOutputTool, nil
}

func toJSONString(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

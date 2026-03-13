package documentor

import (
	"encoding/json"
	"fmt"

	"google.golang.org/adk/tool"
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

func newFetchRepoTreeTool(cfg Config) func(tool.Context, FetchRepoTreeArgs) (FetchRepoTreeResult, error) {
	return func(ctx tool.Context, args FetchRepoTreeArgs) (FetchRepoTreeResult, error) {
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

type ReadRepoFileArgs struct {
	Path string `json:"path"`
}

type ReadRepoFileResult struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func newReadRepoFileTool() func(tool.Context, ReadRepoFileArgs) (ReadRepoFileResult, error) {
	return func(ctx tool.Context, args ReadRepoFileArgs) (ReadRepoFileResult, error) {
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

type WriteOutputFileArgs struct {
	Markdown   string `json:"markdown"`
	OutputPath string `json:"output_path,omitempty"`
}

type WriteOutputFileResult struct {
	Path string `json:"path"`
}

func newWriteOutputFileTool() func(tool.Context, WriteOutputFileArgs) (WriteOutputFileResult, error) {
	return func(ctx tool.Context, args WriteOutputFileArgs) (WriteOutputFileResult, error) {
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

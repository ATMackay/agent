package documentor

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"
)

type Config struct {
	ModelName string
	APIKey    string
	WorkDir   string
}

func NewDocumentorAgent(ctx context.Context, cfg Config) (agent.Agent, error) {
	if cfg.ModelName == "" {
		cfg.ModelName = "gemini-2.5-pro"
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	model, err := gemini.NewModel(ctx, cfg.ModelName, &genai.ClientConfig{
		APIKey: cfg.APIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("create model: %w", err)
	}

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

	return llmagent.New(llmagent.Config{
		Name:        "documentor",
		Model:       model,
		Description: "Retrieves code from a GitHub repository and writes high-quality markdown documentation.",
		Instruction: buildInstruction(),
		Tools: []tool.Tool{
			fetchRepoTreeTool,
			readRepoFileTool,
			writeOutputTool,
		},
		OutputKey: StateDocumentation,
	})
}

func buildInstruction() string {
	return `
You are a code documentation agent.

Repository: {repo_url}
Ref: {repo_ref?}
Sub-path filter: {sub_path?}
Output path: {output_path}
Max files to read: {max_files?}

Workflow:
1. Call fetch_repo_tree first using the repository_url, ref, and sub_path from state.
2. Inspect the manifest and identify the most relevant files for architecture and code-level documentation.
3. Prefer entry points, cmd/, internal/, pkg/, config, and core domain files.
4. Skip tests, generated files, vendor, binaries, and irrelevant assets unless they are central.
5. Do not read more than max_files files.
6. Call read_repo_file for each selected file.
7. Write detailed maintainers' documentation in markdown.
8. Call write_output_file with the completed markdown and output_path.

Requirements:
- Explain architecture and package responsibilities.
- Explain key types, functions, interfaces, and control flow.
- Explain configuration, dependencies, and extension points.
- Mention important file paths and symbol names.
- Do not invent behavior beyond the code retrieved.
- If repository coverage is partial, say so explicitly.
`
}

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

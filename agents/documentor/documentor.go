package documentor

import (
	"context"
	"fmt"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"
)

type Documentor struct {
	inner agent.Agent
}

// NewDocumentorAgent returns a Documentor.
func NewDocumentorAgent(ctx context.Context, cfg Config) (*Documentor, error) {
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

	// Instantiate LLM agent
	da, err := llmagent.New(llmagent.Config{
		Name:        "documentor",
		Model:       model,
		Description: "Retrieves code from a GitHub repository and writes high-quality markdown documentation.",
		Instruction: buildInstruction(),
		Tools: []tool.Tool{
			fetchRepoTreeTool, // Fetch Git Repository files
			readRepoFileTool,
			writeOutputTool,
		},
		OutputKey: StateDocumentation,
	})
	if err != nil {
		return nil, err
	}

	return &Documentor{inner: da}, nil
}

// Agent returns the inner agent interface (higher abstraction may not be necessary but we will see)
func (d *Documentor) Agent() agent.Agent {
	return d.inner
}

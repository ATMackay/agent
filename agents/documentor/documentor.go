package documentor

import (
	"context"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
)

type Documentor struct {
	inner agent.Agent
}

// NewDocumentor returns a Documentor agent.
func NewDocumentor(ctx context.Context, cfg *Config, model model.LLM) (*Documentor, error) {
	// Configure documentor agent tools
	fetchRepoTreeTool, err := NewFetchRepoTreeTool(cfg)
	if err != nil {
		return nil, err
	}

	readRepoFileTool, err := NewReadRepoFileTool(cfg)
	if err != nil {
		return nil, err
	}

	writeOutputTool, err := NewWriteOutputTool(cfg)
	if err != nil {
		return nil, err
	}

	// Instantiate Documentor LLM agent
	da, err := llmagent.New(llmagent.Config{
		Name:        "documentor",
		Model:       model,
		Description: "Retrieves code from a GitHub repository and writes high-quality markdown documentation.",
		Instruction: buildInstruction(),
		Tools: []tool.Tool{
			fetchRepoTreeTool, // Fetch Git Repository files
			readRepoFileTool,  // Read files tool
			writeOutputTool,   // Write output to file tool
		},
		OutputKey: StateDocumentation,
	})
	if err != nil {
		return nil, err
	}

	return &Documentor{inner: da}, nil
}

// Agent returns the inner agent interface (higher abstraction may not be necessary but we will see).
func (d *Documentor) Agent() agent.Agent {
	return d.inner
}

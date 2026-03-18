package documentor

import (
	"context"

	"github.com/ATMackay/agent/tools"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
)

type Documentor struct {
	agent.Agent
}

// NewDocumentor returns a Documentor agent.
func NewDocumentor(ctx context.Context, cfg *Config, model model.LLM) (*Documentor, error) {
	// Configure documentor agent tools and dependencies.
	deps := tools.Deps{}
	deps.AddConfig(tools.FetchRepoTree, tools.FetchRepoTreeConfig{WorkDir: cfg.WorkDir})

	functionTools, err := tools.GetTools([]tools.Kind{
		tools.FetchRepoTree, // Fetch repository tree to understand the structure of the codebase.
		tools.ReadFile,      // Read specific files to understand code details and extract relevant information for documentation.
		tools.SearchRepo,    // Search the repository to find relevant code snippets or information.
		tools.WriteFile,     // Write documentation or other output files.
	}, &deps)
	if err != nil {
		return nil, err
	}

	// Instantiate Documentor LLM agent
	da, err := llmagent.New(llmagent.Config{
		Name:        "documentor",
		Model:       model,
		Description: "Retrieves code from a GitHub repository and writes high-quality markdown documentation.",
		Instruction: buildInstruction(),
		Tools:       functionTools,
		OutputKey:   tools.StateDocumentation,
	})
	if err != nil {
		return nil, err
	}

	return &Documentor{Agent: da}, nil
}

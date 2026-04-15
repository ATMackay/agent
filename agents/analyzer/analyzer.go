package analyzer

import (
	"context"

	"github.com/ATMackay/agent/tools"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
)

const AgentName = "analyzer"

// Analyzer is a general-purpose agent for filesystem and CLI tasks,
// with special focus on document analysis.
type Analyzer struct {
	agent.Agent
}

// NewAnalyzer returns an Analyzer agent wired with its full tool set.
func NewAnalyzer(ctx context.Context, cfg *Config, llm model.LLM) (*Analyzer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	deps := tools.Deps{}

	functionTools, err := tools.GetTools([]tools.Kind{
		tools.ListDir,       // Explore directory trees.
		tools.ReadLocalFile, // Read text files from the local filesystem.
		tools.WriteFile,     // Write output files.
		tools.EditFile,      // Make targeted edits to existing files.
		tools.ExecCommand,   // Run shell commands (build, extract, convert, etc.).
		tools.SearchFiles,   // Search for text patterns across local files.
	}, &deps)
	if err != nil {
		return nil, err
	}

	ag, err := llmagent.New(llmagent.Config{
		Name:        AgentName,
		Model:       llm,
		Description: "Performs filesystem and command-line tasks with special focus on document analysis.",
		Instruction: buildInstruction(),
		Tools:       functionTools,
	})
	if err != nil {
		return nil, err
	}

	return &Analyzer{Agent: ag}, nil
}

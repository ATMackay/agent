package documentor

import (
	"context"
	"log"

	"google.golang.org/genai"

	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
)

type Documentor struct {
	// TODO

}

func NewDocumentorAgent(ctx context.Context) (*Documentor, error) {
	model, err := gemini.NewModel(ctx, "gemini-2.5-flash", &genai.ClientConfig{})
	if err != nil {
		log.Fatalf("failed to create model: %s", err)
	}

	// Copied from ADK examples/workflows

	// --- 1. Define Sub-Agents for Each Pipeline Stage ---

	// Code Writer Agent
	// Takes the initial specification (from user query) and writes code.
	_, err = llmagent.New(llmagent.Config{
		Name:  "CodeWriterAgent",
		Model: model,
		Instruction: `You are a Python Code Generator.
Based *only* on the user's request, write Python code that fulfills the requirement.
Output *only* the complete Python code block, enclosed in triple backticks ('''python ... ''').
Do not add any other text before or after the code block.`,
		Description: "Writes initial Python code based on a specification.",
		OutputKey:   "generated_code", // Stores output in state["generated_code"]
	})
	if err != nil {
		log.Fatalf("failed to create codeWriterAgent: %s", err)
	}

	return &Documentor{}, nil
}

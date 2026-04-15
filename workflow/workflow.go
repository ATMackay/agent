package workflow

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

type Workflow struct {
	name    string
	runner  Runner
	session session.Service
	// state
	initialState map[string]any
}

// New creates a new workflow service.
func New(ctx context.Context, appName string, sessSrv session.Service, ag agent.Agent, initialState map[string]any) (*Workflow, error) {
	// Create runner
	r, err := runner.New(runner.Config{
		AppName:        appName,
		Agent:          ag,
		SessionService: sessSrv,
	})
	if err != nil {
		return nil, fmt.Errorf("create runner: %w", err)
	}
	return &Workflow{
		name:         appName,
		runner:       r,
		session:      sessSrv,
		initialState: initialState,
	}, nil
}

// Start triggers a new agent workflow.
func (s *Workflow) Start(ctx context.Context, userID string, usrMsg *genai.Content) error {
	// Create new session
	resp, err := s.session.Create(ctx, &session.CreateRequest{
		AppName: s.name,
		UserID:  userID,
		State:   s.initialState,
	})
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	slog.Info(
		"running agent",
		"agent_name", s.name,
		"session_id", resp.Session.ID(),
	)

	start := time.Now()
	for event, err := range s.runner.Run(ctx, userID, resp.Session.ID(), usrMsg, agent.RunConfig{}) {
		if err != nil {
			return fmt.Errorf("agent error: %w", err)
		}
		// handle event (log)
		if event.UsageMetadata == nil {
			continue
		}
		slog.Info("tokens_used",
			"event_id", event.ID,
			"author", event.Author,
			"total_tokens", event.UsageMetadata.TotalTokenCount,
			"prompt_tokens", event.UsageMetadata.PromptTokenCount,
			"tool_use_token_count", event.UsageMetadata.ToolUsePromptTokenCount,
			"thought_token_count", event.UsageMetadata.ThoughtsTokenCount,
		)
		if event.Content == nil {
			continue
		}
		for _, p := range event.Content.Parts {
			slog.Debug("response_content",
				"event_id", event.ID,
				"role", event.Content.Role,
				"text", p.Text,
				"function_call", p.FunctionCall,
				"function_response", p.FunctionResponse,
			)
		}
	}
	slog.Info("Agent execution complete", "time_taken", time.Since(start))
	return nil
}

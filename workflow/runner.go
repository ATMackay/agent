package workflow

import (
	"context"
	"iter"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

type Runner interface {
	Run(ctx context.Context, userID string, sessionID string, msg *genai.Content, cfg agent.RunConfig) iter.Seq2[*session.Event, error]
}
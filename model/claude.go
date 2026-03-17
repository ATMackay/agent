package model

import (
	"context"
	"fmt"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	anthropicadk "github.com/louislef299/claude-go-adk"
	"google.golang.org/adk/model"
)

func newClaude(ctx context.Context, cfg *Config) (model.LLM, error) {
	if cfg.apiKey == "" {
		return nil, fmt.Errorf("anthropic api key is required for claude")
	}
	if cfg.Model == "" {
		cfg.Model = string(anthropic.ModelClaudeSonnet4_20250514)
	}
	return anthropicadk.NewModel(cfg.Model, anthropicadk.AnthropicOption(option.WithAPIKey(cfg.apiKey))), nil
}

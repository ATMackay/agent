package model

import (
	"context"
	"fmt"

	"google.golang.org/adk/model"
)

type Provider string

const (
	ProviderGemini Provider = "gemini"
	ProviderClaude Provider = "claude"
	// TODO support more
)

// Config is the provider model config with API access key.
type Config struct {
	Provider Provider
	Model    string

	apiKey string
}

func (c *Config) WithAPIKey(apiKey string) *Config {
	c.apiKey = apiKey
	return c
}

func New(ctx context.Context, cfg *Config) (model.LLM, error) {
	switch cfg.Provider {
	case "", ProviderClaude: // Set Claude as default provider when supplied value is empty.
		return newClaude(ctx, cfg)
	case ProviderGemini:
		return newGemini(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported model provider: %s", cfg.Provider)
	}
}

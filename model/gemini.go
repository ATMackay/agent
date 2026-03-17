package model

import (
	"context"
	"fmt"

	"google.golang.org/adk/model"
	adkgemini "google.golang.org/adk/model/gemini"
	"google.golang.org/genai"
)

func newGemini(ctx context.Context, cfg *Config) (model.LLM, error) {
	if cfg.apiKey == "" {
		return nil, fmt.Errorf("google api key is required for gemini")
	}
	if cfg.Model == "" {
		cfg.Model = "gemini-2.5-pro"
	}

	return adkgemini.NewModel(ctx, cfg.Model, &genai.ClientConfig{
		APIKey: cfg.apiKey,
	})
}

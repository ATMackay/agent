package model

import (
	"context"
	"testing"
)

func Test_NewModel(t *testing.T) {
	t.Run("unsupported-provider", func(t *testing.T) {
		_, err := New(context.Background(), &Config{Provider: "not-supported"})
		if err == nil {
			t.Fatal("expected non-nil error")
		}
	})
	t.Run("empty-config-claude", func(t *testing.T) {
		_, err := New(context.Background(), &Config{})
		if err == nil {
			t.Fatal("expected non-nil error")
		}
	})
	t.Run("empty-config-gemini", func(t *testing.T) {
		_, err := New(context.Background(), &Config{Provider: ProviderGemini})
		if err == nil {
			t.Fatal("expected non-nil error")
		}
	})

	t.Run("happy-path-claude", func(t *testing.T) {
		cfg := &Config{}
		_, err := New(context.Background(), cfg.WithAPIKey("api-key"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	t.Run("happy-path-gemini", func(t *testing.T) {
		_, err := New(context.Background(), &Config{Provider: ProviderGemini, apiKey: "api-key"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

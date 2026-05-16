package agents

import (
	"context"
	"os"

	"github.com/tmc/langchaingo/llms"
)

// LLM defines the interface for interacting with language models.
type LLM interface {
	// GenerateContent sends a prompt to the LLM and returns the generated text.
	GenerateContent(ctx context.Context, contents []llms.MessageContent) (string, error)
}

// Config holds the configuration for the LLM client.
type ClientConfig struct {
	APIKey   string
	Model    string
	BaseURL  string
	Provider string // "openai", "openrouter", or "ollama"
}

// DetectProvider determines the LLM provider from environment variables.
func DetectProvider() string {
	provider := os.Getenv("LLM_PROVIDER")
	if provider != "" {
		return provider
	}
	// Default to openai if OPENAI_API_KEY is set
	if os.Getenv("OPENAI_API_KEY") != "" {
		return "openai"
	}
	// Default to ollama if no API key is set
	return "ollama"
}

// NewClient creates an LLM client based on the provider and configuration.
func NewClient(cfg ClientConfig) (LLM, error) {
	switch cfg.Provider {
	case "openai":
		return newOpenAIClient(cfg)
	case "openrouter":
		return newOpenAIClient(cfg) // OpenRouter uses same interface
	case "ollama":
		return newOllamaClient(cfg)
	default:
		return newOllamaClient(cfg)
	}
}

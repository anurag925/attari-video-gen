package agents

import (
	"context"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

// ollamaClient implements LLM using a local Ollama server.
type ollamaClient struct {
	llm   *ollama.LLM
	model string
}

func newOllamaClient(cfg ClientConfig) (LLM, error) {
	opts := []ollama.Option{ollama.WithModel(cfg.Model)}
	if cfg.BaseURL != "" {
		opts = append(opts, ollama.WithServerURL(cfg.BaseURL))
	} else {
		opts = append(opts, ollama.WithServerURL("http://localhost:11434"))
	}

	llm, err := ollama.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama client: %w", err)
	}

	return &ollamaClient{
		llm:   llm,
		model: cfg.Model,
	}, nil
}

func (c *ollamaClient) GenerateContent(ctx context.Context, contents []llms.MessageContent) (string, error) {
	resp, err := c.llm.GenerateContent(ctx, contents, llms.WithModel(c.model))
	if err != nil {
		return "", fmt.Errorf("Ollama generation failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return strings.TrimSpace(resp.Choices[0].Content), nil
}
package agents

import (
	"context"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// openAIClient implements LLM using the OpenAI-compatible API.
type openAIClient struct {
	llm   *openai.LLM
	model string
}

func newOpenAIClient(cfg ClientConfig) (LLM, error) {
	opts := []openai.Option{openai.WithToken(cfg.APIKey)}
	if cfg.BaseURL != "" {
		opts = append(opts, openai.WithBaseURL(cfg.BaseURL))
	}

	llm, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	return &openAIClient{
		llm:   llm,
		model: cfg.Model,
	}, nil
}

func (c *openAIClient) GenerateContent(ctx context.Context, contents []llms.MessageContent) (string, error) {
	resp, err := c.llm.GenerateContent(ctx, contents, llms.WithModel(c.model))
	if err != nil {
		return "", fmt.Errorf("OpenAI generation failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return strings.TrimSpace(resp.Choices[0].Content), nil
}

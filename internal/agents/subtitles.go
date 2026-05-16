package agents

import (
	"context"
	"fmt"
	"os"

	"github.com/tmc/langchaingo/llms"
)

const (
	wordsPerMinute = 130
)

// Config holds the configuration for video summary generation.
type Config struct {
	Text     string
	Duration int // Duration in seconds to fit the subtitles into
}

// GenerateVideoSummary creates a concise narration summary for a video using the provided LLM.
// The summary is designed to fit within the specified duration.
func GenerateVideoSummary(ctx context.Context, llm LLM, cfg Config) (string, error) {
	if cfg.Duration <= 0 {
		return "", fmt.Errorf("duration must be positive")
	}

	maxWords := wordLimitForDuration(cfg.Duration)

	systemPrompt := "You convert source text into concise narration for a short video. " +
		"Return only the narration that should appear on screen and be spoken in voice-over. " +
		"Do not add titles, labels, explanations, bullet points, markdown, or surrounding quotes. " +
		"Keep the wording concise, natural, and factually grounded in the source. " +
		"The spoken narration must fit within the requested duration, never exceed the requested maximum word count, and may be shorter if needed. " +
		"Use plain text with optional line breaks only when they improve subtitle readability."

	contents := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Create a short spoken summary for a %d second video. The narration must be %d words or fewer when spoken naturally. It is acceptable to be shorter, but never longer.\n\nSource text:\n%s", cfg.Duration, maxWords, cfg.Text)),
	}

	resp, err := llm.GenerateContent(ctx, contents)
	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	return resp, nil
}

func wordLimitForDuration(durationSeconds int) int {
	if durationSeconds <= 0 {
		return 1
	}

	limit := durationSeconds * wordsPerMinute / 60
	if limit < 1 {
		return 1
	}

	return limit
}

// ResolveLLMConfig returns the API key, model, and base URL for the configured LLM provider
// based on environment variables and defaults.
func ResolveLLMConfig() (apiKey, model, baseURL string, err error) {
	provider := DetectProvider()
	apiKey = os.Getenv("OPENAI_API_KEY")
	model = os.Getenv("OPENAI_MODEL")
	baseURL = os.Getenv("OPENAI_BASE_URL")

	if provider == "openrouter" {
		model = os.Getenv("OPENROUTER_MODEL")
		apiKey = os.Getenv("OPENROUTER_API_KEY")
		baseURL = os.Getenv("OPENROUTER_BASE_URL")
	}

	if provider == "ollama" {
		model = os.Getenv("OLLAMA_MODEL")
		baseURL = os.Getenv("OLLAMA_BASE_URL")
		// Ollama doesn't require an API key
		return apiKey, model, baseURL, nil
	}

	if apiKey == "" {
		return "", "", "", fmt.Errorf("missing API key for LLM provider")
	}

	if model == "" {
		return "", "", "", fmt.Errorf("missing model for LLM provider")
	}
	return apiKey, model, baseURL, nil
}

// NewLLMClient creates and configures an LLM client based on the current environment.
func NewLLMClient() (LLM, error) {
	provider := DetectProvider()

	var apiKey, model, baseURL string
	var err error

	if provider == "ollama" {
		model = os.Getenv("OLLAMA_MODEL")
		if model == "" {
			model = "llama3.2" // Default Ollama model
		}
		baseURL = os.Getenv("OLLAMA_BASE_URL")
	} else {
		apiKey, model, baseURL, err = ResolveLLMConfig()
		if err != nil {
			return nil, err
		}
	}

	return NewClient(ClientConfig{
		APIKey:   apiKey,
		Model:    model,
		BaseURL:  baseURL,
		Provider: provider,
	})
}

package agents

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

const (
	wordsPerMinute = 130
)

type Config struct {
	APIKey   string
	Model    string
	BaseURL  string
	Text     string
	Duration int // Duration in seconds to fit the subtitles into
}

func GenerateVideoSummary(ctx context.Context, cfg Config) (string, error) {
	opts := []openai.Option{openai.WithToken(cfg.APIKey)}
	if cfg.BaseURL != "" {
		opts = append(opts, openai.WithBaseURL(cfg.BaseURL))
	}

	llm, err := openai.New(opts...)
	if err != nil {
		return "", err
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

	resp, err := llm.GenerateContent(ctx, contents, llms.WithModel(cfg.Model))
	if err != nil {
		return "", err
	}

	text := strings.TrimSpace(resp.Choices[0].Content)

	return text, nil
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
	provider := os.Getenv("LLM_PROVIDER")
	apiKey = os.Getenv("OPENAI_API_KEY")
	model = os.Getenv("OPENAI_MODEL")
	baseURL = os.Getenv("OPENAI_BASE_URL")

	if provider == "openrouter" {
		model = os.Getenv("OPENROUTER_MODEL")
		apiKey = os.Getenv("OPENROUTER_API_KEY")
		baseURL = os.Getenv("OPENROUTER_BASE_URL")
	}

	if apiKey == "" {
		return "", "", "", fmt.Errorf("missing API key for LLM provider")
	}

	if model == "" {
		return "", "", "", fmt.Errorf("missing model for LLM provider")
	}
	return apiKey, model, baseURL, nil
}

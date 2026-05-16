package agents

import (
	"context"
	"fmt"
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

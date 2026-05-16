package agents

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

const (
	wordsPerMinute     = 130
	maxRewriteAttempts = 3
)

var wordPattern = regexp.MustCompile(`\S+`)
var sentenceBoundaryPattern = regexp.MustCompile(`(?m)([^.!?\n]+[.!?]?|[^\n]+$)`)

type Config struct {
	APIKey   string
	Model    string
	BaseURL  string
	Text     string
	Duration int // Duration in seconds to fit the subtitles into
}

func GenerateSubtitles(ctx context.Context, cfg Config) (string, error) {
	summary, err := GenerateVideoSummary(ctx, cfg)
	if err != nil {
		return "", err
	}

	return FormatAsSRT(summary, cfg.Duration), nil
}

func FormatAsSRT(text string, duration int) string {
	chunks := subtitleChunks(text, duration)
	if len(chunks) == 0 {
		return ""
	}

	cueDurations := cueDurations(chunks, duration)
	gap := cueGap(len(chunks), duration)
	current := 0 * time.Millisecond
	var builder strings.Builder

	for idx, chunk := range chunks {
		if idx > 0 {
			builder.WriteString("\n\n")
		}

		start := current
		end := current + cueDurations[idx]
		if end <= start {
			end = start + time.Second
		}

		builder.WriteString(strconv.Itoa(idx + 1))
		builder.WriteString("\n")
		builder.WriteString(formatSRTTimestamp(start))
		builder.WriteString(" --> ")
		builder.WriteString(formatSRTTimestamp(end))
		builder.WriteString("\n")
		builder.WriteString(chunk)

		current = end + gap
	}

	return builder.String()
}

func ExtractDialogueFromSRT(srt string) string {
	blocks := strings.Split(strings.ReplaceAll(strings.TrimSpace(srt), "\r\n", "\n"), "\n\n")
	var lines []string

	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		for _, line := range strings.Split(block, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || isSRTIndex(line) || isSRTTimestampLine(line) {
				continue
			}
			lines = append(lines, line)
		}
	}

	return strings.TrimSpace(strings.Join(lines, " "))
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

	for attempt := 0; attempt < maxRewriteAttempts; attempt++ {
		resp, err := llm.GenerateContent(ctx, contents, llms.WithModel(cfg.Model))
		if err != nil {
			return "", err
		}

		text := strings.TrimSpace(resp.Choices[0].Content)
		if countWords(text) <= maxWords {
			return text, nil
		}

		contents = append(contents,
			llms.TextParts(llms.ChatMessageTypeAI, text),
			llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("That draft is too long. Rewrite it so it stays at %d words or fewer, keeps the core message, and returns only the final narration.", maxWords)),
		)
	}

	return trimToWordLimit(strings.TrimSpace(cfg.Text), maxWords), nil
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

func countWords(text string) int {
	return len(wordPattern.FindAllString(text, -1))
}

func trimToWordLimit(text string, maxWords int) string {
	if maxWords <= 0 {
		return ""
	}

	words := wordPattern.FindAllString(text, -1)
	if len(words) <= maxWords {
		return strings.TrimSpace(text)
	}

	return strings.Join(words[:maxWords], " ")
}

func subtitleChunks(text string, duration int) []string {
	text = normalizeSubtitleText(text)
	if text == "" {
		return nil
	}

	maxWords := wordLimitForDuration(duration)
	trimmed := trimToWordLimit(text, maxWords)
	sentences := splitSentences(trimmed)
	if len(sentences) == 0 {
		return nil
	}

	chunkWordLimit := chunkWordLimit(duration, maxWords)
	var chunks []string
	var current []string
	currentWords := 0

	flush := func() {
		if len(current) == 0 {
			return
		}
		chunks = append(chunks, strings.Join(current, " "))
		current = nil
		currentWords = 0
	}

	for _, sentence := range sentences {
		sentenceWords := countWords(sentence)
		if sentenceWords == 0 {
			continue
		}

		if sentenceWords > chunkWordLimit {
			flush()
			chunks = append(chunks, splitLongSentence(sentence, chunkWordLimit)...)
			continue
		}

		if currentWords > 0 && currentWords+sentenceWords > chunkWordLimit {
			flush()
		}

		current = append(current, sentence)
		currentWords += sentenceWords
	}

	flush()
	if len(chunks) == 0 {
		return []string{trimmed}
	}

	return chunks
}

func normalizeSubtitleText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	return strings.Join(strings.Fields(text), " ")
}

func splitSentences(text string) []string {
	matches := sentenceBoundaryPattern.FindAllString(text, -1)
	if len(matches) == 0 {
		return []string{strings.TrimSpace(text)}
	}

	parts := make([]string, 0, len(matches))
	for _, match := range matches {
		match = strings.TrimSpace(match)
		if match != "" {
			parts = append(parts, match)
		}
	}

	if len(parts) == 0 {
		return []string{strings.TrimSpace(text)}
	}

	return parts
}

func splitLongSentence(sentence string, maxWords int) []string {
	if maxWords <= 0 {
		return nil
	}

	words := wordPattern.FindAllString(sentence, -1)
	if len(words) <= maxWords {
		return []string{sentence}
	}

	chunks := make([]string, 0, int(math.Ceil(float64(len(words))/float64(maxWords))))
	for start := 0; start < len(words); start += maxWords {
		end := start + maxWords
		if end > len(words) {
			end = len(words)
		}
		chunks = append(chunks, strings.Join(words[start:end], " "))
	}

	return chunks
}

func chunkWordLimit(duration int, maxWords int) int {
	if maxWords <= 0 {
		return 1
	}

	if duration <= 8 {
		return min(maxWords, 6)
	}
	if duration <= 20 {
		return min(maxWords, 10)
	}

	return min(maxWords, 14)
}

func cueDurations(chunks []string, totalDurationSeconds int) []time.Duration {
	total := time.Duration(max(totalDurationSeconds, len(chunks))) * time.Second
	weights := make([]int, len(chunks))
	weightSum := 0
	for idx, chunk := range chunks {
		weights[idx] = max(countWords(chunk), 1)
		weightSum += weights[idx]
	}

	remaining := total
	durations := make([]time.Duration, len(chunks))
	for idx := range chunks {
		remainingCues := len(chunks) - idx
		minRemaining := time.Duration(max(remainingCues-1, 0)) * time.Second
		budget := remaining - minRemaining
		share := time.Duration(float64(total) * float64(weights[idx]) / float64(weightSum))
		if share < time.Second {
			share = time.Second
		}
		if share > budget {
			share = budget
		}
		durations[idx] = share
		remaining -= share
		weightSum -= weights[idx]
	}

	return durations
}

func cueGap(cueCount int, totalDurationSeconds int) time.Duration {
	if cueCount <= 1 || totalDurationSeconds <= cueCount {
		return 0
	}

	return 150 * time.Millisecond
}

func formatSRTTimestamp(value time.Duration) string {
	if value < 0 {
		value = 0
	}

	hours := value / time.Hour
	value -= hours * time.Hour
	minutes := value / time.Minute
	value -= minutes * time.Minute
	seconds := value / time.Second
	value -= seconds * time.Second
	milliseconds := value / time.Millisecond

	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, seconds, milliseconds)
}

func isSRTIndex(line string) bool {
	if line == "" {
		return false
	}

	for _, char := range line {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}

func isSRTTimestampLine(line string) bool {
	return strings.Contains(line, " --> ")
}

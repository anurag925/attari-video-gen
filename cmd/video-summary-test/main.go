package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/anurag925/attari-video-gen/internal/agents"
	"github.com/joho/godotenv"
)

func main() {
	text := flag.String("text", "", "Source text to summarize")
	textFile := flag.String("file", "", "Path to a text file to summarize")
	duration := flag.Int("duration", 30, "Target summary duration in seconds")
	modelFlag := flag.String("model", "", "LLM model override")
	baseURLFlag := flag.String("base-url", "", "LLM base URL override")
	flag.Parse()

	_ = godotenv.Load()

	sourceText, err := resolveSourceText(*text, *textFile)
	if err != nil {
		log.Fatal(err)
	}

	apiKey, model, baseURL, err := resolveLLMConfig(*modelFlag, *baseURLFlag)
	if err != nil {
		log.Fatal(err)
	}

	summary, err := agents.GenerateVideoSummary(context.Background(), agents.Config{
		APIKey:   apiKey,
		Model:    model,
		BaseURL:  baseURL,
		Text:     sourceText,
		Duration: *duration,
	})
	if err != nil {
		log.Fatalf("generate video summary: %v", err)
	}

	fmt.Println(summary)
}

func resolveSourceText(text string, textFile string) (string, error) {
	if strings.TrimSpace(text) != "" {
		return text, nil
	}

	if strings.TrimSpace(textFile) == "" {
		return "", fmt.Errorf("provide either -text or -file")
	}

	content, err := os.ReadFile(textFile)
	if err != nil {
		return "", fmt.Errorf("read source text: %w", err)
	}

	resolved := strings.TrimSpace(string(content))
	if resolved == "" {
		return "", fmt.Errorf("source text is empty")
	}

	return resolved, nil
}

func resolveLLMConfig(modelFlag string, baseURLFlag string) (string, string, string, error) {
	provider := os.Getenv("LLM_PROVIDER")
	apiKey := os.Getenv("OPENAI_API_KEY")
	baseURL := strings.TrimSpace(baseURLFlag)

	if provider == "openrouter" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
		if baseURL == "" {
			baseURL = os.Getenv("OPENROUTER_BASE_URL")
			if baseURL == "" {
				baseURL = "https://openrouter.ai/api/v1"
			}
		}
	}

	if strings.TrimSpace(apiKey) == "" {
		return "", "", "", fmt.Errorf("missing API key for configured LLM provider")
	}

	model := strings.TrimSpace(modelFlag)
	if model == "" {
		model = os.Getenv("OPENAI_MODEL")
	}
	if model == "" {
		model = "gpt-3.5-turbo"
	}

	return apiKey, model, baseURL, nil
}

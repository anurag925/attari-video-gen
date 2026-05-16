package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/anurag925/attari-video-gen/internal/agents"
	"github.com/joho/godotenv"
)

func main() {
	text := flag.String("text", "", "Source text to summarize")
	textFile := flag.String("file", "", "Path to a text file to summarize")
	duration := flag.Int("duration", 30, "Target summary duration in seconds")
	flag.Parse()

	_ = godotenv.Load()

	sourceText, err := resolveSourceText(*text, *textFile)
	if err != nil {
		log.Fatal(err)
	}

	llm, err := agents.NewLLMClient()
	if err != nil {
		log.Fatalf("create LLM client: %v", err)
	}

	summary, err := agents.GenerateVideoSummary(context.Background(), llm, agents.Config{
		Text:     sourceText,
		Duration: *duration,
	})
	if err != nil {
		log.Fatalf("generate video summary: %v", err)
	}

	fmt.Println(summary)
}

func resolveSourceText(text string, textFile string) (string, error) {
	if text != "" {
		return text, nil
	}

	if textFile == "" {
		return "", fmt.Errorf("provide either -text or -file")
	}

	content, err := os.ReadFile(textFile)
	if err != nil {
		return "", fmt.Errorf("read source text: %w", err)
	}

	return string(content), nil
}
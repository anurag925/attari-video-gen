package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/anurag925/attari-video-gen/internal/tts"
	"github.com/joho/godotenv"
)

func main() {
	providerFlag := flag.String("provider", firstEnv("TTS_PROVIDER", tts.ProviderElevenLabs), "TTS provider: elevenlabs or kokoro")
	textFlag := flag.String("text", "Hello world from attari-video-gen", "Text to synthesize")
	textFileFlag := flag.String("file", "", "Path to a text file to synthesize")
	outputFlag := flag.String("output", "", "Output mp3 path; defaults to ./tmp/tts-test/audio.mp3")
	apiKeyFlag := flag.String("api-key", "", "TTS API key override")
	baseURLFlag := flag.String("base-url", "", "TTS base URL override")
	modelFlag := flag.String("model", "", "TTS model override")
	voiceFlag := flag.String("voice", "", "TTS voice override")
	formatFlag := flag.String("format", "", "TTS response format override")
	speedFlag := flag.Float64("speed", 0, "TTS speed override")
	flag.Parse()

	_ = godotenv.Load()

	text, err := resolveSourceText(*textFlag, *textFileFlag)
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := resolveTTSConfig(*providerFlag, *apiKeyFlag, *baseURLFlag, *modelFlag, *voiceFlag, *formatFlag, *speedFlag)
	if err != nil {
		log.Fatal(err)
	}

	outputPath := strings.TrimSpace(*outputFlag)
	if outputPath == "" {
		outputPath = filepath.Join("tmp", "tts-test", "audio.mp3")
	}

	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("create output directory: %v", err)
	}

	generatedPath, err := tts.GenerateToFile(context.Background(), cfg, text, outputDir)
	if err != nil {
		log.Fatalf("generate tts audio: %v", err)
	}

	if generatedPath != outputPath {
		if err := os.Rename(generatedPath, outputPath); err != nil {
			log.Fatalf("move generated audio to requested output path: %v", err)
		}
		generatedPath = outputPath
	}

	info, err := os.Stat(generatedPath)
	if err != nil {
		log.Fatalf("stat generated audio: %v", err)
	}

	fmt.Printf("provider=%s\n", cfg.Provider)
	fmt.Printf("output=%s\n", generatedPath)
	fmt.Printf("bytes=%d\n", info.Size())
	if cfg.BaseURL != "" {
		fmt.Printf("base_url=%s\n", cfg.BaseURL)
	}
	if cfg.Model != "" {
		fmt.Printf("model=%s\n", cfg.Model)
	}
	if cfg.VoiceID != "" {
		fmt.Printf("voice=%s\n", cfg.VoiceID)
	}
}

func resolveSourceText(text string, textFile string) (string, error) {
	if strings.TrimSpace(textFile) != "" {
		content, err := os.ReadFile(textFile)
		if err != nil {
			return "", fmt.Errorf("read text file: %w", err)
		}

		resolved := strings.TrimSpace(string(content))
		if resolved == "" {
			return "", fmt.Errorf("text file is empty")
		}

		return resolved, nil
	}

	resolved := strings.TrimSpace(text)
	if resolved == "" {
		return "", fmt.Errorf("provide non-empty -text or -file")
	}

	return resolved, nil
}

func resolveTTSConfig(provider string, apiKey string, baseURL string, model string, voice string, responseFormat string, speed float64) (tts.Config, error) {
	resolvedProvider := strings.ToLower(strings.TrimSpace(provider))
	if resolvedProvider == "" {
		resolvedProvider = tts.ProviderElevenLabs
	}

	cfg := tts.Config{
		Provider:       resolvedProvider,
		APIKey:         strings.TrimSpace(apiKey),
		BaseURL:        strings.TrimSpace(baseURL),
		Model:          strings.TrimSpace(model),
		VoiceID:        strings.TrimSpace(voice),
		ResponseFormat: strings.TrimSpace(responseFormat),
		Speed:          speed,
	}

	switch resolvedProvider {
	case tts.ProviderElevenLabs:
		if cfg.APIKey == "" {
			cfg.APIKey = strings.TrimSpace(os.Getenv("ELEVENLABS_API_KEY"))
		}
	case tts.ProviderKokoro:
		if cfg.APIKey == "" {
			cfg.APIKey = firstEnv("TTS_API_KEY", "OPENAI_API_KEY")
		}
		if cfg.BaseURL == "" {
			cfg.BaseURL = firstEnv("TTS_BASE_URL", "OPENAI_BASE_URL")
		}
		if cfg.Model == "" {
			cfg.Model = firstEnv("TTS_MODEL", "OPENAI_TTS_MODEL")
		}
		if cfg.VoiceID == "" {
			cfg.VoiceID = firstEnv("TTS_VOICE", "OPENAI_TTS_VOICE")
		}
		if cfg.ResponseFormat == "" {
			cfg.ResponseFormat = firstEnv("TTS_RESPONSE_FORMAT", "OPENAI_TTS_RESPONSE_FORMAT")
		}
		if cfg.Speed <= 0 {
			cfg.Speed = firstEnvFloat("TTS_SPEED", "OPENAI_TTS_SPEED")
		}
	default:
		return tts.Config{}, fmt.Errorf("unsupported provider %q", provider)
	}

	return cfg, nil
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}

	return ""
}

func firstEnvFloat(keys ...string) float64 {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			continue
		}

		parsed, err := strconv.ParseFloat(value, 64)
		if err == nil {
			return parsed
		}
	}

	return 0
}

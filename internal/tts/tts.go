package tts

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	ProviderElevenLabs = "elevenlabs"
	ProviderKokoro     = "kokoro"
)

type Provider interface {
	Generate(ctx context.Context, text string) (io.Reader, error)
}

type Config struct {
	Provider       string
	APIKey         string
	BaseURL        string
	VoiceID        string
	Model          string
	ResponseFormat string
	Speed          float64
	HTTPClient     *http.Client
}

func New(cfg Config) (Provider, error) {
	switch providerName(cfg.Provider) {
	case ProviderElevenLabs:
		return newElevenLabsProvider(cfg)
	case ProviderKokoro:
		return newKokoroProvider(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported tts provider: %s", cfg.Provider)
	}
}

func providerName(name string) string {
	if strings.TrimSpace(name) == "" {
		return ProviderElevenLabs
	}

	return strings.ToLower(strings.TrimSpace(name))
}

func Generate(ctx context.Context, cfg Config, text string) (io.Reader, error) {
	provider, err := New(cfg)
	if err != nil {
		return nil, err
	}

	return provider.Generate(ctx, text)
}

// GenerateToFile generates TTS audio from text and saves it to outputDir/audio.mp3.
func GenerateToFile(ctx context.Context, cfg Config, text, outputDir string) (string, error) {
	audioReader, err := Generate(ctx, cfg, text)
	if err != nil {
		return "", err
	}

	audioPath := filepath.Join(outputDir, "audio.mp3")
	outFile, err := os.Create(audioPath)
	if err != nil {
		return "", err
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, audioReader); err != nil {
		return "", err
	}

	return audioPath, nil
}

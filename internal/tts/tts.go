package tts

import (
	"context"
	"io"
	"log/slog"

	"github.com/plexusone/elevenlabs-go"
)

type Config struct {
	APIKey  string
	VoiceID string
	Model   string
}

func Generate(ctx context.Context, cfg Config, text string) (io.Reader, error) {
	client, err := elevenlabs.NewClient(elevenlabs.WithAPIKey(cfg.APIKey))
	if err != nil {
		return nil, err
	}

	slog.Info("Using API key", "key", cfg.APIKey)

	voiceID := cfg.VoiceID
	if voiceID == "" {
		voiceID = "JBFqnCBsd6RMkjVDRZzb" // Default voice
	}

	modelID := cfg.Model
	if modelID == "" {
		modelID = elevenlabs.DefaultModelID
	}

	slog.Info("Using voice ID", "voice_id", voiceID)
	slog.Info("Using model ID", "model_id", modelID)

	resp, err := client.TextToSpeech().Generate(ctx, &elevenlabs.TTSRequest{
		Text:    text,
		VoiceID: voiceID,
		ModelID: modelID,
	})
	if err != nil {
		return nil, err
	}

	return resp.Audio, nil
}

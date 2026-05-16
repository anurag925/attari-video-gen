package tts

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/plexusone/elevenlabs-go"
)

type elevenLabsProvider struct {
	client  *elevenlabs.Client
	voiceID string
	modelID string
}

func newElevenLabsProvider(cfg Config) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("ELEVENLABS_API_KEY not set")
	}

	client, err := elevenlabs.NewClient(elevenlabs.WithAPIKey(cfg.APIKey))
	if err != nil {
		return nil, err
	}

	voiceID := cfg.VoiceID
	if voiceID == "" {
		voiceID = "JBFqnCBsd6RMkjVDRZzb"
	}

	modelID := cfg.Model
	if modelID == "" {
		modelID = elevenlabs.DefaultModelID
	}

	return &elevenLabsProvider{
		client:  client,
		voiceID: voiceID,
		modelID: modelID,
	}, nil
}

func (p *elevenLabsProvider) Generate(ctx context.Context, text string) (io.Reader, error) {
	slog.Info("Using voice ID", "voice_id", p.voiceID)
	slog.Info("Using model ID", "model_id", p.modelID)

	resp, err := p.client.TextToSpeech().Generate(ctx, &elevenlabs.TTSRequest{
		Text:    text,
		VoiceID: p.voiceID,
		ModelID: p.modelID,
	})
	if err != nil {
		return nil, err
	}

	return resp.Audio, nil
}

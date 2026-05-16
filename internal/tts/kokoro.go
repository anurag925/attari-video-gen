package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

const defaultKokoroBaseURL = "http://localhost:8880/v1"

type kokoroProvider struct {
	apiKey         string
	baseURL        string
	voiceID        string
	model          string
	responseFormat string
	speed          float64
	httpClient     *http.Client
}

func newKokoroProvider(cfg Config) Provider {
	resolved := cfg
	if resolved.BaseURL == "" {
		resolved.BaseURL = defaultKokoroBaseURL
	}
	if resolved.Model == "" {
		resolved.Model = "kokoro"
	}
	if resolved.VoiceID == "" {
		resolved.VoiceID = "af_bella"
	}
	if resolved.ResponseFormat == "" {
		resolved.ResponseFormat = "mp3"
	}
	if resolved.Speed <= 0 {
		resolved.Speed = 1.0
	}
	if resolved.HTTPClient == nil {
		resolved.HTTPClient = http.DefaultClient
	}

	return &kokoroProvider{
		apiKey:         resolved.APIKey,
		baseURL:        resolved.BaseURL,
		voiceID:        resolved.VoiceID,
		model:          resolved.Model,
		responseFormat: resolved.ResponseFormat,
		speed:          resolved.Speed,
		httpClient:     resolved.HTTPClient,
	}
}

func (p *kokoroProvider) Generate(ctx context.Context, text string) (io.Reader, error) {
	payload := map[string]any{
		"model":           p.model,
		"input":           text,
		"voice":           p.voiceID,
		"response_format": p.responseFormat,
		"speed":           p.speed,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimRight(p.baseURL, "/") + "/audio/speech"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	slog.Info("Generating speech", "provider", ProviderKokoro, "base_url", p.baseURL, "voice_id", p.voiceID, "model_id", p.model)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("tts request failed with status %s: %s", resp.Status, strings.TrimSpace(string(audio)))
	}

	if len(audio) == 0 {
		return nil, fmt.Errorf("tts request returned empty audio")
	}

	return bytes.NewReader(audio), nil
}

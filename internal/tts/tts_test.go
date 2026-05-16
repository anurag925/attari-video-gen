package tts

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateUsesOpenAICompatibleSpeechEndpoint(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/audio/speech" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %s", got)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}

		if payload["model"] != "kokoro" {
			t.Fatalf("unexpected model: %#v", payload["model"])
		}
		if payload["input"] != "hello world" {
			t.Fatalf("unexpected input: %#v", payload["input"])
		}
		if payload["voice"] != "af_bella" {
			t.Fatalf("unexpected voice: %#v", payload["voice"])
		}
		if payload["response_format"] != "wav" {
			t.Fatalf("unexpected response format: %#v", payload["response_format"])
		}
		if payload["speed"] != 1.25 {
			t.Fatalf("unexpected speed: %#v", payload["speed"])
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("audio-bytes"))
	}))
	defer server.Close()

	reader, err := Generate(context.Background(), Config{
		Provider:       ProviderKokoro,
		APIKey:         "test-key",
		BaseURL:        server.URL,
		Model:          "kokoro",
		VoiceID:        "af_bella",
		ResponseFormat: "wav",
		Speed:          1.25,
	}, "hello world")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	audio, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read audio: %v", err)
	}

	if string(audio) != "audio-bytes" {
		t.Fatalf("unexpected audio body: %q", string(audio))
	}
}

func TestGenerateToFileWritesAudio(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("mp3-data"))
	}))
	defer server.Close()

	outputDir := t.TempDir()
	audioPath, err := GenerateToFile(context.Background(), Config{Provider: ProviderKokoro, BaseURL: server.URL}, "hello world", outputDir)
	if err != nil {
		t.Fatalf("GenerateToFile returned error: %v", err)
	}

	if audioPath != filepath.Join(outputDir, "audio.mp3") {
		t.Fatalf("unexpected audio path: %s", audioPath)
	}

	audio, err := os.ReadFile(audioPath)
	if err != nil {
		t.Fatalf("read audio file: %v", err)
	}

	if string(audio) != "mp3-data" {
		t.Fatalf("unexpected audio file contents: %q", string(audio))
	}
}

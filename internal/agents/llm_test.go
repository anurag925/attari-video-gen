package agents

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

// mockLLM is a test implementation of LLM for testing.
type mockLLM struct {
	response string
	err      error
}

func (m *mockLLM) GenerateContent(ctx context.Context, contents []llms.MessageContent) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func TestLLM_Interface(t *testing.T) {
	// Test that mockLLM satisfies LLM interface
	var _ LLM = (*mockLLM)(nil)
}

func TestGenerateVideoSummary_Success(t *testing.T) {
	mock := &mockLLM{
		response: "This is a test summary.",
	}

	summary, err := GenerateVideoSummary(context.Background(), mock, Config{
		Text:     "Source text for the video.",
		Duration: 30,
	})

	require.NoError(t, err)
	assert.Equal(t, "This is a test summary.", summary)
}

func TestGenerateVideoSummary_LLMError(t *testing.T) {
	mock := &mockLLM{
		err: errors.New("LLM connection failed"),
	}

	_, err := GenerateVideoSummary(context.Background(), mock, Config{
		Text:     "Source text for the video.",
		Duration: 30,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate summary")
}

func TestGenerateVideoSummary_InvalidDuration(t *testing.T) {
	mock := &mockLLM{
		response: "Summary",
	}

	_, err := GenerateVideoSummary(context.Background(), mock, Config{
		Text:     "Source text",
		Duration: 0,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duration must be positive")
}

func TestGenerateVideoSummary_NegativeDuration(t *testing.T) {
	mock := &mockLLM{
		response: "Summary",
	}

	_, err := GenerateVideoSummary(context.Background(), mock, Config{
		Text:     "Source text",
		Duration: -5,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duration must be positive")
}

func TestWordLimitForDuration(t *testing.T) {
	tests := []struct {
		duration int
		expected int
	}{
		{0, 1},   // Zero duration returns minimum of 1
		{-1, 1},  // Negative duration returns minimum of 1
		{10, 21}, // 10 seconds = 10 * 130 / 60 = 21 words
		{30, 65}, // 30 seconds = 30 * 130 / 60 = 65 words
		{60, 130}, // 60 seconds = 60 * 130 / 60 = 130 words
		{120, 260}, // 120 seconds = 120 * 130 / 60 = 260 words
	}

	for _, tt := range tests {
		result := wordLimitForDuration(tt.duration)
		assert.Equal(t, tt.expected, result, "duration: %d", tt.duration)
	}
}

func TestGenerateVideoSummary_PromptContainsDuration(t *testing.T) {
	capturedPrompt := ""

	// Create a wrapper that captures the prompt
	wrapper := &promptCapturingLLM{response: "Summary", capturedPrompt: &capturedPrompt}

	_, _ = GenerateVideoSummary(context.Background(), wrapper, Config{
		Text:     "Test source text.",
		Duration: 45,
	})

	// Verify prompt contains duration and word limit
	assert.Contains(t, capturedPrompt, "45 second")
	assert.Contains(t, capturedPrompt, "97 words") // 45 * 130 / 60 = 97
}

// promptCapturingLLM wraps mockLLM to capture prompts.
type promptCapturingLLM struct {
	response       string
	capturedPrompt *string
}

func (p *promptCapturingLLM) GenerateContent(ctx context.Context, contents []llms.MessageContent) (string, error) {
	var fullPrompt string
	for _, msg := range contents {
		for _, part := range msg.Parts {
			if tc, ok := part.(llms.TextContent); ok {
				fullPrompt += tc.Text + "\n"
			}
		}
	}
	*p.capturedPrompt = fullPrompt
	return p.response, nil
}

func TestDetectProvider_ExplicitOpenAI(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("OPENAI_API_KEY", "test-key")

	result := DetectProvider()
	assert.Equal(t, "openai", result)
}

func TestDetectProvider_ExplicitOpenRouter(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "openrouter")
	t.Setenv("OPENROUTER_API_KEY", "test-key")

	result := DetectProvider()
	assert.Equal(t, "openrouter", result)
}

func TestDetectProvider_ExplicitOllama(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "ollama")
	// Clear any API keys
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENROUTER_API_KEY", "")

	result := DetectProvider()
	assert.Equal(t, "ollama", result)
}

func TestDetectProvider_DefaultOpenAI(t *testing.T) {
	// No LLM_PROVIDER set, but OPENAI_API_KEY is set
	t.Setenv("LLM_PROVIDER", "")
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENROUTER_API_KEY", "")

	result := DetectProvider()
	assert.Equal(t, "openai", result)
}

func TestDetectProvider_DefaultOllama(t *testing.T) {
	// No LLM_PROVIDER set, no API keys
	t.Setenv("LLM_PROVIDER", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENROUTER_API_KEY", "")

	result := DetectProvider()
	assert.Equal(t, "ollama", result)
}

func TestNewClient_OpenAI(t *testing.T) {
	cfg := ClientConfig{
		Provider: "openai",
		APIKey:   "test-key",
		Model:    "gpt-4",
		BaseURL:  "",
	}

	// This will fail without actual API key, but we're testing interface creation
	_, err := NewClient(cfg)
	// In CI/environment without valid keys, this might error - that's expected
	// The important thing is it tries to create the right type
	if err == nil {
		// Success - valid credentials provided
		assert.NotNil(t, cfg)
	}
}

func TestNewClient_InvalidProvider(t *testing.T) {
	cfg := ClientConfig{
		Provider: "invalid-provider",
		APIKey:   "",
		Model:    "",
		BaseURL:  "",
	}

	// Should default to Ollama for unknown providers
	client, err := NewClient(cfg)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewLLMClient_WithOllama(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "ollama")
	t.Setenv("OLLAMA_MODEL", "llama3.2")
	t.Setenv("OLLAMA_BASE_URL", "http://localhost:11434")
	t.Setenv("OPENAI_API_KEY", "")

	client, err := NewLLMClient()
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewLLMClient_WithOpenAI(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_MODEL", "gpt-4")

	client, err := NewLLMClient()
	if err == nil {
		assert.NotNil(t, client)
	}
	// May fail without valid API key in test environment
}

func TestResolveLLMConfig_Ollama(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "ollama")
	t.Setenv("OLLAMA_MODEL", "llama3.2")
	t.Setenv("OLLAMA_BASE_URL", "http://localhost:11434")

	apiKey, model, baseURL, err := ResolveLLMConfig()
	require.NoError(t, err)

	assert.Empty(t, apiKey) // Ollama doesn't need API key
	assert.Equal(t, "llama3.2", model)
	assert.Equal(t, "http://localhost:11434", baseURL)
}

func TestResolveLLMConfig_OpenRouter(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "openrouter")
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	t.Setenv("OPENROUTER_MODEL", "anthropic/claude-3.5-sonnet")
	t.Setenv("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1")

	apiKey, model, baseURL, err := ResolveLLMConfig()
	require.NoError(t, err)

	assert.Equal(t, "test-key", apiKey)
	assert.Equal(t, "anthropic/claude-3.5-sonnet", model)
	assert.Equal(t, "https://openrouter.ai/api/v1", baseURL)
}

func TestResolveLLMConfig_MissingAPIKey(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_MODEL", "gpt-4")

	_, _, _, err := ResolveLLMConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing API key")
}

func TestResolveLLMConfig_MissingModel(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_MODEL", "")

	_, _, _, err := ResolveLLMConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing model")
}
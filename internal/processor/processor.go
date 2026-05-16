package processor

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/anurag925/attari-video-gen/internal/agents"
	"github.com/anurag925/attari-video-gen/internal/config"
	"github.com/anurag925/attari-video-gen/internal/download"
	"github.com/anurag925/attari-video-gen/internal/scraper"
	"github.com/anurag925/attari-video-gen/internal/state"
	"github.com/anurag925/attari-video-gen/internal/tts"
	"github.com/anurag925/attari-video-gen/internal/video"
)

// LLM is the interface for language model interactions.
type LLM = agents.LLM

// StepHandler processes a single pipeline step and returns the artifact path.
type StepHandler interface {
	Process(ctx context.Context, step config.Step, mgr *state.Manager, input *config.Input, workDir string, llm LLM) (string, error)
}

// Processor manages dynamic pipeline execution.
type Processor struct {
	mgr      *state.Manager
	input    *config.Input
	workDir  string
	llm      LLM
	handlers map[config.StepName]StepHandler
}

// New creates a new Processor.
func New(mgr *state.Manager, input *config.Input, workDir string, llm LLM) *Processor {
	return &Processor{
		mgr:      mgr,
		input:    input,
		workDir:  workDir,
		llm:      llm,
		handlers: make(map[config.StepName]StepHandler),
	}
}

// RegisterHandler registers a step handler for a given step name.
func (p *Processor) RegisterHandler(name config.StepName, h StepHandler) {
	p.handlers[name] = h
}

// RegisterDefaultHandlers registers all built-in step handlers.
func (p *Processor) RegisterDefaultHandlers() {
	p.RegisterHandler(config.StepSourceText, &sourceTextHandler{})
	p.RegisterHandler(config.StepDownload, &downloadHandler{})
	p.RegisterHandler(config.StepCut, &cutHandler{})
	p.RegisterHandler(config.StepSummarizedText, &summarizedTextHandler{})
	p.RegisterHandler(config.StepAudio, &audioHandler{})
	p.RegisterHandler(config.StepSrtSubtitles, &srtSubtitlesHandler{})
	p.RegisterHandler(config.StepSubtitles, &subtitlesHandler{})
	p.RegisterHandler(config.StepSubtitlesBurned, &subtitlesBurnedHandler{})
	p.RegisterHandler(config.StepMerge, &mergeHandler{})
}

// ProcessSteps iterates through the input steps and executes each one.
func (p *Processor) ProcessSteps(ctx context.Context) error {
	for _, step := range p.input.EnsureSteps() {
		if !step.Enabled {
			slog.Info("Skipping disabled step", "name", step.Name)
			continue
		}
		if skip, path := p.mgr.ShouldSkip(step.Name); skip {
			slog.Info("Skipping completed step", "name", step.Name, "path", path)
			continue
		}
		handler, ok := p.handlers[step.Name]
		if !ok {
			slog.Info("No handler registered for step", "name", step.Name)
			continue
		}
		slog.Info("Processing step", "name", step.Name)
		artifactPath, err := handler.Process(ctx, step, p.mgr, p.input, p.workDir, p.llm)
		if err != nil {
			return err
		}
		if err := p.mgr.CompleteStep(step.Name, artifactPath); err != nil {
			return err
		}
	}
	return nil
}

// --- Source Text Handler ---

type sourceTextHandler struct{}

func (h *sourceTextHandler) Process(ctx context.Context, step config.Step, mgr *state.Manager, input *config.Input, workDir string, _ LLM) (string, error) {
	text, err := scraper.GetText(scraper.Config{}, scraper.Input{
		URL:  input.URL,
		Text: input.Text,
	})
	if err != nil {
		return "", err
	}
	return state.WriteTextArtifact(workDir, input.OutputName, "source.txt", text)
}

// --- Download Handler ---

type downloadHandler struct{}

func (h *downloadHandler) Process(ctx context.Context, step config.Step, mgr *state.Manager, input *config.Input, workDir string, _ LLM) (string, error) {
	return download.Download(ctx, download.Config{OutputDir: workDir}, input.VideoURL)
}

// --- Cut Handler ---

type cutHandler struct{}

func (h *cutHandler) Process(ctx context.Context, step config.Step, mgr *state.Manager, input *config.Input, workDir string, _ LLM) (string, error) {
	downloadedPath := mgr.GetArtifact(config.StepDownload)
	if downloadedPath == "" {
		return "", nil
	}
	return video.Cut(ctx, video.Config{WorkingDir: workDir}, downloadedPath, input.Duration)
}

// --- Summarized Text Handler ---

type summarizedTextHandler struct{}

func (h *summarizedTextHandler) Process(ctx context.Context, step config.Step, mgr *state.Manager, input *config.Input, workDir string, llm LLM) (string, error) {
	sourceText, err := state.ReadTextFile(mgr.GetArtifact(config.StepSourceText))
	if err != nil {
		return "", err
	}
	summarized, err := agents.GenerateVideoSummary(ctx, llm, agents.Config{
		Text:     sourceText,
		Duration: input.Duration,
	})
	if err != nil {
		return "", err
	}
	return state.WriteTextArtifact(workDir, input.OutputName, "summarized.txt", summarized)
}

// --- Audio Handler ---

type audioHandler struct{}

func (h *audioHandler) Process(ctx context.Context, step config.Step, mgr *state.Manager, input *config.Input, workDir string, _ LLM) (string, error) {
	summarizedText, err := state.ReadTextFile(mgr.GetArtifact(config.StepSummarizedText))
	if err != nil {
		return "", err
	}
	cfg := ttsConfigFromEnv()
	return tts.GenerateToFile(ctx, cfg, summarizedText, workDir)
}

func ttsConfigFromEnv() tts.Config {
	provider := firstEnv("TTS_PROVIDER")
	cfg := tts.Config{
		Provider:       provider,
		BaseURL:        firstEnv("TTS_BASE_URL", "OPENAI_BASE_URL"),
		Model:          firstEnv("TTS_MODEL", "OPENAI_TTS_MODEL"),
		VoiceID:        firstEnv("TTS_VOICE", "OPENAI_TTS_VOICE"),
		ResponseFormat: firstEnv("TTS_RESPONSE_FORMAT", "OPENAI_TTS_RESPONSE_FORMAT"),
	}

	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", tts.ProviderElevenLabs:
		cfg.APIKey = os.Getenv("ELEVENLABS_API_KEY")
	case tts.ProviderKokoro:
		cfg.APIKey = firstEnv("TTS_API_KEY", "OPENAI_API_KEY")
	}

	return cfg
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

// --- SRT Subtitles Handler ---

type srtSubtitlesHandler struct{}

func (h *srtSubtitlesHandler) Process(ctx context.Context, step config.Step, mgr *state.Manager, input *config.Input, workDir string, _ LLM) (string, error) {
	srtPath, err := state.WriteTextArtifact(workDir, input.OutputName, "sub.srt", "")
	if err != nil {
		return "", err
	}
	audioPath := mgr.GetArtifact(config.StepAudio)
	if err := video.GenerateSubtitles(audioPath, srtPath); err != nil {
		return "", err
	}
	return srtPath, nil
}

// --- Subtitles (ASS) Handler ---

type subtitlesHandler struct{}

func (h *subtitlesHandler) Process(ctx context.Context, step config.Step, mgr *state.Manager, input *config.Input, workDir string, _ LLM) (string, error) {
	assPath, err := state.WriteTextArtifact(workDir, input.OutputName, "final.ass", "")
	if err != nil {
		return "", err
	}
	srtPath := mgr.GetArtifact(config.StepSrtSubtitles)
	if err := video.ConvertSRTToASS(srtPath, assPath); err != nil {
		return "", err
	}
	return assPath, nil
}

// --- Subtitles Burned Handler ---

type subtitlesBurnedHandler struct{}

func (h *subtitlesBurnedHandler) Process(ctx context.Context, step config.Step, mgr *state.Manager, input *config.Input, workDir string, _ LLM) (string, error) {
	if err := download.EnsureFFmpeg(); err != nil {
		return "", err
	}

	cutVideoPath := mgr.GetArtifact(config.StepCut)
	assPath := mgr.GetArtifact(config.StepSubtitles)
	subtitledOutputPath := state.ArtifactPath(workDir, state.BaseName(input.OutputName)+"-subtitled.mp4", mgr.GetArtifact(config.StepSubtitlesBurned), cutVideoPath)
	return video.AddSubtitles(ctx, video.Config{
		WorkingDir: workDir,
		OutputPath: subtitledOutputPath,
	}, cutVideoPath, assPath)
}

// --- Merge Handler ---

type mergeHandler struct{}

func (h *mergeHandler) Process(ctx context.Context, step config.Step, mgr *state.Manager, input *config.Input, workDir string, _ LLM) (string, error) {
	videoWithSubsPath := mgr.GetArtifact(config.StepSubtitlesBurned)
	if videoWithSubsPath == "" {
		return "", nil
	}
	audioPath := mgr.GetArtifact(config.StepAudio)
	if audioPath == "" {
		return "", nil
	}
	finalOutputPath := state.ArtifactPath(workDir, input.OutputName, mgr.GetArtifact(config.StepMerge), videoWithSubsPath)
	return video.MergeAudioVideo(ctx, video.Config{
		WorkingDir: workDir,
		OutputPath: finalOutputPath,
	}, videoWithSubsPath, audioPath, state.BaseName(input.OutputName))
}
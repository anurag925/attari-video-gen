package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/anurag925/attari-video-gen/internal/agents"
	"github.com/anurag925/attari-video-gen/internal/download"
	"github.com/anurag925/attari-video-gen/internal/scraper"
	"github.com/anurag925/attari-video-gen/internal/tts"
	"github.com/anurag925/attari-video-gen/internal/video"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Input struct {
	VideoURL   string `yaml:"video_url"`
	URL        string `yaml:"url"`  // Page URL to scrape for text
	Text       string `yaml:"text"` // Fallback: direct text input
	Duration   int    `yaml:"duration"`
	OutputName string `yaml:"output_name"`
}

type pipelineState struct {
	Signature string `json:"signature"`

	SourceTextPath     string `json:"source_text_path,omitempty"`
	SummarisedTextPath string `json:"summarized_text_path,omitempty"`
	SrtSubtitlesPath   string `json:"srt_subtitles_path,omitempty"`
	SubtitlesPath      string `json:"subtitles_path,omitempty"`
	DownloadedPath     string `json:"downloaded_path,omitempty"`
	CutVideoPath       string `json:"cut_video_path,omitempty"`
	AudioPath          string `json:"audio_path,omitempty"`
	VideoWithSubsPath  string `json:"video_with_subs_path,omitempty"`
	FinalPath          string `json:"final_path,omitempty"`

	SourceTextDone     bool `json:"source_text_done"`
	SummarisedTextDone bool `json:"summarized_text_done"`
	SrtSubtitlesDone   bool `json:"srt_subtitles_done"`
	SubtitlesDone      bool `json:"subtitles_done"`
	DownloadDone       bool `json:"download_done"`
	CutDone            bool `json:"cut_done"`
	AudioDone          bool `json:"audio_done"`
	SubtitlesBurned    bool `json:"subtitles_burned"`
	MergeDone          bool `json:"merge_done"`
}

var (
	flagInput    string
	flagTextOnly bool
)

func main() {
	flag.StringVar(&flagInput, "i", "", "Input YAML file path")
	flag.BoolVar(&flagTextOnly, "text-only", false, "Only output scraped/summarized text, skip video generation")
	flag.Parse()

	if flagInput == "" {
		slog.Info("Usage: video-maker -i <input.yaml>")
		flag.Usage()
		os.Exit(1)
	}

	// Load .env file if present
	if err := godotenv.Load(); err != nil {
		log.Fatal("error reading .env file, proceeding with environment variables only")
	}

	apiKey, model, baseURL, err := resolveLLMConfig()
	if err != nil {
		slog.Info("Error resolving LLM config", "error", err)
		os.Exit(1)
	}

	// Read and parse input file
	input, err := parseInputFile(flagInput)
	if err != nil {
		slog.Info("Error reading input file", "error", err)
		os.Exit(1)
	}

	if err := validateInput(input); err != nil {
		slog.Info("Invalid input", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Get working directory
	workDir, err := download.WorkingDir()
	if err != nil {
		slog.Info("Error getting working directory", "error", err)
		os.Exit(1)
	}

	slog.Info("Working directory", "path", workDir)

	statePath := pipelineStatePath(workDir, input.OutputName)
	runSignature := inputSignature(input)
	state, err := loadPipelineState(statePath)
	if err != nil {
		slog.Info("Error loading pipeline state", "error", err)
		os.Exit(1)
	}
	if state.Signature != "" && state.Signature != runSignature {
		slog.Info("Input changed, resetting saved pipeline state", "path", statePath)
		state = &pipelineState{}
	}
	state.Signature = runSignature

	// Scrape text from URL or use direct text
	var sourceText string
	if state.SourceTextDone && fileExists(state.SourceTextPath) {
		sourceText, err = readTextFile(state.SourceTextPath)
		if err != nil {
			slog.Info("Error reading saved source text", "error", err)
			os.Exit(1)
		}
		slog.Info("Skipping text fetch", "path", state.SourceTextPath)
	} else {
		sourceText, err = getText(ctx, input)
		if err != nil {
			slog.Info("Error getting text", "error", err)
			os.Exit(1)
		}

		sourceTextPath, err := writeTextArtifact(workDir, input.OutputName, "source.txt", sourceText)
		if err != nil {
			slog.Info("Error saving source text", "error", err)
			os.Exit(1)
		}

		state.SourceTextPath = sourceTextPath
		state.SourceTextDone = true
		if err := savePipelineState(statePath, state); err != nil {
			slog.Info("Error saving pipeline state", "error", err)
			os.Exit(1)
		}
	}

	// Download video
	var downloadedPath string
	if state.DownloadDone && fileExists(state.DownloadedPath) {
		downloadedPath = state.DownloadedPath
		slog.Info("Skipping video download", "path", downloadedPath)
	} else {
		slog.Info("Downloading video...")
		downloadedPath, err = download.Download(ctx, download.Config{OutputDir: workDir}, input.VideoURL)
		if err != nil {
			slog.Info("Error downloading video", "error", err)
			os.Exit(1)
		}
		state.DownloadedPath = downloadedPath
		state.DownloadDone = true
		if err := savePipelineState(statePath, state); err != nil {
			slog.Info("Error saving pipeline state", "error", err)
			os.Exit(1)
		}
	}
	slog.Info("Video downloaded", "path", downloadedPath)

	// Cut video to specified duration
	var cutVideoPath string
	if state.CutDone && fileExists(state.CutVideoPath) {
		cutVideoPath = state.CutVideoPath
		slog.Info("Skipping video cut", "path", cutVideoPath)
	} else {
		slog.Info("Cutting video", "duration_seconds", input.Duration)
		cutVideoPath, err = video.Cut(ctx, video.Config{WorkingDir: workDir}, downloadedPath, input.Duration)
		if err != nil {
			slog.Info("Error cutting video", "error", err)
			os.Exit(1)
		}
		state.CutVideoPath = cutVideoPath
		state.CutDone = true
		if err := savePipelineState(statePath, state); err != nil {
			slog.Info("Error saving pipeline state", "error", err)
			os.Exit(1)
		}
	}
	slog.Info("Video cut", "path", cutVideoPath)

	// Generate summarised text from source text
	var summarizedText string
	if state.SummarisedTextDone && fileExists(state.SummarisedTextPath) {
		summarizedText, err = readTextFile(state.SummarisedTextPath)
		if err != nil {
			slog.Info("Error reading saved summarized text", "error", err)
			os.Exit(1)
		}
		slog.Info("Skipping text summarization", "text_length", len(summarizedText))
	} else {
		summarizedText, err = agents.GenerateVideoSummary(ctx, agents.Config{
			APIKey:   apiKey,
			Model:    model,
			BaseURL:  baseURL,
			Text:     sourceText,
			Duration: input.Duration,
		})
		if err != nil {
			slog.Info("Error generating video summary", "error", err)
			os.Exit(1)
		}
		slog.Info("Video summary generated", "text_length", len(summarizedText))
		state.SummarisedTextDone = true

		summarizedTextPath, err := writeTextArtifact(workDir, input.OutputName, "summarized.txt", summarizedText)
		if err != nil {
			slog.Info("Error saving summarized text", "error", err)
			os.Exit(1)
		}

		state.SummarisedTextPath = summarizedTextPath
		if err := savePipelineState(statePath, state); err != nil {
			slog.Info("Error saving pipeline state", "error", err)
			os.Exit(1)
		}
	}

	slog.Info("Summarized text", "text", summarizedText)

	// Generate TTS audio
	var audioPath string
	if state.AudioDone && fileExists(state.AudioPath) {
		audioPath = state.AudioPath
		slog.Info("Skipping TTS generation", "path", audioPath)
	} else {
		slog.Info("Generating TTS audio...")
		audioPath, err = generateTTS(ctx, summarizedText, workDir)
		if err != nil {
			slog.Info("Error generating TTS", "error", err)
			os.Exit(1)
		}
		state.AudioPath = audioPath
		state.AudioDone = true
		if err := savePipelineState(statePath, state); err != nil {
			slog.Info("Error saving pipeline state", "error", err)
			os.Exit(1)
		}
	}

	slog.Info("TTS audio generated", "path", audioPath)

	// Check ffmpeg
	if err := download.EnsureFFmpeg(); err != nil {
		slog.Info("Error ensuring ffmpeg", "error", err)
		os.Exit(1)
	}

	// generate SRT subtitles from TTS audio
	var srtSubtitlesPath string
	if state.SrtSubtitlesDone && fileExists(state.SrtSubtitlesPath) {
		srtSubtitlesPath = state.SrtSubtitlesPath
		slog.Info("Skipping SRT subtitle generation", "path", srtSubtitlesPath)
	} else {
		srtSubtitlesPath, err = writeTextArtifact(workDir, input.OutputName, "sub.srt", "")
		if err != nil {
			slog.Info("Error creating placeholder SRT file", "error", err)
			os.Exit(1)
		}
		err = video.GenerateSubtitles(state.AudioPath, srtSubtitlesPath)
		if err != nil {
			slog.Info("Error generating SRT subtitles", "error", err)
			os.Exit(1)
		}

		state.SrtSubtitlesPath = srtSubtitlesPath
		state.SrtSubtitlesDone = true
		if err := savePipelineState(statePath, state); err != nil {
			slog.Info("Error saving pipeline state", "error", err)
			os.Exit(1)
		}
	}
	slog.Info("SRT subtitles ready", "path", srtSubtitlesPath)

	// Srt to ass subtitle
	var assSubtitlesPath string
	if state.SubtitlesDone && fileExists(state.SubtitlesPath) {
		assSubtitlesPath = state.SubtitlesPath
		slog.Info("Skipping SRT to ASS conversion", "path", assSubtitlesPath)
	} else {
		subtitlePath, err := writeTextArtifact(workDir, input.OutputName, "final.ass", "")
		if err != nil {
			slog.Info("Error creating placeholder ASS file", "error", err)
			os.Exit(1)
		}
		err = video.ConvertSRTToASS(state.SrtSubtitlesPath, subtitlePath)
		if err != nil {
			slog.Info("Error converting SRT to ASS", "error", err)
			os.Exit(1)
		}
		state.SubtitlesPath = subtitlePath
		state.SubtitlesDone = true
		if err := savePipelineState(statePath, state); err != nil {
			slog.Info("Error saving pipeline state", "error", err)
			os.Exit(1)
		}
	}

	// Burn subtitles into the cut video.
	var videoWithSubsPath string
	if state.SubtitlesBurned && fileExists(state.VideoWithSubsPath) {
		videoWithSubsPath = state.VideoWithSubsPath
		slog.Info("Skipping subtitle burn", "path", videoWithSubsPath)
	} else {
		subtitledOutputPath := artifactPathFromState(workDir, baseOutputName(input.OutputName)+"-subtitled.mp4", state.CutVideoPath, cutVideoPath)
		slog.Info("Burning subtitles into video", "video", cutVideoPath, "subtitles", state.SubtitlesPath)
		videoWithSubsPath, err = video.AddSubtitles(ctx, video.Config{
			WorkingDir: workDir,
			OutputPath: subtitledOutputPath,
		}, cutVideoPath, state.SubtitlesPath)
		if err != nil {
			slog.Info("Error burning subtitles", "error", err)
			os.Exit(1)
		}

		state.VideoWithSubsPath = videoWithSubsPath
		state.SubtitlesBurned = true
		if err := savePipelineState(statePath, state); err != nil {
			slog.Info("Error saving pipeline state", "error", err)
			os.Exit(1)
		}
	}
	if videoWithSubsPath == "" {
		videoWithSubsPath = state.VideoWithSubsPath
	}
	if videoWithSubsPath == "" {
		slog.Info("Missing subtitled video path after subtitle burn")
		os.Exit(1)
	}
	if state.VideoWithSubsPath != videoWithSubsPath {
		state.VideoWithSubsPath = videoWithSubsPath
		if err := savePipelineState(statePath, state); err != nil {
			slog.Info("Error saving pipeline state", "error", err)
			os.Exit(1)
		}
	}
	slog.Info("Subtitled video ready", "path", videoWithSubsPath)

	// Merge the subtitled video with the generated audio.
	var finalPath string
	if state.MergeDone && fileExists(state.FinalPath) {
		finalPath = state.FinalPath
		slog.Info("Skipping audio merge", "path", finalPath)
	} else {
		finalOutputPath := artifactPathFromState(workDir, input.OutputName, state.VideoWithSubsPath, videoWithSubsPath)
		slog.Info("Merging subtitled video with audio", "video", videoWithSubsPath, "audio", audioPath)
		finalPath, err = video.MergeAudioVideo(ctx, video.Config{
			WorkingDir: workDir,
			OutputPath: finalOutputPath,
		}, videoWithSubsPath, audioPath, baseOutputName(input.OutputName))
		if err != nil {
			slog.Info("Error merging audio and video", "error", err)
			os.Exit(1)
		}

		state.FinalPath = finalPath
		state.MergeDone = true
		if err := savePipelineState(statePath, state); err != nil {
			slog.Info("Error saving pipeline state", "error", err)
			os.Exit(1)
		}
	}
	if finalPath == "" {
		finalPath = state.FinalPath
	}
	slog.Info("Final video ready", "path", finalPath)
}

func parseInputFile(path string) (*Input, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var input Input
	if err := yaml.Unmarshal(data, &input); err != nil {
		return nil, err
	}

	return &input, nil
}

func validateInput(input *Input) error {
	if input.VideoURL == "" {
		return fmt.Errorf("video_url is required")
	}
	if input.URL == "" && input.Text == "" {
		return fmt.Errorf("either url or text is required")
	}
	if input.Duration <= 0 {
		return fmt.Errorf("duration must be positive")
	}
	if input.OutputName == "" {
		return fmt.Errorf("output_name is required")
	}
	return nil
}

func getText(ctx context.Context, input *Input) (string, error) {
	// If URL is provided, scrape it
	if input.URL != "" {
		slog.Info("Scraping text", "url", input.URL)
		text, err := scraper.Scrape(scraper.Config{}, input.URL)
		if err != nil {
			return "", fmt.Errorf("failed to scrape URL: %w", err)
		}
		slog.Info("Scraped text", "characters", len(text))
		return text, nil
	}

	// Otherwise use direct text
	return input.Text, nil
}

func generateTTS(ctx context.Context, text string, outputDir string) (string, error) {
	apiKey := os.Getenv("ELEVENLABS_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("ELEVENLABS_API_KEY not set")
	}

	slog.Info("check voice api key", "key", apiKey)

	audioReader, err := tts.Generate(ctx, tts.Config{APIKey: apiKey}, text)
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

func writeSubtitlesFile(outputDir string, outputName string, subtitles string) (string, error) {
	return writeTextArtifact(outputDir, outputName, "subtitles.srt", subtitles)
}

func writeTextArtifact(outputDir string, outputName string, suffix string, content string) (string, error) {
	path := filepath.Join(outputDir, baseOutputName(outputName)+"-"+suffix)
	if err := os.WriteFile(path, []byte(content+"\n"), 0644); err != nil {
		return "", err
	}

	return path, nil
}

func readTextFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

func baseOutputName(outputName string) string {
	baseName := filepath.Base(outputName)
	ext := filepath.Ext(baseName)
	name := strings.TrimSuffix(baseName, ext)
	if name == "" {
		return "output"
	}

	return name
}

func pipelineStatePath(outputDir string, outputName string) string {
	return filepath.Join(outputDir, baseOutputName(outputName)+"-progress.json")
}

func artifactPathFromState(fallbackDir string, fileName string, candidates ...string) string {
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}

		return filepath.Join(filepath.Dir(candidate), fileName)
	}

	return filepath.Join(fallbackDir, fileName)
}

func loadPipelineState(path string) (*pipelineState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &pipelineState{}, nil
		}
		return nil, err
	}

	var state pipelineState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

func savePipelineState(path string, state *pipelineState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func inputSignature(input *Input) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		input.VideoURL,
		input.URL,
		input.Text,
		fmt.Sprintf("%d", input.Duration),
		input.OutputName,
	}, "\n")))

	return hex.EncodeToString(sum[:])
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return !info.IsDir()
}

// resolveLLMConfig returns the API key, model, and base URL for the configured LLM provider based on environment variables and defaults.
func resolveLLMConfig() (string, string, string, error) {
	provider := os.Getenv("LLM_PROVIDER")
	apiKey := os.Getenv("OPENAI_API_KEY")
	model := os.Getenv("OPENAI_MODEL")
	baseURL := os.Getenv("OPENAI_BASE_URL")

	if provider == "openrouter" {
		model = os.Getenv("OPENROUTER_MODEL")
		apiKey = os.Getenv("OPENROUTER_API_KEY")
		baseURL = os.Getenv("OPENROUTER_BASE_URL")
	}

	if apiKey == "" {
		return "", "", "", fmt.Errorf("missing API key for LLM provider")
	}

	if model == "" {
		return "", "", "", fmt.Errorf("missing model for LLM provider")
	}
	return apiKey, model, baseURL, nil
}

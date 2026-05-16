package main

import (
	"context"
	"flag"
	"log"
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
	"github.com/joho/godotenv"
)

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

	// Create LLM client based on environment configuration
	llm, err := agents.NewLLMClient()
	if err != nil {
		slog.Info("Error creating LLM client", "error", err)
		os.Exit(1)
	}
	slog.Info("LLM client created", "provider", agents.DetectProvider())

	// Read and parse input file
	input, err := config.ParseInputFile(flagInput)
	if err != nil {
		slog.Info("Error reading input file", "error", err)
		os.Exit(1)
	}

	if err := config.ValidateInput(input); err != nil {
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

	// Initialize state manager
	signature := state.ComputeSignature(
		input.VideoURL,
		input.URL,
		input.Text,
		state.Itoa(input.Duration),
		input.OutputName,
	)
	statePath := state.StatePath(workDir, input.OutputName)
	mgr := state.NewManager(statePath)
	if err := mgr.LoadState(signature); err != nil {
		slog.Info("Error loading pipeline state", "error", err)
		os.Exit(1)
	}
	if mgr.State().Signature != "" && mgr.State().Signature != signature {
		slog.Info("Input changed, resetting saved pipeline state", "path", statePath)
		mgr.Reset(signature)
	}
	mgr.State().Signature = signature
	if err := mgr.Save(); err != nil {
		slog.Info("Error saving pipeline state", "error", err)
		os.Exit(1)
	}

	// Scrape text from URL or use direct text
	var sourceText string
	if skip, path := mgr.ShouldSkip("source_text"); skip {
		sourceText, err = state.ReadTextFile(path)
		if err != nil {
			slog.Info("Error reading saved source text", "error", err)
			os.Exit(1)
		}
		slog.Info("Skipping text fetch", "path", path)
	} else {
		sourceText, err = scraper.GetText(scraper.Config{}, scraper.Input{
			URL:  input.URL,
			Text: input.Text,
		})
		if err != nil {
			slog.Info("Error getting text", "error", err)
			os.Exit(1)
		}

		sourceTextPath, err := state.WriteTextArtifact(workDir, input.OutputName, "source.txt", sourceText)
		if err != nil {
			slog.Info("Error saving source text", "error", err)
			os.Exit(1)
		}

		if err := mgr.CompleteStep("source_text", sourceTextPath); err != nil {
			slog.Info("Error saving pipeline state", "error", err)
			os.Exit(1)
		}
	}

	// Download video
	var downloadedPath string
	if skip, path := mgr.ShouldSkip("download"); skip {
		downloadedPath = path
		slog.Info("Skipping video download", "path", downloadedPath)
	} else {
		slog.Info("Downloading video...")
		downloadedPath, err = download.Download(ctx, download.Config{OutputDir: workDir}, input.VideoURL)
		if err != nil {
			slog.Info("Error downloading video", "error", err)
			os.Exit(1)
		}
		if err := mgr.CompleteStep("download", downloadedPath); err != nil {
			slog.Info("Error saving pipeline state", "error", err)
			os.Exit(1)
		}
	}
	slog.Info("Video downloaded", "path", downloadedPath)

	// Cut video to specified duration
	var cutVideoPath string
	if skip, path := mgr.ShouldSkip("cut"); skip {
		cutVideoPath = path
		slog.Info("Skipping video cut", "path", cutVideoPath)
	} else {
		slog.Info("Cutting video", "duration_seconds", input.Duration)
		cutVideoPath, err = video.Cut(ctx, video.Config{WorkingDir: workDir}, downloadedPath, input.Duration)
		if err != nil {
			slog.Info("Error cutting video", "error", err)
			os.Exit(1)
		}
		if err := mgr.CompleteStep("cut", cutVideoPath); err != nil {
			slog.Info("Error saving pipeline state", "error", err)
			os.Exit(1)
		}
	}
	slog.Info("Video cut", "path", cutVideoPath)

	// Generate summarised text from source text
	var summarizedText string
	if skip, path := mgr.ShouldSkip("summarized_text"); skip {
		summarizedText, err = state.ReadTextFile(path)
		if err != nil {
			slog.Info("Error reading saved summarized text", "error", err)
			os.Exit(1)
		}
		slog.Info("Skipping text summarization", "text_length", len(summarizedText))
	} else {
		summarizedText, err = agents.GenerateVideoSummary(ctx, llm, agents.Config{
			Text:     sourceText,
			Duration: input.Duration,
		})
		if err != nil {
			slog.Info("Error generating video summary", "error", err)
			os.Exit(1)
		}
		slog.Info("Video summary generated", "text_length", len(summarizedText))

		summarizedTextPath, err := state.WriteTextArtifact(workDir, input.OutputName, "summarized.txt", summarizedText)
		if err != nil {
			slog.Info("Error saving summarized text", "error", err)
			os.Exit(1)
		}

		if err := mgr.CompleteStep("summarized_text", summarizedTextPath); err != nil {
			slog.Info("Error saving pipeline state", "error", err)
			os.Exit(1)
		}
	}

	slog.Info("Summarized text", "text", summarizedText)

	// Generate TTS audio
	var audioPath string
	if skip, path := mgr.ShouldSkip("audio"); skip {
		audioPath = path
		slog.Info("Skipping TTS generation", "path", audioPath)
	} else {
		slog.Info("Generating TTS audio...")
		audioPath, err = tts.GenerateToFile(ctx, ttsConfigFromEnv(), summarizedText, workDir)
		if err != nil {
			slog.Info("Error generating TTS", "error", err)
			os.Exit(1)
		}
		if err := mgr.CompleteStep("audio", audioPath); err != nil {
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
	if skip, path := mgr.ShouldSkip("srt_subtitles"); skip {
		srtSubtitlesPath = path
		slog.Info("Skipping SRT subtitle generation", "path", srtSubtitlesPath)
	} else {
		srtSubtitlesPath, err = state.WriteTextArtifact(workDir, input.OutputName, "sub.srt", "")
		if err != nil {
			slog.Info("Error creating placeholder SRT file", "error", err)
			os.Exit(1)
		}
		err = video.GenerateSubtitles(mgr.GetArtifact("audio"), srtSubtitlesPath)
		if err != nil {
			slog.Info("Error generating SRT subtitles", "error", err)
			os.Exit(1)
		}

		if err := mgr.CompleteStep("srt_subtitles", srtSubtitlesPath); err != nil {
			slog.Info("Error saving pipeline state", "error", err)
			os.Exit(1)
		}
	}
	slog.Info("SRT subtitles ready", "path", srtSubtitlesPath)

	// Srt to ass subtitle
	var assSubtitlesPath string
	if skip, path := mgr.ShouldSkip("subtitles"); skip {
		assSubtitlesPath = path
		slog.Info("Skipping SRT to ASS conversion", "path", assSubtitlesPath)
	} else {
		assSubtitlesPath, err = state.WriteTextArtifact(workDir, input.OutputName, "final.ass", "")
		if err != nil {
			slog.Info("Error creating placeholder ASS file", "error", err)
			os.Exit(1)
		}
		err = video.ConvertSRTToASS(mgr.GetArtifact("srt_subtitles"), assSubtitlesPath)
		if err != nil {
			slog.Info("Error converting SRT to ASS", "error", err)
			os.Exit(1)
		}
		if err := mgr.CompleteStep("subtitles", assSubtitlesPath); err != nil {
			slog.Info("Error saving pipeline state", "error", err)
			os.Exit(1)
		}
	}

	// Burn subtitles into the cut video.
	var videoWithSubsPath string
	if skip, path := mgr.ShouldSkip("subtitles_burned"); skip {
		videoWithSubsPath = path
		slog.Info("Skipping subtitle burn", "path", videoWithSubsPath)
	} else {
		subtitledOutputPath := state.ArtifactPath(workDir, state.BaseName(input.OutputName)+"-subtitled.mp4", mgr.GetArtifact("cut"), cutVideoPath)
		slog.Info("Burning subtitles into video", "video", cutVideoPath, "subtitles", mgr.GetArtifact("subtitles"))
		videoWithSubsPath, err = video.AddSubtitles(ctx, video.Config{
			WorkingDir: workDir,
			OutputPath: subtitledOutputPath,
		}, cutVideoPath, mgr.GetArtifact("subtitles"))
		if err != nil {
			slog.Info("Error burning subtitles", "error", err)
			os.Exit(1)
		}

		if err := mgr.CompleteStep("subtitles_burned", videoWithSubsPath); err != nil {
			slog.Info("Error saving pipeline state", "error", err)
			os.Exit(1)
		}
	}
	if videoWithSubsPath == "" {
		videoWithSubsPath = mgr.GetArtifact("subtitles_burned")
	}
	if videoWithSubsPath == "" {
		slog.Info("Missing subtitled video path after subtitle burn")
		os.Exit(1)
	}
	if mgr.GetArtifact("subtitles_burned") != videoWithSubsPath {
		mgr.State().SetArtifact("subtitles_burned", videoWithSubsPath)
		if err := mgr.Save(); err != nil {
			slog.Info("Error saving pipeline state", "error", err)
			os.Exit(1)
		}
	}
	slog.Info("Subtitled video ready", "path", videoWithSubsPath)

	// Merge the subtitled video with the generated audio.
	var finalPath string
	if skip, path := mgr.ShouldSkip("merge"); skip {
		finalPath = path
		slog.Info("Skipping audio merge", "path", finalPath)
	} else {
		finalOutputPath := state.ArtifactPath(workDir, input.OutputName, mgr.GetArtifact("subtitles_burned"), videoWithSubsPath)
		slog.Info("Merging subtitled video with audio", "video", videoWithSubsPath, "audio", audioPath)
		finalPath, err = video.MergeAudioVideo(ctx, video.Config{
			WorkingDir: workDir,
			OutputPath: finalOutputPath,
		}, videoWithSubsPath, audioPath, state.BaseName(input.OutputName))
		if err != nil {
			slog.Info("Error merging audio and video", "error", err)
			os.Exit(1)
		}

		if err := mgr.CompleteStep("merge", finalPath); err != nil {
			slog.Info("Error saving pipeline state", "error", err)
			os.Exit(1)
		}
	}
	if finalPath == "" {
		finalPath = mgr.GetArtifact("merge")
	}
	slog.Info("Final video ready", "path", finalPath)
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

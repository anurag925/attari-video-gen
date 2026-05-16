package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"

	"github.com/anurag925/attari-video-gen/internal/agents"
	"github.com/anurag925/attari-video-gen/internal/config"
	"github.com/anurag925/attari-video-gen/internal/download"
	"github.com/anurag925/attari-video-gen/internal/processor"
	"github.com/anurag925/attari-video-gen/internal/state"
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

	if err := godotenv.Load(); err != nil {
		log.Fatal("error reading .env file, proceeding with environment variables only")
	}

	llm, err := agents.NewLLMClient()
	if err != nil {
		slog.Info("Error creating LLM client", "error", err)
		os.Exit(1)
	}
	slog.Info("LLM client created", "provider", agents.DetectProvider())

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

	workDir, err := download.WorkingDir()
	if err != nil {
		slog.Info("Error getting working directory", "error", err)
		os.Exit(1)
	}
	slog.Info("Working directory", "path", workDir)

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

	proc := processor.New(mgr, input, workDir, llm)
	proc.RegisterDefaultHandlers()

	if err := proc.ProcessSteps(ctx); err != nil {
		slog.Info("Error processing steps", "error", err)
		os.Exit(1)
	}

	slog.Info("Pipeline completed", "output", mgr.GetArtifact("merge"))
}
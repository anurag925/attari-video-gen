package main

import (
	"context"
	"flag"
	"fmt"
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

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("error reading .env file, proceeding with environment variables only")
	}

	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		initCmd()
	case "generate", "run":
		generateCmd()
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("Usage: video-maker <command> [options]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  init       Generate an input YAML template")
	fmt.Println("  generate   Run the video generation pipeline")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  video-maker init -o input.yaml")
	fmt.Println("  video-maker generate -i input.yaml")
}

func generateCmd() {
	flag.CommandLine = flag.NewFlagSet("generate", flag.ExitOnError)
	flagInput := flag.CommandLine.String("i", "", "Input YAML file path")

	if err := flag.CommandLine.Parse(os.Args[2:]); err != nil {
		slog.Info("Error parsing flags", "error", err)
		os.Exit(1)
	}

	if *flagInput == "" {
		slog.Info("Usage: video-maker generate -i <input.yaml>")
		flag.Usage()
		os.Exit(1)
	}

	llm, err := agents.NewLLMClient()
	if err != nil {
		slog.Info("Error creating LLM client", "error", err)
		os.Exit(1)
	}
	slog.Info("LLM client created", "provider", agents.DetectProvider())

	input, err := config.ParseInputFile(*flagInput)
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

	slog.Info("Pipeline completed", "output", mgr.GetArtifact(config.StepMerge))
}

func initCmd() {
	flag.CommandLine = flag.NewFlagSet("init", flag.ExitOnError)
	flagOutput := flag.CommandLine.String("o", "", "Output YAML file path")

	if err := flag.CommandLine.Parse(os.Args[2:]); err != nil {
		slog.Info("Error parsing flags", "error", err)
		os.Exit(1)
	}

	if *flagOutput == "" {
		slog.Info("Usage: video-maker init -o <output.yaml>")
		flag.Usage()
		os.Exit(1)
	}

	data, err := config.DefaultInputTemplate()
	if err != nil {
		log.Fatal("Error generating template:", err)
	}

	if err := os.WriteFile(*flagOutput, data, 0644); err != nil {
		log.Fatal("Error writing file:", err)
	}

	slog.Info("Generated input template", "path", *flagOutput)
	fmt.Println("Created:", *flagOutput)
	fmt.Println("\nEdit the file and run:")
	fmt.Printf("  video-maker generate -i %s\n", *flagOutput)
}
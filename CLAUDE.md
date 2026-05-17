# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
# Build all packages
go build ./...

# Build CLI
go build ./cmd/video-maker

# Run CLI
go run ./cmd/video-maker --help
```

## Test Commands

```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./internal/state/...

# Run tests with verbose output
go test -v ./internal/state/...

# Run a single test
go test -v -run TestManager_ShouldSkip ./internal/state/...
```

## CLI Usage

The CLI uses subcommands:

```bash
# Generate input template
go run ./cmd/video-maker init -o input.yaml

# Run pipeline
go run ./cmd/video-maker generate -i input.yaml
```

## Architecture

### Pipeline System

The pipeline is **dynamic** - steps are defined in the input YAML and executed in order. Each step is **idempotent** and **stateful** (can resume after interruption).

Key components:
- `internal/config/config.go` - `StepName` enum (all 9 steps), `Input` struct, YAML parsing
- `internal/state/state.go` - `Manager` tracks step completion with `map[string]StepState`
- `internal/processor/processor.go` - `Processor` iterates steps, calls registered handlers

### Step Handler Pattern

All handlers implement `StepHandler` interface:
```go
type StepHandler interface {
    Process(ctx context.Context, step config.Step, mgr *state.Manager, input *config.Input, workDir string, llm LLM) (string, error)
}
```

Register handlers via `processor.RegisterHandler(config.StepName, handler)`.

### State Management

Pipeline state is persisted to `progress.json` in the work directory. Resume works by:
1. Loading existing input from `/tmp/attari-video-gen/inputs/input.<signature>.yaml`
2. Loading `progress.json` from the stored `WorkDir`
3. Skipping completed steps via `mgr.ShouldSkip(stepName)`

### Output Directory Structure

```
/tmp/attari-video-gen/
├── inputs/
│   └── input.<signature>.yaml   # Input config with _work_dir field
└── assets/
    └── <output-name>-<random>/  # Per-input working directory
        ├── progress.json        # Pipeline state for resume
        ├── source.txt
        ├── video.mp4
        └── ...
```

Final output is copied to user's current working directory on completion.

## Environment Setup

Required in `.env`:
```bash
# LLM (one of)
OPENAI_API_KEY=sk-...           # OpenAI
OPENROUTER_API_KEY=sk-or-...    # OpenRouter
# LLM_PROVIDER=ollama           # Local Ollama

# TTS (one of)
ELEVENLABS_API_KEY=...          # ElevenLabs
# TTS_PROVIDER=kokoro           # Local Kokoro
```

External tools required in PATH: `yt-dlp`, `ffmpeg`

## Key Patterns

1. **StepName enum** - Use constants like `config.StepSourceText` instead of strings
2. **Manager.GetArtifact(step)** - Returns artifact path for a completed step
3. **Manager.ShouldSkip(step)** - Returns (true, path) if step already done
4. **state.WriteTextArtifact()** - Writes text files to workDir with naming `<outputName>-<suffix>`
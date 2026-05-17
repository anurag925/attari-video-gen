# Attari Video Generator

A Go-based video generation pipeline that transforms web content (articles, Reddit posts) into narrated short-form videos with subtitles. It scrapes text, downloads source videos, generates AI summaries, converts text-to-speech, creates subtitles via Whisper, and burns subtitles into the final video output.

## Table of Contents

- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Pipeline Steps](#pipeline-steps)
- [Configuration](#configuration)
- [Environment Variables](#environment-variables)
- [API Server](#api-server)
- [CLI Tools](#cli-tools)
- [Input YAML Format](#input-yaml-format)
- [API Endpoints](#api-endpoints)
- [TTS Providers](#tts-providers)
- [LLM Providers](#llm-providers)
- [Testing Individual Components](#testing-individual-components)
- [Output Artifacts](#output-artifacts)
- [Development](#development)

## Architecture

The project follows a modular pipeline architecture with the following components:

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Scraper   │────▶│  LLM Agent   │────▶│  TTS Engine │
│  (content)  │     │  (summary)   │     │  (narration)│
└─────────────┘     └──────────────┘     └─────────────┘
       │                                       │
       ▼                                       ▼
┌─────────────┐                          ┌───────────┐
│  Download   │─────────────────────────▶│ Subtitles │
│  (yt-dlp)   │                          │ (Whisper) │
└─────────────┘                          └───────────┘
       │                                       │
       ▼                                       ▼
┌─────────────┐                          ┌───────────┐
│    Cut     │─────────────────────────▶│  Burn     │
│  (FFmpeg)  │                          │  Subs     │
└─────────────┘                          └───────────┘
                                                    │
                                                    ▼
                                             ┌───────────┐
                                             │   Merge   │
                                             │ (Audio+   │
                                             │  Video)   │
                                             └───────────┘
```

The pipeline is stateful — it tracks completed steps and can resume from where it left off if interrupted.

## Project Structure

```
attari-video-gen/
├── cmd/
│   ├── api/                    # REST API server entry point
│   │   └── main.go
│   ├── video-maker/            # CLI pipeline runner
│   │   └── main.go
│   └── integrations/           # Integration test utilities
│       ├── audio-merge-test/
│       ├── generate-subtitles-test/
│       ├── subtitles-test/
│       ├── tts-test/
│       └── video-summary-test/
│
├── internal/
│   ├── agents/                 # LLM integrations
│   │   ├── llm.go              # LLM interface & factory
│   │   ├── openai.go           # OpenAI / OpenRouter client
│   │   ├── ollama.go           # Local Ollama client
│   │   ├── subtitles.go        # Video summary generation
│   │   └── *_test.go
│   │
│   ├── config/                 # YAML config parsing & validation
│   │   └── config.go
│   │
│   ├── download/              # Video downloading via yt-dlp
│   │   └── download.go
│   │
│   ├── processor/             # Pipeline orchestrator
│   │   └── processor.go
│   │
│   ├── scraper/               # Web content extraction
│   │   └── scraper.go
│   │
│   ├── server/               # Echo REST API
│   │   ├── server.go
│   │   ├── server_extra.go
│   │   ├── handlers/         # HTTP handlers
│   │   │   ├── health.go
│   │   │   ├── pipelines.go
│   │   │   └── artifacts.go
│   │   ├── models/          # API data models
│   │   │   ├── artifact.go
│   │   │   ├── pipeline.go
│   │   │   └── common.go
│   │   ├── store/           # In-memory store
│   │   │   └── store.go
│   │   └── README.md
│   │
│   ├── state/                # Pipeline state persistence
│   │   ├── state.go
│   │   └── state_test.go
│   │
│   ├── tts/                  # Text-to-speech engines
│   │   ├── tts.go            # TTS interface & factory
│   │   ├── elevenlabs.go     # ElevenLabs provider
│   │   ├── kokoro.go         # Kokoro provider
│   │   └── tts_test.go
│   │
│   └── video/                # FFmpeg video processing
│       ├── video.go          # Video cutting & utilities
│       ├── audio.go          # Audio/video merging
│       ├── subtitles.go      # Subtitle burning
│       ├── convert_srt_to_ass.go
│       └── convert_text_to_srt.go
│
└── assets/
    ├── inputs/               # Example input YAML files
    │   └── input-example.yaml
    ├── shorts/               # Short sample assets
    └── shorts-outputs/       # Pipeline output directory
```

## Prerequisites

The following tools must be installed and available in your `PATH`:

| Tool | Purpose | Install |
|------|---------|---------|
| **yt-dlp** | Download YouTube videos | `brew install yt-dlp` |
| **FFmpeg** | Video/audio processing | `brew install ffmpeg` |

Optional (for local LLM):

| Tool | Purpose | Install |
|------|---------|---------|
| **Ollama** | Local LLM inference | [ollama.com](https://ollama.com) |
| **Kokoro** | Local TTS server | [kokoro-onnx](https://github.com/remsky/Kokoro-ONNX) |

## Installation

Clone the repository and install dependencies:

```bash
git clone https://github.com/anurag925/attari-video-gen.git
cd attari-video-gen
go mod download
```

## Quick Start

1. Create a `.env` file (see [Environment Variables](#environment-variables) for required keys)

2. Create an input YAML file:

```yaml
video_url: "https://www.youtube.com/watch?v=VIDEO_ID"
url: "https://example.com/article"
duration: 30
output_name: "my-video.mp4"
```

3. Run the pipeline:

```bash
go run ./cmd/video-maker -i assets/inputs/input-example.yaml
```

## Pipeline Steps

The video generation pipeline executes the following steps in order:

| Step | Description | Handler |
|------|-------------|---------|
| `source_text` | Scrapes text content from URL or uses direct text input | `sourceTextHandler` |
| `download` | Downloads source video via yt-dlp | `downloadHandler` |
| `cut` | Trims video to target duration using FFmpeg | `cutHandler` |
| `summarized_text` | Generates concise narration using LLM | `summarizedTextHandler` |
| `audio` | Converts summarized text to speech via TTS | `audioHandler` |
| `srt_subtitles` | Generates SRT subtitles from audio via Whisper | `srtSubtitlesHandler` |
| `subtitles` | Converts SRT to ASS format | `subtitlesHandler` |
| `subtitles_burned` | Burns ASS subtitles into video | `subtitlesBurnedHandler` |
| `merge` | Merges TTS audio with video | `mergeHandler` |

Each step is **idempotent** and **stateful** — if the pipeline is interrupted, it can resume from the last completed step on a subsequent run, provided the input configuration hasn't changed (detected via SHA-256 signature).

### Disabling Steps

Override steps in the input YAML to disable or reorder them:

```yaml
video_url: "https://www.youtube.com/watch?v=VIDEO_ID"
url: "https://example.com/article"
duration: 30
output_name: "my-video.mp4"
steps:
  - name: source_text
    enabled: true
  - name: download
    enabled: true
  - name: cut
    enabled: false
  - name: summarized_text
    enabled: false
  - name: audio
    enabled: false
  - name: srt_subtitles
    enabled: true
  - name: subtitles
    enabled: true
  - name: subtitles_burned
    enabled: true
  - name: merge
    enabled: true
```

## Configuration

### Input YAML Format

The pipeline reads configuration from a YAML file:

```yaml
video_url: "https://www.youtube.com/watch?v=VIDEO_ID"   # Required: Source video URL
url: "https://example.com/article"                      # Required (if no text): URL to scrape
text: "Direct text content here"                        # Required (if no url): Direct text input
duration: 30                                           # Required: Target duration in seconds
output_name: "my-video.mp4"                            # Required: Output filename
steps:                                                  # Optional: Override default steps
  - name: source_text
    enabled: true
```

**Validation rules:**
- `video_url` is required
- Either `url` or `text` is required
- `duration` must be positive
- `output_name` is required

## Environment Variables

### LLM Configuration

**OpenAI:**
```bash
OPENAI_API_KEY=sk-...
OPENAI_MODEL=gpt-4o-mini
OPENAI_BASE_URL=https://api.openai.com/v1
```

**OpenRouter:**
```bash
LLM_PROVIDER=openrouter
OPENROUTER_API_KEY=sk-or-...
OPENROUTER_MODEL=anthropic/claude-3-haiku
OPENROUTER_BASE_URL=https://openrouter.ai/api/v1
```

**Ollama (local):**
```bash
LLM_PROVIDER=ollama
OLLAMA_MODEL=llama3.2
OLLAMA_BASE_URL=http://localhost:11434
```

### TTS Configuration

**ElevenLabs:**
```bash
TTS_PROVIDER=elevenlabs
ELEVENLABS_API_KEY=...
TTS_VOICE=JBFqnCBsd6RMkjVDRZzb  # Optional: voice ID
```

**Kokoro (local):**
```bash
TTS_PROVIDER=kokoro
TTS_BASE_URL=http://localhost:8880/v1
TTS_MODEL=kokoro
TTS_VOICE=af_bella
```

The TTS layer also reads OpenAI-compatible env vars as fallbacks (`OPENAI_API_KEY`, `OPENAI_TTS_MODEL`, etc.).

## API Server

A REST API server built with Echo for programmatic access to pipelines and artifacts.

### Running the Server

```bash
go run ./cmd/api/main.go -port 8080 -host localhost
```

### Endpoints

#### Health
- `GET /health` — Health check

#### Artifacts
- `GET /api/v1/artifacts` — List all artifacts
- `GET /api/v1/artifacts/:name` — Get artifact details
- `GET /api/v1/artifacts/:name/download` — Download artifact file
- `DELETE /api/v1/artifacts/:name` — Delete artifact

#### Pipelines
- `GET /api/v1/pipelines` — List all pipelines
- `GET /api/v1/pipelines/:id` — Get pipeline details
- `POST /api/v1/pipelines` — Start a new pipeline
- `POST /api/v1/pipelines/:id/cancel` — Cancel a running pipeline
- `DELETE /api/v1/pipelines/:id` — Delete a pipeline

### API Examples

**Start a pipeline:**
```bash
curl -X POST http://localhost:8080/api/v1/pipelines \
  -H "Content-Type: application/json" \
  -d '{
    "video_url": "https://youtube.com/watch?v=...",
    "url": "https://example.com/article",
    "duration": 60,
    "output_name": "my-video.mp4"
  }'
```

**List pipelines:**
```bash
curl http://localhost:8080/api/v1/pipelines
```

**Get pipeline status:**
```bash
curl http://localhost:8080/api/v1/pipelines/:id
```

**List artifacts:**
```bash
curl http://localhost:8080/api/v1/artifacts
```

## CLI Tools

### video-maker

The CLI uses subcommands for different operations:

```bash
# Generate an input YAML template
go run ./cmd/video-maker init -o input.yaml

# Run the pipeline
go run ./cmd/video-maker generate -i input.yaml
```

Options:
- `init -o <path>` — Generate input YAML template
- `generate -i <path>` — Run the pipeline (required)
- `generate -copy-output=false` — Skip copying final output to current directory

### Integration Tests

Individual component test utilities in `cmd/integrations/`:

#### Video Summary Test
```bash
go run ./cmd/integrations/video-summary-test \
  -file assets/shorts/my-short-video-source.txt \
  -duration 30
```

#### TTS Test
```bash
# ElevenLabs
go run ./cmd/integrations/tts-test \
  -provider elevenlabs \
  -text "Hello from ElevenLabs" \
  -output tmp/tts-test/elevenlabs-audio.mp3

# Kokoro
go run ./cmd/integrations/tts-test \
  -provider kokoro \
  -base-url http://localhost:8880/v1 \
  -model kokoro \
  -voice af_bella \
  -text "Hello from Kokoro" \
  -output tmp/tts-test/kokoro-audio.mp3
```

#### Subtitles Test
```bash
go run ./cmd/integrations/subtitles-test \
  -video assets/shorts/video.mp4 \
  -subtitles assets/shorts/subtitles.ass \
  -output tmp/subtitles-test/subtitled.mp4 \
  -workdir tmp/subtitles-test
```

#### Generate Subtitles Test
```bash
go run ./cmd/integrations/generate-subtitles-test \
  -audio assets/shorts/audio.mp3 \
  -output /tmp/shorts-outputs-1/generated-1234-audio.srt
```

#### Audio Merge Test
```bash
go run ./cmd/integrations/audio-merge-test \
  -video assets/shorts/sample-video.mp4 \
  -audio assets/shorts-outputs-1/audio.mp3 \
  -output tmp/audio-merge-test/final.mp4
```

## TTS Providers

### ElevenLabs
- **Default voice**: `JBFqnCBsd6RMkjVDRZzb`
- **Default model**: ElevenLabs default
- Requires `ELEVENLABS_API_KEY`

### Kokoro (local)
- **Default base URL**: `http://localhost:8880/v1`
- **Default voice**: `af_bella`
- **Default model**: `kokoro`
- Requires a running Kokoro TTS server

## LLM Providers

### OpenAI
Uses the OpenAI Chat Completions API via `langchaingo`. Supports any OpenAI-compatible endpoint (Azure, proxies).

### OpenRouter
Uses the OpenAI-compatible OpenRouter API for access to various LLMs (Claude, Gemini, etc.).

### Ollama
Connects to a local Ollama server (default: `http://localhost:11434`). Does not require an API key. Falls back to `llama3.2` model if not specified.

## Testing Individual Components

Each integration test can be run independently to verify a single component works:

```bash
# Test LLM summary generation
go run ./cmd/integrations/video-summary-test -file <text-file> -duration 30

# Test TTS (ensure TTS server is running for Kokoro)
go run ./cmd/integrations/tts-test -provider elevenlabs -text "Test" -output /tmp/test.mp3

# Test subtitle generation
go run ./cmd/integrations/generate-subtitles-test -audio <audio-file> -output <srt-path>

# Test subtitle burning
go run ./cmd/integrations/subtitles-test -video <video> -subtitles <ass-file> -output <out>

# Test audio/video merge
go run ./cmd/integrations/audio-merge-test -video <video> -audio <audio> -output <out>
```

## Output Artifacts

Pipeline outputs are stored in `/tmp/attari-video-gen/` with each input having its own working directory:

```
/tmp/attari-video-gen/
├── inputs/
│   └── input.<signature>.yaml          # Input config (for repeat detection)
└── assets/
    └── <output-name>-<randomword>/     # Per-input working directory
        ├── progress.json              # Pipeline state (for resumption)
        ├── source.txt                 # Scraped source text
        ├── summarized.txt             # LLM-generated narration
        ├── video.mp4                  # Downloaded source video
        ├── video_cut.mp4              # Trimmed video
        ├── audio.mp3                  # TTS audio
        ├── sub.srt                    # SRT subtitles from Whisper
        ├── final.ass                  # ASS subtitles
        ├── <output>-subtitled.mp4     # Video with burned-in subtitles
        └── <output>.mp4               # Final output (merged with audio)
```

On pipeline completion, the final output is copied to the user's current working directory.

## Development

### Running Tests
```bash
go test ./...                    # Run all tests
go test -v -run TestName ./...   # Run specific test
```

### Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/labstack/echo/v4` | REST API framework |
| `github.com/tmc/langchaingo` | LLM client (OpenAI, Ollama) |
| `github.com/plexusone/elevenlabs-go` | ElevenLabs TTS |
| `github.com/joho/godotenv` | `.env` file loading |
| `github.com/PuerkitoBio/goquery` | HTML scraping |
| `gopkg.in/yaml.v3` | YAML config parsing |
| `github.com/stretchr/testify` | Testing assertions |

### Adding a New LLM Provider

1. Implement the `LLM` interface in `internal/agents/llm.go`:
   ```go
   type LLM interface {
       GenerateContent(ctx context.Context, contents []llms.MessageContent) (string, error)
   }
   ```

2. Add the provider factory in `NewClient()`:
   ```go
   case "yourprovider":
       return newYourProviderClient(cfg)
   ```

3. Add `YOUR_PROVIDER_*` env var handling in `ResolveLLMConfig()` and `DetectProvider()`.

### Adding a New TTS Provider

1. Implement the `Provider` interface in `internal/tts/tts.go`:
   ```go
   type Provider interface {
       Generate(ctx context.Context, text string) (io.Reader, error)
   }
   ```

2. Add the provider factory in `New()`:
   ```go
   case "yourprovider":
       return newYourProvider(cfg)
   ```

3. Set `ProviderKokoro` and `ProviderElevenLabs` as constants; new providers follow the same pattern.

### Adding a New Pipeline Step

1. Add a `StepName` constant in `internal/config/config.go`:
   ```go
   const StepYourNewStep StepName = "your_new_step"
   ```

2. Add it to `AllSteps()` in the correct position.

3. Create a handler struct and implement `StepHandler`:
   ```go
   type yourNewStepHandler struct{}
   func (h *yourNewStepHandler) Process(ctx context.Context, step config.Step, mgr *state.Manager, input *config.Input, workDir string, llm LLM) (string, error) {
       // Your logic here
   }
   ```

4. Register the handler in `Processor.RegisterDefaultHandlers()`:
   ```go
   p.RegisterHandler(config.StepYourNewStep, &yourNewStepHandler{})
   ```

### Roadmap / TODOs

See `internal/server/README.md` for API server TODOs:

- [ ] Implement pipeline execution (connect to existing processor)
- [ ] Add WebSocket support for real-time progress updates
- [ ] Add persistence (database) for store
- [ ] Add authentication
- [ ] Add pagination for list endpoints
- [ ] Add filtering for artifacts by type/pipeline
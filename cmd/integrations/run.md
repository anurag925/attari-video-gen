# Integration Runs

## Video Summary Test

```bash
go run ./cmd/integrations/video-summary-test \
	-file assets/shorts/my-short-video-source.txt \
	-duration 30
```

Requires LLM env vars such as `OPENAI_API_KEY` and `OPENAI_MODEL`, or the OpenRouter equivalents.

## TTS Test

ElevenLabs:

```bash
go run ./cmd/integrations/tts-test \
	-provider elevenlabs \
	-text "Hello from ElevenLabs" \
	-output tmp/tts-test/elevenlabs-audio.mp3
```

Kokoro:

```bash
go run ./cmd/integrations/tts-test \
	-provider kokoro \
	-base-url http://localhost:8880/v1 \
	-model kokoro \
	-voice af_bella \
	-text "Hello from Kokoro" \
	-output tmp/tts-test/kokoro-audio.mp3
```

## Generate Subtitles Test

```bash
go run ./cmd/integrations/generate-subtitles-test \
	-audio assets/shorts/audio.mp3 \
	-output /tmp/shorts-outputs-1/generated-1234-audio.srt
```

## Burn Subtitles Test

```bash
go run ./cmd/integrations/subtitles-test \
	-video assets/shorts/video.mp4 \
	-subtitles assets/shorts/subtitles.ass \
	-output tmp/subtitles-test/subtitled.mp4 \
	-workdir tmp/subtitles-test
```

Replace `assets/shorts/sample-video.mp4` with a real local video path.

## Audio Merge Test

```bash
go run ./cmd/integrations/audio-merge-test \
	-video assets/shorts/sample-video.mp4 \
	-audio assets/shorts-outputs-1/audio.mp3 \
	-output tmp/audio-merge-test/final.mp4
```

Replace `assets/shorts/sample-video.mp4` with a real local video path.

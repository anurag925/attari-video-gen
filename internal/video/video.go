package video

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	WorkingDir      string
	SubtitleASSPath string
	OutputPath      string
}

func Cut(ctx context.Context, cfg Config, inputPath string, durationSeconds int) (string, error) {
	outputPath := filepath.Join(cfg.WorkingDir, "video_cut.mp4")

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-i", inputPath,
		"-t", fmt.Sprintf("%d", durationSeconds),
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-c:a", "aac",
		"-strict", "experimental",
		outputPath,
	)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to cut video: %w", err)
	}

	return outputPath, nil
}

func createASSFile(path string, srt string) error {
	lines := []string{
		"[Script Info]",
		"Title: Subtitles",
		"ScriptType: v4.00",
		"WrapStyle: 0",
		"PlayResX: 1280",
		"PlayResY: 720",
		"MSFT: MSFilter",
		"",
		"[V4+ Styles]",
		"Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding",
		`Style: Default,Arial,56,&H00FFFFFF,&H000000FF,&H00000000,&H00000000,-1,0,0,0,100,100,0,0,1,3,0,5,10,10,60,1`,
		"",
		"[Events]",
		"Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text",
	}

	for _, cue := range parseSRTCues(srt) {
		text := strings.ReplaceAll(cue.text, "\n", "\\N")
		lines = append(lines, fmt.Sprintf("Dialogue: 0,%s,%s,Default,,0,0,0,,%s", formatASSTimestamp(cue.start), formatASSTimestamp(cue.end), text))
	}

	content := strings.Join(lines, "\n")
	return os.WriteFile(path, []byte(content), 0644)
}

type srtCue struct {
	start time.Duration
	end   time.Duration
	text  string
}

func parseSRTCues(srt string) []srtCue {
	blocks := strings.Split(strings.ReplaceAll(strings.TrimSpace(srt), "\r\n", "\n"), "\n\n")
	cues := make([]srtCue, 0, len(blocks))

	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		lines := strings.Split(block, "\n")
		if len(lines) < 2 {
			continue
		}

		timestampLineIndex := 0
		if isNumericLine(strings.TrimSpace(lines[0])) {
			timestampLineIndex = 1
		}
		if timestampLineIndex >= len(lines) {
			continue
		}

		start, end, ok := parseSRTTiming(strings.TrimSpace(lines[timestampLineIndex]))
		if !ok {
			continue
		}

		textLines := lines[timestampLineIndex+1:]
		for idx := range textLines {
			textLines[idx] = strings.TrimSpace(textLines[idx])
		}

		text := strings.TrimSpace(strings.Join(textLines, "\n"))
		if text == "" {
			continue
		}

		cues = append(cues, srtCue{start: start, end: end, text: text})
	}

	return cues
}

func parseSRTTiming(line string) (time.Duration, time.Duration, bool) {
	parts := strings.Split(line, " --> ")
	if len(parts) != 2 {
		return 0, 0, false
	}

	start, err := parseSRTTimestamp(parts[0])
	if err != nil {
		return 0, 0, false
	}
	end, err := parseSRTTimestamp(parts[1])
	if err != nil || end <= start {
		return 0, 0, false
	}

	return start, end, true
}

func parseSRTTimestamp(value string) (time.Duration, error) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid timestamp: %q", value)
	}

	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}
	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}
	secondsParts := strings.Split(parts[2], ",")
	if len(secondsParts) != 2 {
		return 0, fmt.Errorf("invalid timestamp: %q", value)
	}
	seconds, err := strconv.Atoi(secondsParts[0])
	if err != nil {
		return 0, err
	}
	milliseconds, err := strconv.Atoi(secondsParts[1])
	if err != nil {
		return 0, err
	}

	return time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second +
		time.Duration(milliseconds)*time.Millisecond, nil
}

func formatASSTimestamp(value time.Duration) string {
	if value < 0 {
		value = 0
	}

	hours := value / time.Hour
	value -= hours * time.Hour
	minutes := value / time.Minute
	value -= minutes * time.Minute
	seconds := value / time.Second
	value -= seconds * time.Second
	centiseconds := value / (10 * time.Millisecond)

	return fmt.Sprintf("%d:%02d:%02d.%02d", hours, minutes, seconds, centiseconds)
}

func isNumericLine(line string) bool {
	if line == "" {
		return false
	}

	for _, char := range line {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}

func Cleanup(cfg Config, files []string) {
	for _, f := range files {
		os.Remove(f)
	}
}

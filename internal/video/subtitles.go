package video

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func AddSubtitles(ctx context.Context, cfg Config, videoPath string, subtitlePath string) (string, error) {
	if videoPath == "" {
		return "", fmt.Errorf("video path is required")
	}
	if subtitlePath == "" {
		return "", fmt.Errorf("subtitle path is required")
	}

	outputPath := cfg.OutputPath
	replaceInput := false
	if outputPath == "" {
		tempFile, err := os.CreateTemp(filepath.Dir(videoPath), "subtitled-*"+filepath.Ext(videoPath))
		if err != nil {
			return "", fmt.Errorf("create temp subtitle output: %w", err)
		}

		outputPath = tempFile.Name()
		if err := tempFile.Close(); err != nil {
			return "", fmt.Errorf("close temp subtitle output: %w", err)
		}

		replaceInput = true
	}

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-i", videoPath,
		"-vf", fmt.Sprintf("ass=filename='%s'", escapeFFmpegFilterPath(subtitlePath)),
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-c:a", "copy",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to burn subtitles: %w: %s", err, strings.TrimSpace(string(output)))
	}

	if replaceInput {
		if err := os.Rename(outputPath, videoPath); err != nil {
			_ = os.Remove(outputPath)
			return "", fmt.Errorf("replace source video with subtitled output: %w", err)
		}

		return videoPath, nil
	}

	return outputPath, nil
}

func escapeFFmpegFilterPath(path string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`:`, `\:`,
		`,`, `\,`,
		`[`, `\[`,
		`]`, `\]`,
		`'`, `\'`,
	)

	return replacer.Replace(path)
}

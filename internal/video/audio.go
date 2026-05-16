package video

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func MergeAudioVideo(ctx context.Context, cfg Config, videoPath string, audioPath string, outputName string) (string, error) {
	if videoPath == "" {
		return "", fmt.Errorf("video path is required")
	}
	if audioPath == "" {
		return "", fmt.Errorf("audio path is required")
	}

	outputPath := cfg.OutputPath
	replaceInput := false
	if outputPath == "" {
		tempFile, err := os.CreateTemp(filepath.Dir(videoPath), "merged-*.mp4")
		if err != nil {
			return "", fmt.Errorf("create temp merged output: %w", err)
		}

		outputPath = tempFile.Name()
		if err := tempFile.Close(); err != nil {
			return "", fmt.Errorf("close temp merged output: %w", err)
		}

		replaceInput = true
	}

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-i", videoPath,
		"-i", audioPath,
		"-map", "0:v:0",
		"-map", "1:a:0",
		"-c:v", "copy",
		"-c:a", "aac",
		"-shortest",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to merge audio and video: %w: %s", err, strings.TrimSpace(string(output)))
	}

	if replaceInput {
		if err := os.Rename(outputPath, videoPath); err != nil {
			_ = os.Remove(outputPath)
			return "", fmt.Errorf("replace source video with merged output: %w", err)
		}

		return videoPath, nil
	}

	return outputPath, nil
}

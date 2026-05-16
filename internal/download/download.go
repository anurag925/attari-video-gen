package download

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Config struct {
	OutputDir string
}

func Download(ctx context.Context, cfg Config, url string) (string, error) {
	if err := checkYTDLP(); err != nil {
		return "", fmt.Errorf("yt-dlp not found: %w", err)
	}

	if cfg.OutputDir == "" {
		cfg.OutputDir = "."
	}

	outputPath := filepath.Join(cfg.OutputDir, "video.mp4")

	cmd := exec.CommandContext(ctx, "yt-dlp",
		"-f", "best[height<=720]",
		"-o", outputPath,
		"--no-playlist",
		"--merge-output-format", "mp4",
		url,
	)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to download video: %w", err)
	}

	return outputPath, nil
}

func checkYTDLP() error {
	cmd := exec.Command("yt-dlp", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("yt-dlp is not installed. Install via: brew install yt-dlp")
	}
	return nil
}

func EnsureFFmpeg() error {
	cmd := exec.Command("ffmpeg", "-version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg is not installed. Install via: brew install ffmpeg")
	}
	return nil
}

func WorkingDir() (string, error) {
	dir := filepath.Join("assets", "shorts-outputs")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	return abs, nil
}

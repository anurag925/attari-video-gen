package video

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GenerateSubtitles now accepts a custom output filename/path
func GenerateSubtitles(audioPath, outputSRTPath string) error {
	// 1. Validate audio with ffprobe
	slog.Info("🔍 Validating audio...", "path", audioPath)
	dur, err := getAudioDuration(audioPath)
	if err != nil {
		return fmt.Errorf("ffprobe failed: %w", err)
	}
	slog.Info("📀 Audio duration", "duration", dur)

	// 2. Create a temporary directory for whisperx output
	tmpDir, err := os.MkdirTemp("", "whisperx_out_*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir) // Clean up automatically when function returns

	// 3. Run whisperx alignment
	slog.Info("🎙️ Running whisperX alignment...", "path", audioPath, "temp_dir", tmpDir)
	cmd := exec.Command("uvx", "whisperx",
		audioPath,
		"--model", "base",
		"--language", "en",
		"--output_format", "srt",
		"--output_dir", tmpDir,
		"--compute_type", "int8",
	)
	slog.Info("🚀 Executing command", "cmd", cmd.String())
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("whisperx failed: %w", err)
	}

	slog.Info("whisperx output", "output", string(out))

	// 4. Locate the auto-generated SRT file
	audioBase := strings.TrimSuffix(filepath.Base(audioPath), filepath.Ext(audioPath))
	generatedPath := filepath.Join(tmpDir, audioBase+".srt")

	if _, err := os.Stat(generatedPath); os.IsNotExist(err) {
		// Fallback: scan temp dir if naming convention differs by version
		matches, _ := filepath.Glob(filepath.Join(tmpDir, "*.srt"))
		if len(matches) == 0 {
			return fmt.Errorf("whisperx did not generate an SRT file in %s", tmpDir)
		}
		generatedPath = matches[0]
	}

	// 5. Ensure target directory exists
	if err := os.MkdirAll(filepath.Dir(outputSRTPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 6. Move to desired output path
	slog.Info("📄 Moving to custom name...", "output_path", outputSRTPath)
	return safeMove(generatedPath, outputSRTPath)
}

func getAudioDuration(path string) (float64, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1", path)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	var dur float64
	_, err = fmt.Sscan(strings.TrimSpace(string(out)), &dur)
	return dur, err
}

// safeMove handles cross-filesystem renames
func safeMove(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}
	// Fallback for cross-device moves
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		os.Remove(dst) // Clean up partial file
		return err
	}
	return os.Remove(src)
}

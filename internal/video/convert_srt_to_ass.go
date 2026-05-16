package video

import (
	"fmt"
	"os/exec"
)

func ConvertSRTToASS(srtPath string, assPath string) error {
	cmd := exec.Command("ffmpeg", "-y", "-i", srtPath, assPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to convert SRT to ASS: %v, output: %s", err, string(output))
	}
	return nil
}

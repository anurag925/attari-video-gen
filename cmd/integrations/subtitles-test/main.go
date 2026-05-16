package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/anurag925/attari-video-gen/internal/video"
)

func main() {
	videoPath := flag.String("video", "", "Path to the input video")
	subtitlePath := flag.String("subtitles", "", "Path to the ASS subtitle file")
	outputPath := flag.String("output", "", "Path to the output video")
	workDir := flag.String("workdir", "", "Working directory for generated output")
	flag.Parse()

	if *videoPath == "" || *subtitlePath == "" {
		flag.Usage()
		log.Fatal("both -video and -subtitles are required")
	}

	resolvedWorkDir := *workDir
	if resolvedWorkDir == "" {
		if *outputPath != "" {
			resolvedWorkDir = filepath.Dir(*outputPath)
		} else {
			cwd, err := os.Getwd()
			if err != nil {
				log.Fatalf("get working directory: %v", err)
			}
			resolvedWorkDir = cwd
		}
	}

	result, err := video.AddSubtitles(context.Background(), video.Config{
		WorkingDir: resolvedWorkDir,
		OutputPath: *outputPath,
	}, *videoPath, *subtitlePath)
	if err != nil {
		log.Fatalf("burn subtitles: %v", err)
	}

	fmt.Println(result)
	fmt.Println("Run completed successfully")
}

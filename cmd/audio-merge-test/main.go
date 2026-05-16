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
	videoPath := flag.String("video", "assets/shorts-outputs/video_cut.mp4", "Path to the input video")
	audioPath := flag.String("audio", "assets/shorts-outputs/audio.mp3", "Path to the input audio")
	outputPath := flag.String("output", "", "Path to the output video")
	flag.Parse()

	if *videoPath == "" || *audioPath == "" {
		flag.Usage()
		log.Fatal("-video, -audio are required")
	}

	if _, err := os.Stat(*videoPath); err != nil {
		log.Fatalf("video file not found: %v", err)
	}
	if _, err := os.Stat(*audioPath); err != nil {
		log.Fatalf("audio file not found: %v", err)
	}

	result, err := video.MergeAudioVideo(context.Background(), video.Config{
		WorkingDir: filepath.Dir(*outputPath),
		OutputPath: *outputPath,
	}, *videoPath, *audioPath, outputName(*outputPath))
	if err != nil {
		log.Fatalf("merge audio/video: %v", err)
	}

	fmt.Println(result)
	fmt.Println("Run completed successfully")
}

func outputName(path string) string {
	base := filepath.Base(path)
	return base[:len(base)-len(filepath.Ext(base))]
}

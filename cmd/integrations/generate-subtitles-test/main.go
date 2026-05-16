package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/anurag925/attari-video-gen/internal/video"
)

func main() {
	audioPath := flag.String("audio", "assets/shorts-outputs/audio.mp3", "Path to the input audio file")
	outputPath := flag.String("output", "assets/shorts-outputs/generated-1234-audio.srt", "Path to the output SRT file")
	flag.Parse()

	if *audioPath == "" || *outputPath == "" {
		flag.Usage()
		log.Fatal("both -audio and -output are required")
	}

	if _, err := os.Stat(*audioPath); err != nil {
		log.Fatalf("audio file not found: %v", err)
	}

	if err := video.GenerateSubtitles(*audioPath, *outputPath); err != nil {
		log.Fatalf("generate subtitles: %v", err)
	}

	fmt.Printf("Generated subtitles at %s\n", *outputPath)
	fmt.Println("Run completed successfully")
}

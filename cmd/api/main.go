package main

import (
	"flag"
	"log"
	"log/slog"
	"os"

	"github.com/anurag925/attari-video-gen/internal/server"
	"github.com/joho/godotenv"
)

var (
	flagPort string
	flagHost string
)

func main() {
	flag.StringVar(&flagPort, "port", "8080", "Server port")
	flag.StringVar(&flagHost, "host", "localhost", "Server host")
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Println("Warning: error reading .env file, proceeding with environment variables only")
	}

	slog.Info("Starting API server", "host", flagHost, "port", flagPort)

	srv := server.New()
	srv.SetupRoutes()

	if err := srv.StartWithGracefulShutdown(flagHost, flagPort); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}
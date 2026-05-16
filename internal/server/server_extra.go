package server

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

// Start runs the HTTP server and blocks until the context is cancelled.
func (s *Server) Start(host, port string) error {
	return s.echo.Start(host + ":" + port)
}

// StartWithGracefulShutdown runs the server and waits for interrupt signal for graceful shutdown.
func (s *Server) StartWithGracefulShutdown(host, port string) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := s.echo.Start(host + ":" + port); err != nil {
			slog.Info("shutting down server", "error", err)
		}
	}()

	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10)
	defer cancel()
	return s.echo.Shutdown(ctx)
}

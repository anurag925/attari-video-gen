package server

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/anurag925/attari-video-gen/internal/server/handlers"
	"github.com/anurag925/attari-video-gen/internal/server/store"
)

type Server struct {
	echo        *echo.Echo
	artifacts   handlers.ArtifactsHandler
	pipelines   handlers.PipelinesHandler
	health      handlers.HealthHandler
	store       *store.Store
}

func New() *Server {
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	return &Server{
		echo: e,
		store: store.New(),
	}
}

func (s *Server) SetupRoutes() {
	s.artifacts = *handlers.NewArtifactsHandler(s.store)
	s.pipelines = *handlers.NewPipelinesHandler(s.store)
	s.health = *handlers.NewHealthHandler()

	// Health
	s.echo.GET("/health", s.health.Check)

	// API v1
	v1 := s.echo.Group("/api/v1")

	// Artifacts
	artifacts := v1.Group("/artifacts")
	artifacts.GET("", s.artifacts.List)
	artifacts.GET("/:name", s.artifacts.Get)
	artifacts.GET("/:name/download", s.artifacts.Download)
	artifacts.DELETE("/:name", s.artifacts.Delete)

	// Pipelines
	pipelines := v1.Group("/pipelines")
	pipelines.GET("", s.pipelines.List)
	pipelines.GET("/:id", s.pipelines.Get)
	pipelines.POST("", s.pipelines.Create)
	pipelines.POST("/:id/cancel", s.pipelines.Cancel)
	pipelines.DELETE("/:id", s.pipelines.Delete)
}
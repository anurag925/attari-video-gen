package handlers

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v4"

	"github.com/anurag925/attari-video-gen/internal/server/models"
	"github.com/anurag925/attari-video-gen/internal/server/store"
)

type ArtifactsHandler struct {
	store *store.Store
}

func NewArtifactsHandler(s *store.Store) *ArtifactsHandler {
	return &ArtifactsHandler{store: s}
}

// List returns all artifacts.
// GET /api/v1/artifacts
func (h *ArtifactsHandler) List(c echo.Context) error {
	artifacts := h.store.ListArtifacts()

	response := models.ArtifactListResponse{
		Artifacts: make([]models.Artifact, len(artifacts)),
		Total:     len(artifacts),
	}
	for i, a := range artifacts {
		response.Artifacts[i] = *a
	}

	return c.JSON(http.StatusOK, response)
}

// Get returns a specific artifact by name.
// GET /api/v1/artifacts/:name
func (h *ArtifactsHandler) Get(c echo.Context) error {
	name := c.Param("name")
	artifact := h.store.GetArtifact(name)

	if artifact == nil {
		return c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "not_found",
			Message: "artifact not found",
		})
	}

	return c.JSON(http.StatusOK, artifact)
}

// Download returns the artifact file for download.
// GET /api/v1/artifacts/:name/download
func (h *ArtifactsHandler) Download(c echo.Context) error {
	name := c.Param("name")
	artifact := h.store.GetArtifact(name)

	if artifact == nil {
		return c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "not_found",
			Message: "artifact not found",
		})
	}

	if _, err := os.Stat(artifact.Path); os.IsNotExist(err) {
		return c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "not_found",
			Message: "artifact file not found on disk",
		})
	}

	return c.File(artifact.Path)
}

// Delete removes an artifact.
// DELETE /api/v1/artifacts/:name
func (h *ArtifactsHandler) Delete(c echo.Context) error {
	name := c.Param("name")
	artifact := h.store.GetArtifact(name)

	if artifact == nil {
		return c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "not_found",
			Message: "artifact not found",
		})
	}

	// Delete file from disk if exists
	if _, err := os.Stat(artifact.Path); err == nil {
		if err := os.Remove(artifact.Path); err != nil {
			// Log error but continue with store deletion
		}
	}

	h.store.DeleteArtifact(name)

	return c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "artifact deleted",
	})
}

// ServeStatic serves static files from a directory.
// GET /api/v1/artifacts/static/:name
func (h *ArtifactsHandler) ServeStatic(c echo.Context) error {
	name := c.Param("name")
	artifact := h.store.GetArtifact(name)

	if artifact == nil {
		return c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "not_found",
			Message: "artifact not found",
		})
	}

	ext := filepath.Ext(artifact.Path)
	contentType := getContentType(ext)
	slog.Info("Serving static file", "name", name, "path", artifact.Path, "content_type", contentType)
	return c.File(artifact.Path)
}

func getContentType(ext string) string {
	switch ext {
	case ".mp4":
		return "video/mp4"
	case ".mp3", ".m4a":
		return "audio/mpeg"
	case ".ass":
		return "text/plain"
	case ".srt":
		return "text/plain"
	case ".txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

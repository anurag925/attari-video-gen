package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/anurag925/attari-video-gen/internal/config"
	"github.com/anurag925/attari-video-gen/internal/server/models"
	"github.com/anurag925/attari-video-gen/internal/server/store"
)

type PipelinesHandler struct {
	store *store.Store
}

func NewPipelinesHandler(s *store.Store) *PipelinesHandler {
	return &PipelinesHandler{store: s}
}

// List returns all pipelines.
// GET /api/v1/pipelines
func (h *PipelinesHandler) List(c echo.Context) error {
	pipelines := h.store.ListPipelines()

	response := models.PipelineListResponse{
		Pipelines: make([]models.Pipeline, len(pipelines)),
		Total:     len(pipelines),
	}
	for i, p := range pipelines {
		response.Pipelines[i] = *p
	}

	return c.JSON(http.StatusOK, response)
}

// Get returns a specific pipeline by ID.
// GET /api/v1/pipelines/:id
func (h *PipelinesHandler) Get(c echo.Context) error {
	id := c.Param("id")
	pipeline := h.store.GetPipeline(id)

	if pipeline == nil {
		return c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "not_found",
			Message: "pipeline not found",
		})
	}

	return c.JSON(http.StatusOK, pipeline)
}

// Create starts a new pipeline.
// POST /api/v1/pipelines
func (h *PipelinesHandler) Create(c echo.Context) error {
	var req models.CreatePipelineRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "invalid request body",
		})
	}

	// Validation
	if req.VideoURL == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: "video_url is required",
		})
	}
	if req.Duration <= 0 {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: "duration must be positive",
		})
	}
	if req.URL == "" && req.Text == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: "either url or text is required",
		})
	}

	// Generate output name if not provided
	outputName := req.OutputName
	if outputName == "" {
		outputName = "output-" + uuid.New().String()[:8] + ".mp4"
	}

	now := time.Now()
	pipeline := &models.Pipeline{
		ID:          uuid.New().String(),
		Status:      models.PipelineStatusPending,
		OutputName:  outputName,
		Progress:    0.0,
		Steps:       buildPipelineSteps(req.Steps),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Store pipeline input for later processing
	pipeline.Input = models.PipelineInput{
		VideoURL:   req.VideoURL,
		URL:        req.URL,
		Text:       req.Text,
		Duration:   req.Duration,
		Steps:      req.Steps,
	}

	h.store.SavePipeline(pipeline)

	// TODO: Start pipeline execution asynchronously
	go h.executePipeline(pipeline.ID)

	return c.JSON(http.StatusCreated, pipeline)
}

// Cancel stops a running pipeline.
// POST /api/v1/pipelines/:id/cancel
func (h *PipelinesHandler) Cancel(c echo.Context) error {
	id := c.Param("id")
	pipeline := h.store.GetPipeline(id)

	if pipeline == nil {
		return c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "not_found",
			Message: "pipeline not found",
		})
	}

	if pipeline.Status != models.PipelineStatusRunning && pipeline.Status != models.PipelineStatusPending {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_state",
			Message: "pipeline cannot be cancelled in current state",
		})
	}

	now := time.Now()
	pipeline.Status = models.PipelineStatusCancelled
	pipeline.UpdatedAt = now
	pipeline.CompletedAt = &now

	h.store.SavePipeline(pipeline)

	return c.JSON(http.StatusOK, pipeline)
}

// Delete removes a pipeline.
// DELETE /api/v1/pipelines/:id
func (h *PipelinesHandler) Delete(c echo.Context) error {
	id := c.Param("id")
	pipeline := h.store.GetPipeline(id)

	if pipeline == nil {
		return c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "not_found",
			Message: "pipeline not found",
		})
	}

	h.store.DeletePipeline(id)

	// TODO: Clean up associated artifacts

	return c.JSON(http.StatusOK, models.SuccessResponse{
		Message: "pipeline deleted",
	})
}

// executePipeline runs the pipeline in the background.
// This is a placeholder that will be connected to the existing processor.
func (h *PipelinesHandler) executePipeline(pipelineID string) {
	// TODO: Implement pipeline execution using the existing processor
}

// buildPipelineSteps creates the pipeline step list from the request.
func buildPipelineSteps(steps []string) []models.PipelineStep {
	if len(steps) == 0 {
		steps = config.AllStepsNames()
	}

	result := make([]models.PipelineStep, len(steps))
	for i, name := range steps {
		result[i] = models.PipelineStep{
			Name:   name,
			Status: models.StepStatusPending,
		}
	}
	return result
}
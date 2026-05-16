package models

import "time"

// Pipeline represents a video generation pipeline execution.
type Pipeline struct {
	ID          string           `json:"id"`
	Status      PipelineStatus   `json:"status"`
	Input       PipelineInput    `json:"input"`
	OutputName  string           `json:"output_name"`
	Progress    float64          `json:"progress"` // 0.0 to 1.0
	Steps       []PipelineStep   `json:"steps"`
	Artifcats   []string         `json:"artifacts,omitempty"`
	Error       string           `json:"error,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	CompletedAt *time.Time       `json:"completed_at,omitempty"`
}

// PipelineStatus represents the current state of a pipeline.
type PipelineStatus string

const (
	PipelineStatusPending   PipelineStatus = "pending"
	PipelineStatusRunning   PipelineStatus = "running"
	PipelineStatusCompleted PipelineStatus = "completed"
	PipelineStatusFailed    PipelineStatus = "failed"
	PipelineStatusCancelled PipelineStatus = "cancelled"
)

// PipelineStep represents a single step within the pipeline.
type PipelineStep struct {
	Name        string        `json:"name"`
	Status      StepStatus    `json:"status"`
	StartedAt   *time.Time    `json:"started_at,omitempty"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
	Artifact    string        `json:"artifact,omitempty"`
	Error       string        `json:"error,omitempty"`
}

// StepStatus represents the current state of a pipeline step.
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
)

// PipelineInput represents the input configuration for starting a pipeline.
type PipelineInput struct {
	VideoURL   string   `json:"video_url"`
	URL        string   `json:"url,omitempty"`
	Text       string   `json:"text,omitempty"`
	Duration   int      `json:"duration"`
	Steps      []string `json:"steps,omitempty"` // optional, uses defaults if empty
}

// PipelineListResponse is the response for listing pipelines.
type PipelineListResponse struct {
	Pipelines []Pipeline `json:"pipelines"`
	Total     int        `json:"total"`
}

// CreatePipelineRequest is the request body for creating a new pipeline.
type CreatePipelineRequest struct {
	VideoURL   string   `json:"video_url" validate:"required"`
	URL        string   `json:"url"`
	Text       string   `json:"text"`
	Duration   int      `json:"duration" validate:"required,gt=0"`
	OutputName string   `json:"output_name"`
	Steps      []string `json:"steps"`
}
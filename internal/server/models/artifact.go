package models

import "time"

// Artifact represents a generated output artifact (video, audio, subtitles, etc.)
type Artifact struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"` // video, audio, subtitles, text
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	PipelineID  string    `json:"pipeline_id,omitempty"`
	StepName    string    `json:"step_name,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	DownloadURL string    `json:"download_url,omitempty"`
}

// ArtifactListResponse is the response for listing artifacts.
type ArtifactListResponse struct {
	Artifacts []Artifact `json:"artifacts"`
	Total     int        `json:"total"`
}
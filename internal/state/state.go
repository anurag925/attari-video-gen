package state

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PipelineState tracks the progress of a video generation pipeline.
// Each field represents either an artifact path or a completion flag for a pipeline step.
type PipelineState struct {
	// Signature is the hash of the input configuration, used to detect changes.
	Signature string `json:"signature"`

	// Artifact paths
	SourceTextPath     string `json:"source_text_path,omitempty"`
	SummarisedTextPath string `json:"summarized_text_path,omitempty"`
	SrtSubtitlesPath   string `json:"srt_subtitles_path,omitempty"`
	SubtitlesPath      string `json:"subtitles_path,omitempty"`
	DownloadedPath     string `json:"downloaded_path,omitempty"`
	CutVideoPath       string `json:"cut_video_path,omitempty"`
	AudioPath          string `json:"audio_path,omitempty"`
	VideoWithSubsPath  string `json:"video_with_subs_path,omitempty"`
	FinalPath          string `json:"final_path,omitempty"`

	// Completion flags
	SourceTextDone     bool `json:"source_text_done"`
	SummarisedTextDone bool `json:"summarized_text_done"`
	SrtSubtitlesDone   bool `json:"srt_subtitles_done"`
	SubtitlesDone      bool `json:"subtitles_done"`
	DownloadDone       bool `json:"download_done"`
	CutDone            bool `json:"cut_done"`
	AudioDone          bool `json:"audio_done"`
	SubtitlesBurned    bool `json:"subtitles_burned"`
	MergeDone          bool `json:"merge_done"`
}

// New creates a fresh pipeline state.
func New() *PipelineState {
	return &PipelineState{}
}

// Load reads a pipeline state from the given path. Returns an empty state if the file does not exist.
func Load(path string) (*PipelineState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return New(), nil
		}
		return nil, err
	}

	var state PipelineState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// Save writes the pipeline state to the given path.
func (s *PipelineState) Save(path string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Savepoint writes the current state to the state file. Use after each pipeline step completes.
func (s *PipelineState) Savepoint(statePath string) error {
	return s.Save(statePath)
}

// IsStepDone returns true if the given completion flag is set.
func (s *PipelineState) IsStepDone(step string) bool {
	switch step {
	case "source_text":
		return s.SourceTextDone
	case "summarized_text":
		return s.SummarisedTextDone
	case "srt_subtitles":
		return s.SrtSubtitlesDone
	case "subtitles":
		return s.SubtitlesDone
	case "download":
		return s.DownloadDone
	case "cut":
		return s.CutDone
	case "audio":
		return s.AudioDone
	case "subtitles_burned":
		return s.SubtitlesBurned
	case "merge":
		return s.MergeDone
	default:
		return false
	}
}

// SetStepDone marks the given step as complete and persists the state.
func (s *PipelineState) SetStepDone(statePath, step string, artifactPath string) error {
	switch step {
	case "source_text":
		s.SourceTextDone = true
		s.SourceTextPath = artifactPath
	case "summarized_text":
		s.SummarisedTextDone = true
		s.SummarisedTextPath = artifactPath
	case "srt_subtitles":
		s.SrtSubtitlesDone = true
		s.SrtSubtitlesPath = artifactPath
	case "subtitles":
		s.SubtitlesDone = true
		s.SubtitlesPath = artifactPath
	case "download":
		s.DownloadDone = true
		s.DownloadedPath = artifactPath
	case "cut":
		s.CutDone = true
		s.CutVideoPath = artifactPath
	case "audio":
		s.AudioDone = true
		s.AudioPath = artifactPath
	case "subtitles_burned":
		s.SubtitlesBurned = true
		s.VideoWithSubsPath = artifactPath
	case "merge":
		s.MergeDone = true
		s.FinalPath = artifactPath
	}

	return s.Save(statePath)
}

// GetArtifact returns the artifact path for the given step.
func (s *PipelineState) GetArtifact(step string) string {
	switch step {
	case "source_text":
		return s.SourceTextPath
	case "summarized_text":
		return s.SummarisedTextPath
	case "srt_subtitles":
		return s.SrtSubtitlesPath
	case "subtitles":
		return s.SubtitlesPath
	case "download":
		return s.DownloadedPath
	case "cut":
		return s.CutVideoPath
	case "audio":
		return s.AudioPath
	case "subtitles_burned":
		return s.VideoWithSubsPath
	case "merge":
		return s.FinalPath
	default:
		return ""
	}
}

// SetArtifact updates the artifact path for the given step.
func (s *PipelineState) SetArtifact(step, path string) {
	switch step {
	case "source_text":
		s.SourceTextPath = path
	case "summarized_text":
		s.SummarisedTextPath = path
	case "srt_subtitles":
		s.SrtSubtitlesPath = path
	case "subtitles":
		s.SubtitlesPath = path
	case "download":
		s.DownloadedPath = path
	case "cut":
		s.CutVideoPath = path
	case "audio":
		s.AudioPath = path
	case "subtitles_burned":
		s.VideoWithSubsPath = path
	case "merge":
		s.FinalPath = path
	}
}

// PathExists returns true if the artifact path for the given step exists and is a file.
func (s *PipelineState) PathExists(step string) bool {
	path := s.GetArtifact(step)
	if path == "" {
		return false
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return !info.IsDir()
}

// Reset clears all completion flags and artifact paths, keeping the signature.
func (s *PipelineState) Reset() {
	*s = PipelineState{
		Signature: s.Signature,
	}
}

// ComputeSignature generates a hash from the input configuration to detect changes.
func ComputeSignature(videoURL, pageURL, text, duration, outputName string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		videoURL,
		pageURL,
		text,
		duration,
		outputName,
	}, "\n")))

	return hex.EncodeToString(sum[:])
}

// Itoa converts an integer to a string.
func Itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

// StatePath returns the path for the pipeline state file in the given output directory.
func StatePath(outputDir, outputName string) string {
	return filepath.Join(outputDir, BaseName(outputName)+"-progress.json")
}

// BaseName returns the base name of a file without extension.
func BaseName(outputName string) string {
	baseName := filepath.Base(outputName)
	ext := filepath.Ext(baseName)
	name := strings.TrimSuffix(baseName, ext)
	if name == "" {
		return "output"
	}
	return name
}

// ArtifactPath resolves an output path for an artifact. It prefers the first non-empty candidate
// and derives the directory from it, falling back to fallbackDir.
func ArtifactPath(fallbackDir, fileName string, candidates ...string) string {
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		return filepath.Join(filepath.Dir(candidate), fileName)
	}
	return filepath.Join(fallbackDir, fileName)
}

// ReadTextFile reads and returns the contents of a text file, trimmed of whitespace.
func ReadTextFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// WriteTextArtifact writes content to a file in the output directory with a given suffix.
func WriteTextArtifact(outputDir, outputName, suffix, content string) (string, error) {
	path := filepath.Join(outputDir, BaseName(outputName)+"-"+suffix)
	if err := os.WriteFile(path, []byte(content+"\n"), 0644); err != nil {
		return "", err
	}
	return path, nil
}

// Manager provides a higher-level interface for managing pipeline state with validation.
type Manager struct {
	statePath string
	state     *PipelineState
}

// NewManager creates a state manager for the given state file path.
func NewManager(statePath string) *Manager {
	return &Manager{
		statePath: statePath,
		state:     New(),
	}
}

// LoadState reads the state from disk. Returns an error if the input signature has changed.
func (m *Manager) LoadState(signature string) error {
	state, err := Load(m.statePath)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	if state.Signature != "" && state.Signature != signature {
		state = New()
		state.Signature = signature
	}
	m.state = state
	return nil
}

// State returns the current pipeline state.
func (m *Manager) State() *PipelineState {
	return m.state
}

// Save persists the current state to disk.
func (m *Manager) Save() error {
	return m.state.Save(m.statePath)
}

// CompleteStep marks a pipeline step as done with its artifact path and persists the state.
func (m *Manager) CompleteStep(step, artifactPath string) error {
	return m.state.SetStepDone(m.statePath, step, artifactPath)
}

// IsStepDone checks if a pipeline step has been completed.
func (m *Manager) IsStepDone(step string) bool {
	return m.state.IsStepDone(step)
}

// GetArtifact returns the artifact path for a completed step.
func (m *Manager) GetArtifact(step string) string {
	return m.state.GetArtifact(step)
}

// ArtifactExists checks if the artifact for a step exists on disk.
func (m *Manager) ArtifactExists(step string) bool {
	return m.state.PathExists(step)
}

// ShouldSkip returns true if the step is already done and the artifact exists.
func (m *Manager) ShouldSkip(step string) (bool, string) {
	if m.state.IsStepDone(step) && m.state.PathExists(step) {
		return true, m.state.GetArtifact(step)
	}
	return false, ""
}

// Reset discards the current state and reinitializes with the given signature.
func (m *Manager) Reset(signature string) {
	m.state = New()
	m.state.Signature = signature
}
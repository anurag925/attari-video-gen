package state

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anurag925/attari-video-gen/internal/config"
)

// StepState tracks whether a step is done and its artifact path.
type StepState struct {
	Done         bool   `json:"done"`
	ArtifactPath string `json:"artifact_path,omitempty"`
}

// PipelineState tracks the progress of a video generation pipeline using dynamic step names.
type PipelineState struct {
	Signature string                `json:"signature"`
	Steps     map[string]StepState `json:"steps,omitempty"`
}

// New creates a fresh pipeline state.
func New() *PipelineState {
	return &PipelineState{
		Steps: make(map[string]StepState),
	}
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

	if state.Steps == nil {
		state.Steps = make(map[string]StepState)
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

// IsStepDone returns true if the given step is marked as done.
func (s *PipelineState) IsStepDone(step config.StepName) bool {
	if s.Steps == nil {
		return false
	}
	st, ok := s.Steps[step.String()]
	return ok && st.Done
}

// SetStepDone marks the given step as complete with its artifact path and persists.
func (s *PipelineState) SetStepDone(statePath string, step config.StepName, artifactPath string) error {
	if s.Steps == nil {
		s.Steps = make(map[string]StepState)
	}
	s.Steps[step.String()] = StepState{Done: true, ArtifactPath: artifactPath}
	return s.Save(statePath)
}

// GetArtifact returns the artifact path for the given step.
func (s *PipelineState) GetArtifact(step config.StepName) string {
	if s.Steps == nil {
		return ""
	}
	st, ok := s.Steps[step.String()]
	if !ok {
		return ""
	}
	return st.ArtifactPath
}

// SetArtifact updates the artifact path for the given step.
func (s *PipelineState) SetArtifact(step config.StepName, path string) {
	if s.Steps == nil {
		s.Steps = make(map[string]StepState)
	}
	st, ok := s.Steps[step.String()]
	if !ok {
		st = StepState{}
	}
	st.ArtifactPath = path
	s.Steps[step.String()] = st
}

// PathExists returns true if the artifact path for the given step exists and is a file.
func (s *PipelineState) PathExists(step config.StepName) bool {
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
		Steps:     make(map[string]StepState),
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
func (m *Manager) CompleteStep(step config.StepName, artifactPath string) error {
	return m.state.SetStepDone(m.statePath, step, artifactPath)
}

// IsStepDone checks if a pipeline step has been completed.
func (m *Manager) IsStepDone(step config.StepName) bool {
	return m.state.IsStepDone(step)
}

// GetArtifact returns the artifact path for a completed step.
func (m *Manager) GetArtifact(step config.StepName) string {
	return m.state.GetArtifact(step)
}

// ArtifactExists checks if the artifact for a step exists on disk.
func (m *Manager) ArtifactExists(step config.StepName) bool {
	return m.state.PathExists(step)
}

// ShouldSkip returns true if the step is already done and the artifact exists.
func (m *Manager) ShouldSkip(step config.StepName) (bool, string) {
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
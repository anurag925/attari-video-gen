package config

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

const (
	// BaseDir is the parent directory for all attari-video-gen files
	BaseDir = "/tmp/attari-video-gen"
	// InputsDir is the directory for input files
	InputsDir = "/tmp/attari-video-gen/inputs"
	// AssetsDir is the directory for generated assets
	AssetsDir = "/tmp/attari-video-gen/assets"
)

// StepName represents the name of a pipeline step.
type StepName string

const (
	StepSourceText      StepName = "source_text"
	StepDownload        StepName = "download"
	StepCut             StepName = "cut"
	StepSummarizedText  StepName = "summarized_text"
	StepAudio           StepName = "audio"
	StepSrtSubtitles    StepName = "srt_subtitles"
	StepSubtitles       StepName = "subtitles"
	StepSubtitlesBurned StepName = "subtitles_burned"
	StepMerge           StepName = "merge"
)

// AllSteps returns all known step names in order.
func AllSteps() []StepName {
	return []StepName{
		StepSourceText,
		StepDownload,
		StepCut,
		StepSummarizedText,
		StepAudio,
		StepSrtSubtitles,
		StepSubtitles,
		StepSubtitlesBurned,
		StepMerge,
	}
}

// String returns the string representation of a StepName.
func (s StepName) String() string {
	return string(s)
}

// IsValid checks if the step name is a known valid step.
func (s StepName) IsValid() bool {
	for _, valid := range AllSteps() {
		if s == valid {
			return true
		}
	}
	return false
}

// AllStepsNames returns all step names as strings.
func AllStepsNames() []string {
	steps := AllSteps()
	names := make([]string, len(steps))
	for i, s := range steps {
		names[i] = s.String()
	}
	return names
}

// Input represents the configuration for a video generation pipeline.
type Input struct {
	VideoURL   string `yaml:"video_url"`
	URL        string `yaml:"url"`  // Page URL to scrape for text
	Text       string `yaml:"text"` // Fallback: direct text input
	Duration   int    `yaml:"duration"`
	OutputName string `yaml:"output_name"`
	WorkDir    string `yaml:"_work_dir"` // Internal: working directory for this input
	Steps      []Step `yaml:"steps"`
}

// Step represents a pipeline step with its configuration.
type Step struct {
	Name    StepName `yaml:"name"`
	Enabled bool     `yaml:"enabled"`
}

// DefaultSteps returns the standard pipeline steps in order.
func DefaultSteps() []Step {
	steps := make([]Step, len(AllSteps()))
	for i, name := range AllSteps() {
		steps[i] = Step{Name: name, Enabled: true}
	}
	return steps
}

// ParseInputFile reads and parses a YAML configuration file.
func ParseInputFile(path string) (*Input, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading input file: %w", err)
	}

	var input Input
	if err := yaml.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("parsing input file: %w", err)
	}

	return &input, nil
}

// SaveInput saves the input configuration to a file.
func (input *Input) SaveInput(path string) error {
	data, err := yaml.Marshal(input)
	if err != nil {
		return fmt.Errorf("marshaling input: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// ValidateInput checks that all required fields are present and valid.
func ValidateInput(input *Input) error {
	if input.VideoURL == "" {
		return fmt.Errorf("video_url is required")
	}
	if input.URL == "" && input.Text == "" {
		return fmt.Errorf("either url or text is required")
	}
	if input.Duration <= 0 {
		return fmt.Errorf("duration must be positive")
	}
	if input.OutputName == "" {
		return fmt.Errorf("output_name is required")
	}
	return nil
}

// EnsureSteps returns the steps if set, otherwise returns the default steps.
func (input *Input) EnsureSteps() []Step {
	if len(input.Steps) == 0 {
		return DefaultSteps()
	}
	return input.Steps
}

// ComputeSignature generates a unique signature for this input configuration.
func (input *Input) ComputeSignature() string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		input.VideoURL,
		input.URL,
		input.Text,
		fmt.Sprintf("%d", input.Duration),
		input.OutputName,
	}, "\n")))
	return hex.EncodeToString(sum[:])
}

// InputFilePath returns the path to the input file for this signature.
func (input *Input) InputFilePath() string {
	return filepath.Join(InputsDir, "input."+input.ComputeSignature()+".yaml")
}

// EnsureWorkDir creates and returns the working directory for this input.
// If WorkDir is already set and exists, returns it.
// Otherwise creates a new directory <output-name>-<randomword> in AssetsDir.
func (input *Input) EnsureWorkDir() (string, error) {
	if input.WorkDir != "" {
		if _, err := os.Stat(input.WorkDir); err == nil {
			return input.WorkDir, nil
		}
	}

	baseName := BaseName(input.OutputName)
	workDir := filepath.Join(AssetsDir, baseName+"-"+randomWord(6))
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return "", fmt.Errorf("creating work dir: %w", err)
	}
	input.WorkDir = workDir
	return workDir, nil
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

// randomWord generates a random lowercase word of the given length.
func randomWord(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz"
	result := make([]byte, length)
	for i := range result {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		result[i] = chars[n.Int64()]
	}
	return string(result)
}

// EnsureBaseDirs creates the base directories if they don't exist.
func EnsureBaseDirs() error {
	if err := os.MkdirAll(InputsDir, 0755); err != nil {
		return fmt.Errorf("creating inputs dir: %w", err)
	}
	if err := os.MkdirAll(AssetsDir, 0755); err != nil {
		return fmt.Errorf("creating assets dir: %w", err)
	}
	return nil
}

const inputTemplate = `# Video generation pipeline configuration
video_url: "https://www.youtube.com/watch?v=VIDEO_ID"
url: "https://example.com/page-to-scrape"  # Page to scrape for text
# text: "Fallback direct text if no url provided"
duration: 20  # Target video duration in seconds
output_name: "output.mp4"

# Pipeline steps (order matters, set enabled: false to skip)
steps:
{{- range .Steps}}
  - name: {{ .Name }}
    enabled: {{ .Enabled }}
{{- end}}
`

// DefaultInputTemplate returns a default YAML template for input configuration.
func DefaultInputTemplate() ([]byte, error) {
	tmpl, err := template.New("input").Parse(inputTemplate)
	if err != nil {
		return nil, err
	}

	input := &Input{
		Steps: DefaultSteps(),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, input); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
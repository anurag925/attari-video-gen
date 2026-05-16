package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Input represents the configuration for a video generation pipeline.
type Input struct {
	VideoURL   string `yaml:"video_url"`
	URL        string `yaml:"url"`  // Page URL to scrape for text
	Text       string `yaml:"text"` // Fallback: direct text input
	Duration   int    `yaml:"duration"`
	OutputName string `yaml:"output_name"`
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
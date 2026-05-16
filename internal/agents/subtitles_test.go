package agents

import (
	"testing"
)

func TestWordLimitForDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration int
		want     int
	}{
		{name: "zero duration still allows one word", duration: 0, want: 1},
		{name: "short duration rounds down conservatively", duration: 15, want: 32},
		{name: "one minute maps to speaking budget", duration: 60, want: 130},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := wordLimitForDuration(tt.duration); got != tt.want {
				t.Fatalf("wordLimitForDuration(%d) = %d, want %d", tt.duration, got, tt.want)
			}
		})
	}
}

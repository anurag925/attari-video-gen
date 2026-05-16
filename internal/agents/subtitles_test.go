package agents

import (
	"strings"
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

func TestCountWords(t *testing.T) {
	text := "Line one.\nLine two, with punctuation."

	if got := countWords(text); got != 6 {
		t.Fatalf("countWords() = %d, want 6", got)
	}
}

func TestTrimToWordLimit(t *testing.T) {
	text := "alpha beta gamma delta"

	if got := trimToWordLimit(text, 3); got != "alpha beta gamma" {
		t.Fatalf("trimToWordLimit() = %q, want %q", got, "alpha beta gamma")
	}
}

func TestTrimToWordLimitLeavesShortTextUntouched(t *testing.T) {
	text := "already short"

	if got := trimToWordLimit(text, 10); got != text {
		t.Fatalf("trimToWordLimit() = %q, want %q", got, text)
	}
}

func TestFormatAsSRT(t *testing.T) {
	text := "Welcome to the tutorial on formatting! This is how a standard SRT file looks."

	got := FormatAsSRT(text, 8)

	if !strings.Contains(got, "1\n00:00:00,000 -->") {
		t.Fatalf("FormatAsSRT() did not include the first cue header: %q", got)
	}
	if gotDialogue := ExtractDialogueFromSRT(got); gotDialogue != text {
		t.Fatalf("FormatAsSRT() dialogue = %q, want %q", gotDialogue, text)
	}
}

func TestExtractDialogueFromSRT(t *testing.T) {
	srt := strings.Join([]string{
		"1",
		"00:00:00,000 --> 00:00:02,000",
		"Welcome to the tutorial on formatting!",
		"",
		"2",
		"00:00:02,150 --> 00:00:04,000",
		"This is how a standard SRT file looks.",
	}, "\n")

	got := ExtractDialogueFromSRT(srt)
	want := "Welcome to the tutorial on formatting! This is how a standard SRT file looks."
	if got != want {
		t.Fatalf("ExtractDialogueFromSRT() = %q, want %q", got, want)
	}
}

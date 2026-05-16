package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	s := New()
	assert.NotNil(t, s)
	assert.Empty(t, s.Signature)
	assert.False(t, s.SourceTextDone)
	assert.False(t, s.SummarisedTextDone)
	assert.False(t, s.DownloadDone)
	assert.False(t, s.CutDone)
	assert.False(t, s.AudioDone)
	assert.False(t, s.SrtSubtitlesDone)
	assert.False(t, s.SubtitlesDone)
	assert.False(t, s.SubtitlesBurned)
	assert.False(t, s.MergeDone)
}

func TestPipelineState_SaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	s := New()
	s.Signature = "test-signature"
	s.SourceTextDone = true
	s.SourceTextPath = "/path/to/source.txt"

	err := s.Save(statePath)
	require.NoError(t, err)

	loaded, err := Load(statePath)
	require.NoError(t, err)
	assert.Equal(t, s.Signature, loaded.Signature)
	assert.Equal(t, s.SourceTextDone, loaded.SourceTextDone)
	assert.Equal(t, s.SourceTextPath, loaded.SourceTextPath)
}

func TestLoad_FileNotFound(t *testing.T) {
	s, err := Load("/nonexistent/path/state.json")
	require.NoError(t, err)
	assert.NotNil(t, s)
	assert.Empty(t, s.Signature)
}

func TestLoad_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "invalid.json")

	err := os.WriteFile(statePath, []byte("not valid json"), 0644)
	require.NoError(t, err)

	_, err = Load(statePath)
	assert.Error(t, err)
}

func TestPipelineState_IsStepDone(t *testing.T) {
	s := New()

	tests := []struct {
		step   string
		done   bool
		setter func()
	}{
		{"source_text", false, func() {}},
		{"summarized_text", false, func() {}},
		{"download", false, func() {}},
		{"cut", false, func() {}},
		{"audio", false, func() {}},
		{"srt_subtitles", false, func() {}},
		{"subtitles", false, func() {}},
		{"subtitles_burned", false, func() {}},
		{"merge", false, func() {}},
		{"source_text", true, func() { s.SourceTextDone = true }},
		{"summarized_text", true, func() { s.SummarisedTextDone = true }},
		{"download", true, func() { s.DownloadDone = true }},
		{"cut", true, func() { s.CutDone = true }},
		{"audio", true, func() { s.AudioDone = true }},
		{"srt_subtitles", true, func() { s.SrtSubtitlesDone = true }},
		{"subtitles", true, func() { s.SubtitlesDone = true }},
		{"subtitles_burned", true, func() { s.SubtitlesBurned = true }},
		{"merge", true, func() { s.MergeDone = true }},
	}

	for _, tt := range tests {
		tt.setter()
		assert.Equal(t, tt.done, s.IsStepDone(tt.step), "step: %s", tt.step)
	}
}

func TestPipelineState_IsStepDone_UnknownStep(t *testing.T) {
	s := New()
	assert.False(t, s.IsStepDone("unknown_step"))
}

func TestPipelineState_SetStepDone(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	tests := []struct {
		step          string
		artifactPath  string
		expectedDone  bool
		expectedPath  string
	}{
		{"source_text", "/path/source.txt", true, "/path/source.txt"},
		{"summarized_text", "/path/summarized.txt", true, "/path/summarized.txt"},
		{"download", "/path/video.mp4", true, "/path/video.mp4"},
		{"cut", "/path/cut.mp4", true, "/path/cut.mp4"},
		{"audio", "/path/audio.mp3", true, "/path/audio.mp3"},
		{"srt_subtitles", "/path/sub.srt", true, "/path/sub.srt"},
		{"subtitles", "/path/final.ass", true, "/path/final.ass"},
		{"subtitles_burned", "/path/subtitled.mp4", true, "/path/subtitled.mp4"},
		{"merge", "/path/final.mp4", true, "/path/final.mp4"},
	}

	for _, tt := range tests {
		s := New()
		err := s.SetStepDone(statePath, tt.step, tt.artifactPath)
		require.NoError(t, err)
		assert.Equal(t, tt.expectedDone, s.IsStepDone(tt.step), "step: %s", tt.step)
		assert.Equal(t, tt.expectedPath, s.GetArtifact(tt.step), "step: %s", tt.step)
	}
}

func TestPipelineState_GetArtifact(t *testing.T) {
	s := New()
	s.SourceTextPath = "/path/source.txt"
	s.SummarisedTextPath = "/path/summarized.txt"
	s.DownloadedPath = "/path/video.mp4"
	s.CutVideoPath = "/path/cut.mp4"
	s.AudioPath = "/path/audio.mp3"
	s.SrtSubtitlesPath = "/path/sub.srt"
	s.SubtitlesPath = "/path/final.ass"
	s.VideoWithSubsPath = "/path/subtitled.mp4"
	s.FinalPath = "/path/final.mp4"

	tests := []struct {
		step    string
		path    string
	}{
		{"source_text", s.SourceTextPath},
		{"summarized_text", s.SummarisedTextPath},
		{"download", s.DownloadedPath},
		{"cut", s.CutVideoPath},
		{"audio", s.AudioPath},
		{"srt_subtitles", s.SrtSubtitlesPath},
		{"subtitles", s.SubtitlesPath},
		{"subtitles_burned", s.VideoWithSubsPath},
		{"merge", s.FinalPath},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.path, s.GetArtifact(tt.step), "step: %s", tt.step)
	}
}

func TestPipelineState_GetArtifact_UnknownStep(t *testing.T) {
	s := New()
	assert.Empty(t, s.GetArtifact("unknown_step"))
}

func TestPipelineState_SetArtifact(t *testing.T) {
	s := New()

	s.SetArtifact("source_text", "/new/path/source.txt")
	assert.Equal(t, "/new/path/source.txt", s.SourceTextPath)

	s.SetArtifact("summarized_text", "/new/path/summarized.txt")
	assert.Equal(t, "/new/path/summarized.txt", s.SummarisedTextPath)

	s.SetArtifact("subtitles_burned", "/new/path/video.mp4")
	assert.Equal(t, "/new/path/video.mp4", s.VideoWithSubsPath)
}

func TestPipelineState_PathExists(t *testing.T) {
	tmpDir := t.TempDir()

	s := New()
	s.SourceTextPath = filepath.Join(tmpDir, "source.txt")
	s.DownloadDone = true

	// File doesn't exist yet
	assert.False(t, s.PathExists("source_text"))

	// Create the file
	f, err := os.Create(s.SourceTextPath)
	require.NoError(t, err)
	f.Close()

	// Now it exists
	assert.True(t, s.PathExists("source_text"))
}

func TestPipelineState_PathExists_EmptyPath(t *testing.T) {
	s := New()
	assert.False(t, s.PathExists("source_text"))
}

func TestPipelineState_PathExists_UnknownStep(t *testing.T) {
	s := New()
	assert.False(t, s.PathExists("unknown_step"))
}

func TestPipelineState_Reset(t *testing.T) {
	s := New()
	s.Signature = "test-signature"
	s.SourceTextDone = true
	s.SourceTextPath = "/path/source.txt"
	s.DownloadDone = true
	s.DownloadedPath = "/path/video.mp4"

	s.Reset()

	assert.Equal(t, "test-signature", s.Signature)
	assert.False(t, s.SourceTextDone)
	assert.Empty(t, s.SourceTextPath)
	assert.False(t, s.DownloadDone)
	assert.Empty(t, s.DownloadedPath)
}

func TestComputeSignature(t *testing.T) {
	sig1 := ComputeSignature("url1", "page1", "text1", "60", "output1")
	sig2 := ComputeSignature("url1", "page1", "text1", "60", "output1")
	sig3 := ComputeSignature("url2", "page1", "text1", "60", "output1")

	assert.NotEmpty(t, sig1)
	assert.Equal(t, sig1, sig2) // Same inputs should produce same signature
	assert.NotEqual(t, sig3, sig1) // Different inputs should produce different signature
	assert.Len(t, sig1, 64)        // SHA256 hex is 64 characters
}

func TestStatePath(t *testing.T) {
	tests := []struct {
		outputDir  string
		outputName string
		expected   string
	}{
		{"/tmp", "output", "/tmp/output-progress.json"},
		{"/tmp", "video.mp4", "/tmp/video-progress.json"},
		{"/tmp", "/path/to/video.mp4", "/tmp/video-progress.json"},
	}

	for _, tt := range tests {
		result := StatePath(tt.outputDir, tt.outputName)
		assert.Equal(t, tt.expected, result)
	}
}

func TestBaseName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"output", "output"},
		{"video.mp4", "video"},
		{"/path/to/video.mp4", "video"},
		{"video.tar.gz", "video.tar"},
		{"", "output"},
		{".hidden", "output"},
	}

	for _, tt := range tests {
		result := BaseName(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestArtifactPath(t *testing.T) {
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "existing.mp4")
	f, err := os.Create(existingFile)
	require.NoError(t, err)
	f.Close()

	tests := []struct {
		fallbackDir string
		fileName    string
		candidates  []string
		expected    string
	}{
		{tmpDir, "output.mp4", []string{}, filepath.Join(tmpDir, "output.mp4")},
		{tmpDir, "output.mp4", []string{"", ""}, filepath.Join(tmpDir, "output.mp4")},
		{tmpDir, "output.mp4", []string{existingFile}, filepath.Join(tmpDir, "output.mp4")},
		{"/fallback", "output.mp4", []string{"", existingFile, ""}, filepath.Join(tmpDir, "output.mp4")},
		{"/fallback", "output.mp4", []string{"", ""}, filepath.Join("/fallback", "output.mp4")},
	}

	for _, tt := range tests {
		result := ArtifactPath(tt.fallbackDir, tt.fileName, tt.candidates...)
		assert.Equal(t, tt.expected, result)
	}
}

func TestReadTextFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	// Write file with trailing newline
	err := os.WriteFile(filePath, []byte("hello world\n"), 0644)
	require.NoError(t, err)

	content, err := ReadTextFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, "hello world", content)
}

func TestReadTextFile_NotFound(t *testing.T) {
	_, err := ReadTextFile("/nonexistent/file.txt")
	assert.Error(t, err)
}

func TestWriteTextArtifact(t *testing.T) {
	tmpDir := t.TempDir()

	path, err := WriteTextArtifact(tmpDir, "output", "source.txt", "hello world")
	require.NoError(t, err)

	expected := filepath.Join(tmpDir, "output-source.txt")
	assert.Equal(t, expected, path)

	content, err := ReadTextFile(path)
	require.NoError(t, err)
	assert.Equal(t, "hello world", content)
}

func TestWriteTextArtifact_PathFormat(t *testing.T) {
	tmpDir := t.TempDir()

	path, err := WriteTextArtifact(tmpDir, "output", "source.txt", "hello world")
	require.NoError(t, err)

	expected := filepath.Join(tmpDir, "output-source.txt")
	assert.Equal(t, expected, path)

	content, err := ReadTextFile(path)
	require.NoError(t, err)
	assert.Equal(t, "hello world", content)
}

func TestManager_NewManager(t *testing.T) {
	mgr := NewManager("/path/to/state.json")
	assert.NotNil(t, mgr)
	assert.Equal(t, "/path/to/state.json", mgr.statePath)
	assert.NotNil(t, mgr.State())
}

func TestManager_LoadState_ResetsOnSignatureChange(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Create initial state with signature "old-sig"
	s := New()
	s.Signature = "old-sig"
	s.SourceTextDone = true
	s.SourceTextPath = "/path/to/source.txt"
	err := s.Save(statePath)
	require.NoError(t, err)

	// Load with different signature should reset state
	mgr := NewManager(statePath)
	err = mgr.LoadState("new-sig")
	require.NoError(t, err)

	// State should be reset (no step done, empty paths)
	assert.False(t, mgr.State().SourceTextDone)
	assert.Empty(t, mgr.State().SourceTextPath)
	// Signature should be set to new signature
	assert.Equal(t, "new-sig", mgr.State().Signature)
}

func TestManager_LoadState_KeepsStateOnSameSignature(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Create initial state with signature "same-sig"
	s := New()
	s.Signature = "same-sig"
	s.SourceTextDone = true
	s.SourceTextPath = "/path/to/source.txt"
	err := s.Save(statePath)
	require.NoError(t, err)

	// Load with same signature should keep state
	mgr := NewManager(statePath)
	err = mgr.LoadState("same-sig")
	require.NoError(t, err)

	assert.Equal(t, "same-sig", mgr.State().Signature)
	assert.True(t, mgr.State().SourceTextDone)
	assert.Equal(t, "/path/to/source.txt", mgr.State().SourceTextPath)
}

func TestManager_LoadState_FileNotFound(t *testing.T) {
	mgr := NewManager("/nonexistent/path/state.json")
	err := mgr.LoadState("new-sig")
	require.NoError(t, err)
	// When file doesn't exist, Load returns empty state (no error)
	// Signature will be set by caller after LoadState
	assert.Empty(t, mgr.State().Signature)
}

func TestManager_ShouldSkip(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Create a real file for testing PathExists
	existingFile := filepath.Join(tmpDir, "video.mp4")
	f, err := os.Create(existingFile)
	require.NoError(t, err)
	f.Close()

	t.Run("not done, file doesn't exist", func(t *testing.T) {
		mgr := NewManager(statePath)
		skip, _ := mgr.ShouldSkip("download")
		assert.False(t, skip)
	})

	t.Run("done but file doesn't exist", func(t *testing.T) {
		mgr := NewManager(statePath)
		mgr.state.DownloadDone = true
		mgr.state.DownloadedPath = "/nonexistent/video.mp4"
		skip, _ := mgr.ShouldSkip("download")
		assert.False(t, skip)
	})

	t.Run("done and file exists", func(t *testing.T) {
		mgr := NewManager(statePath)
		mgr.state.CutDone = true
		mgr.state.CutVideoPath = existingFile
		skip, path := mgr.ShouldSkip("cut")
		assert.True(t, skip)
		assert.Equal(t, existingFile, path)
	})
}

func TestManager_CompleteStep(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	mgr := NewManager(statePath)
	mgr.state.Signature = "test-sig"

	err := mgr.CompleteStep("source_text", "/path/to/source.txt")
	require.NoError(t, err)

	assert.True(t, mgr.State().SourceTextDone)
	assert.Equal(t, "/path/to/source.txt", mgr.State().SourceTextPath)

	// Verify persisted state
	loaded, err := Load(statePath)
	require.NoError(t, err)
	assert.True(t, loaded.SourceTextDone)
	assert.Equal(t, "/path/to/source.txt", loaded.SourceTextPath)
}

func TestManager_CompleteStep_AllSteps(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	mgr := NewManager(statePath)
	mgr.state.Signature = "test-sig"

	steps := []struct {
		step         string
		artifactPath string
	}{
		{"source_text", "/path/source.txt"},
		{"summarized_text", "/path/summarized.txt"},
		{"download", "/path/video.mp4"},
		{"cut", "/path/cut.mp4"},
		{"audio", "/path/audio.mp3"},
		{"srt_subtitles", "/path/sub.srt"},
		{"subtitles", "/path/final.ass"},
		{"subtitles_burned", "/path/subtitled.mp4"},
		{"merge", "/path/final.mp4"},
	}

	for _, tt := range steps {
		err := mgr.CompleteStep(tt.step, tt.artifactPath)
		require.NoError(t, err)
		assert.True(t, mgr.State().IsStepDone(tt.step), "step: %s", tt.step)
		assert.Equal(t, tt.artifactPath, mgr.State().GetArtifact(tt.step), "step: %s", tt.step)
	}
}

func TestManager_Reset(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	mgr := NewManager(statePath)
	mgr.state.Signature = "old-sig"
	mgr.state.SourceTextDone = true
	mgr.state.SourceTextPath = "/path/source.txt"

	mgr.Reset("new-sig")

	assert.Equal(t, "new-sig", mgr.State().Signature)
	assert.False(t, mgr.State().SourceTextDone)
	assert.Empty(t, mgr.State().SourceTextPath)
}

func TestManager_Save(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	mgr := NewManager(statePath)
	mgr.state.Signature = "test-sig"
	mgr.state.DownloadDone = true
	mgr.state.DownloadedPath = "/path/video.mp4"

	err := mgr.Save()
	require.NoError(t, err)

	// Verify by loading directly
	loaded, err := Load(statePath)
	require.NoError(t, err)
	assert.Equal(t, "test-sig", loaded.Signature)
	assert.True(t, loaded.DownloadDone)
	assert.Equal(t, "/path/video.mp4", loaded.DownloadedPath)
}

func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{60, "60"},
		{3600, "3600"},
		{-1, "-1"},
	}

	for _, tt := range tests {
		result := Itoa(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

// TestJSONRoundTrip verifies that the JSON serialization is reversible
func TestPipelineState_JSONRoundTrip(t *testing.T) {
	original := &PipelineState{
		Signature:         "test-signature",
		SourceTextDone:   true,
		SourceTextPath:   "/path/to/source.txt",
		SummarisedTextDone: true,
		SummarisedTextPath: "/path/to/summarized.txt",
		DownloadDone:     true,
		DownloadedPath:   "/path/to/video.mp4",
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(original, "", "  ")
	require.NoError(t, err)

	// Deserialize back
	var loaded PipelineState
	err = json.Unmarshal(data, &loaded)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original.Signature, loaded.Signature)
	assert.Equal(t, original.SourceTextDone, loaded.SourceTextDone)
	assert.Equal(t, original.SourceTextPath, loaded.SourceTextPath)
	assert.Equal(t, original.SummarisedTextDone, loaded.SummarisedTextDone)
	assert.Equal(t, original.SummarisedTextPath, loaded.SummarisedTextPath)
	assert.Equal(t, original.DownloadDone, loaded.DownloadDone)
	assert.Equal(t, original.DownloadedPath, loaded.DownloadedPath)
}
package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anurag925/attari-video-gen/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	s := New()
	assert.NotNil(t, s)
	assert.Empty(t, s.Signature)
	assert.NotNil(t, s.Steps)
	assert.Empty(t, s.Steps)
}

func TestPipelineState_SaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	s := New()
	s.Signature = "test-signature"
	s.Steps["source_text"] = StepState{Done: true, ArtifactPath: "/path/to/source.txt"}

	err := s.Save(statePath)
	require.NoError(t, err)

	loaded, err := Load(statePath)
	require.NoError(t, err)
	assert.Equal(t, s.Signature, loaded.Signature)
	assert.True(t, loaded.IsStepDone(config.StepSourceText))
	assert.Equal(t, "/path/to/source.txt", loaded.GetArtifact(config.StepSourceText))
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

	// Initially no steps are done
	for _, step := range config.AllSteps() {
		assert.False(t, s.IsStepDone(step), "step %s should not be done initially", step)
	}

	// Mark source_text as done
	s.Steps["source_text"] = StepState{Done: true, ArtifactPath: "/path/to/source.txt"}
	assert.True(t, s.IsStepDone(config.StepSourceText))
	assert.False(t, s.IsStepDone(config.StepDownload))
}

func TestPipelineState_IsStepDone_UnknownStep(t *testing.T) {
	s := New()
	// Unknown steps should return false
	assert.False(t, s.IsStepDone("unknown_step"))
}

func TestPipelineState_SetStepDone(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	s := New()
	err := s.SetStepDone(statePath, config.StepSourceText, "/path/source.txt")
	require.NoError(t, err)
	assert.True(t, s.IsStepDone(config.StepSourceText))
	assert.Equal(t, "/path/source.txt", s.GetArtifact(config.StepSourceText))

	err = s.SetStepDone(statePath, config.StepDownload, "/path/video.mp4")
	require.NoError(t, err)
	assert.True(t, s.IsStepDone(config.StepDownload))
	assert.Equal(t, "/path/video.mp4", s.GetArtifact(config.StepDownload))
}

func TestPipelineState_GetArtifact(t *testing.T) {
	s := New()
	s.Steps["source_text"] = StepState{Done: true, ArtifactPath: "/path/source.txt"}
	s.Steps["download"] = StepState{Done: true, ArtifactPath: "/path/video.mp4"}

	assert.Equal(t, "/path/source.txt", s.GetArtifact(config.StepSourceText))
	assert.Equal(t, "/path/video.mp4", s.GetArtifact(config.StepDownload))
	assert.Empty(t, s.GetArtifact(config.StepCut))
}

func TestPipelineState_GetArtifact_UnknownStep(t *testing.T) {
	s := New()
	assert.Empty(t, s.GetArtifact("unknown_step"))
}

func TestPipelineState_SetArtifact(t *testing.T) {
	s := New()

	s.SetArtifact(config.StepSourceText, "/new/path/source.txt")
	assert.Equal(t, "/new/path/source.txt", s.GetArtifact(config.StepSourceText))

	s.SetArtifact(config.StepSubtitlesBurned, "/new/path/video.mp4")
	assert.Equal(t, "/new/path/video.mp4", s.GetArtifact(config.StepSubtitlesBurned))
}

func TestPipelineState_PathExists(t *testing.T) {
	tmpDir := t.TempDir()

	s := New()
	s.Steps["source_text"] = StepState{ArtifactPath: filepath.Join(tmpDir, "source.txt")}

	// File doesn't exist yet
	assert.False(t, s.PathExists(config.StepSourceText))

	// Create the file
	f, err := os.Create(s.GetArtifact(config.StepSourceText))
	require.NoError(t, err)
	f.Close()

	// Now it exists
	assert.True(t, s.PathExists(config.StepSourceText))
}

func TestPipelineState_PathExists_EmptyPath(t *testing.T) {
	s := New()
	assert.False(t, s.PathExists(config.StepSourceText))
}

func TestPipelineState_PathExists_UnknownStep(t *testing.T) {
	s := New()
	assert.False(t, s.PathExists("unknown_step"))
}

func TestPipelineState_Reset(t *testing.T) {
	s := New()
	s.Signature = "test-signature"
	s.Steps["source_text"] = StepState{Done: true, ArtifactPath: "/path/source.txt"}
	s.Steps["download"] = StepState{Done: true, ArtifactPath: "/path/video.mp4"}

	s.Reset()

	assert.Equal(t, "test-signature", s.Signature)
	assert.Empty(t, s.Steps)
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
	s.Steps["source_text"] = StepState{Done: true, ArtifactPath: "/path/to/source.txt"}
	err := s.Save(statePath)
	require.NoError(t, err)

	// Load with different signature should reset state
	mgr := NewManager(statePath)
	err = mgr.LoadState("new-sig")
	require.NoError(t, err)

	// State should be reset (no step done, empty paths)
	assert.False(t, mgr.State().IsStepDone(config.StepSourceText))
	assert.Empty(t, mgr.State().GetArtifact(config.StepSourceText))
	// Signature should be set to new signature
	assert.Equal(t, "new-sig", mgr.State().Signature)
}

func TestManager_LoadState_KeepsStateOnSameSignature(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Create initial state with signature "same-sig"
	s := New()
	s.Signature = "same-sig"
	s.Steps["source_text"] = StepState{Done: true, ArtifactPath: "/path/to/source.txt"}
	err := s.Save(statePath)
	require.NoError(t, err)

	// Load with same signature should keep state
	mgr := NewManager(statePath)
	err = mgr.LoadState("same-sig")
	require.NoError(t, err)

	assert.Equal(t, "same-sig", mgr.State().Signature)
	assert.True(t, mgr.State().IsStepDone(config.StepSourceText))
	assert.Equal(t, "/path/to/source.txt", mgr.State().GetArtifact(config.StepSourceText))
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
		skip, _ := mgr.ShouldSkip(config.StepDownload)
		assert.False(t, skip)
	})

	t.Run("done but file doesn't exist", func(t *testing.T) {
		mgr := NewManager(statePath)
		mgr.state.Steps["download"] = StepState{Done: true, ArtifactPath: "/nonexistent/video.mp4"}
		skip, _ := mgr.ShouldSkip(config.StepDownload)
		assert.False(t, skip)
	})

	t.Run("done and file exists", func(t *testing.T) {
		mgr := NewManager(statePath)
		mgr.state.Steps["cut"] = StepState{Done: true, ArtifactPath: existingFile}
		skip, path := mgr.ShouldSkip(config.StepCut)
		assert.True(t, skip)
		assert.Equal(t, existingFile, path)
	})
}

func TestManager_CompleteStep(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	mgr := NewManager(statePath)
	mgr.state.Signature = "test-sig"

	err := mgr.CompleteStep(config.StepSourceText, "/path/to/source.txt")
	require.NoError(t, err)

	assert.True(t, mgr.State().IsStepDone(config.StepSourceText))
	assert.Equal(t, "/path/to/source.txt", mgr.State().GetArtifact(config.StepSourceText))

	// Verify persisted state
	loaded, err := Load(statePath)
	require.NoError(t, err)
	assert.True(t, loaded.IsStepDone(config.StepSourceText))
	assert.Equal(t, "/path/to/source.txt", loaded.GetArtifact(config.StepSourceText))
}

func TestManager_CompleteStep_AllSteps(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	mgr := NewManager(statePath)
	mgr.state.Signature = "test-sig"

	steps := []struct {
		step         config.StepName
		artifactPath string
	}{
		{config.StepSourceText, "/path/source.txt"},
		{config.StepSummarizedText, "/path/summarized.txt"},
		{config.StepDownload, "/path/video.mp4"},
		{config.StepCut, "/path/cut.mp4"},
		{config.StepAudio, "/path/audio.mp3"},
		{config.StepSrtSubtitles, "/path/sub.srt"},
		{config.StepSubtitles, "/path/final.ass"},
		{config.StepSubtitlesBurned, "/path/subtitled.mp4"},
		{config.StepMerge, "/path/final.mp4"},
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
	mgr.state.Steps["source_text"] = StepState{Done: true, ArtifactPath: "/path/source.txt"}

	mgr.Reset("new-sig")

	assert.Equal(t, "new-sig", mgr.State().Signature)
	assert.False(t, mgr.State().IsStepDone(config.StepSourceText))
	assert.Empty(t, mgr.State().GetArtifact(config.StepSourceText))
}

func TestManager_Save(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	mgr := NewManager(statePath)
	mgr.state.Signature = "test-sig"
	mgr.state.Steps["download"] = StepState{Done: true, ArtifactPath: "/path/video.mp4"}

	err := mgr.Save()
	require.NoError(t, err)

	// Verify by loading directly
	loaded, err := Load(statePath)
	require.NoError(t, err)
	assert.Equal(t, "test-sig", loaded.Signature)
	assert.True(t, loaded.IsStepDone(config.StepDownload))
	assert.Equal(t, "/path/video.mp4", loaded.GetArtifact(config.StepDownload))
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

func TestPipelineState_JSONRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	original := &PipelineState{
		Signature: "test-signature",
		Steps: map[string]StepState{
			"source_text":      {Done: true, ArtifactPath: "/path/to/source.txt"},
			"summarized_text":   {Done: true, ArtifactPath: "/path/to/summarized.txt"},
			"download":         {Done: true, ArtifactPath: "/path/to/video.mp4"},
		},
	}

	// Serialize to JSON
	err := original.Save(statePath)
	require.NoError(t, err)

	// Deserialize back
	loaded, err := Load(statePath)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original.Signature, loaded.Signature)
	assert.True(t, loaded.IsStepDone(config.StepSourceText))
	assert.Equal(t, "/path/to/source.txt", loaded.GetArtifact(config.StepSourceText))
}
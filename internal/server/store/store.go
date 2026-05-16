package store

import (
	"sync"

	"github.com/anurag925/attari-video-gen/internal/server/models"
)

// Store is an in-memory store for pipelines and artifacts.
type Store struct {
	mu        sync.RWMutex
	pipelines map[string]*models.Pipeline
	artifacts  map[string]*models.Artifact
}

// New creates a new in-memory store.
func New() *Store {
	return &Store{
		pipelines: make(map[string]*models.Pipeline),
		artifacts:  make(map[string]*models.Artifact),
	}
}

// Pipeline operations

func (s *Store) ListPipelines() []*models.Pipeline {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*models.Pipeline, 0, len(s.pipelines))
	for _, p := range s.pipelines {
		result = append(result, p)
	}
	return result
}

func (s *Store) GetPipeline(id string) *models.Pipeline {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pipelines[id]
}

func (s *Store) SavePipeline(p *models.Pipeline) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pipelines[p.ID] = p
}

func (s *Store) DeletePipeline(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pipelines, id)
}

// Artifact operations

func (s *Store) ListArtifacts() []*models.Artifact {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*models.Artifact, 0, len(s.artifacts))
	for _, a := range s.artifacts {
		result = append(result, a)
	}
	return result
}

func (s *Store) GetArtifact(name string) *models.Artifact {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.artifacts[name]
}

func (s *Store) SaveArtifact(a *models.Artifact) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.artifacts[a.Name] = a
}

func (s *Store) DeleteArtifact(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.artifacts, name)
}

// FindArtifactsByPipeline returns all artifacts for a given pipeline ID.
func (s *Store) FindArtifactsByPipeline(pipelineID string) []*models.Artifact {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.Artifact
	for _, a := range s.artifacts {
		if a.PipelineID == pipelineID {
			result = append(result, a)
		}
	}
	return result
}

// FindArtifactsByType returns all artifacts of a given type.
func (s *Store) FindArtifactsByType(type_ string) []*models.Artifact {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.Artifact
	for _, a := range s.artifacts {
		if a.Type == type_ {
			result = append(result, a)
		}
	}
	return result
}
package ingest

import (
	"context"

	"github.com/Prasad-178/reps/internal/config"
	"github.com/Prasad-178/reps/internal/llm"
	"github.com/Prasad-178/reps/internal/store"
)

type Pipeline struct {
	cfg    config.Config
	store  *store.Store
	client *llm.Client
}

func NewPipeline(cfg config.Config, s *store.Store, c *llm.Client) *Pipeline {
	return &Pipeline{cfg: cfg, store: s, client: c}
}

// Sub-files fill in:
//   - IngestResume(ctx, path) (sourceID, error)        — resume.go
//   - IngestGitHub(ctx, user) (sourceID, error)        — github.go
//   - Rebuild(ctx) error                                — rebuild.go
//   - chunk + embed                                     — chunk.go
//   - profile synthesis                                 — synthesize.go

var _ = context.Background

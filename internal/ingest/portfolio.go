package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Prasad-178/reps/internal/store"
	"github.com/google/uuid"
)

func (p *Pipeline) IngestPortfolio(ctx context.Context, url string) (string, error) {
	text, err := fetchPage(ctx, url)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", url, err)
	}
	if len(text) < 200 {
		return "", fmt.Errorf("portfolio body too small (%d chars) — page may have blocked us", len(text))
	}
	id := uuid.NewString()
	rawPath := filepath.Join(p.cfg.Paths.Sources, id+".txt")
	if err := os.WriteFile(rawPath, []byte(text), 0o644); err != nil {
		return "", err
	}
	meta, _ := json.Marshal(map[string]any{"url": url, "chars": len(text)})
	return id, p.store.InsertSource(store.Source{
		ID: id, Kind: "portfolio", Ref: url, RawPath: rawPath,
		FetchedAt: time.Now(), MetaJSON: string(meta),
	})
}

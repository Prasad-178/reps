package ingest

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Prasad-178/reps/internal/store"
	"github.com/google/uuid"
)

func (p *Pipeline) IngestResume(ctx context.Context, path string) (string, error) {
	abs, err := filepath.Abs(expand(path))
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("resume not found: %s", abs)
	}
	if _, err := exec.LookPath("pdftotext"); err != nil {
		return "", fmt.Errorf("`pdftotext` not on PATH — install poppler (brew install poppler)")
	}
	text, err := pdfToText(ctx, abs)
	if err != nil {
		return "", err
	}
	id := uuid.NewString()
	rawPath := filepath.Join(p.cfg.Paths.Sources, id+".txt")
	if err := os.WriteFile(rawPath, []byte(text), 0o644); err != nil {
		return "", err
	}
	return id, p.store.InsertSource(store.Source{
		ID: id, Kind: "resume_pdf", Ref: abs, RawPath: rawPath,
		FetchedAt: time.Now(),
	})
}

func pdfToText(ctx context.Context, path string) (string, error) {
	cmd := exec.CommandContext(ctx, "pdftotext", "-layout", "-nopgbrk", "-enc", "UTF-8", path, "-")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext: %w", err)
	}
	return string(out), nil
}

func expand(p string) string {
	if len(p) >= 2 && p[:2] == "~/" {
		if h, err := os.UserHomeDir(); err == nil {
			return filepath.Join(h, p[2:])
		}
	}
	return p
}

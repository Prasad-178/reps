package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Prasad-178/reps/internal/store"
	"github.com/google/uuid"
)

func (p *Pipeline) IngestLinkedIn(ctx context.Context, ref, fromFile string) (string, error) {
	return p.ingestPaste(ctx, "linkedin", ref, fromFile)
}

func (p *Pipeline) IngestX(ctx context.Context, handle, fromFile string) (string, error) {
	return p.ingestPaste(ctx, "x", handle, fromFile)
}

func (p *Pipeline) IngestNote(ctx context.Context, path string) (string, error) {
	abs, err := filepath.Abs(expand(path))
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", fmt.Errorf("read note %s: %w", abs, err)
	}
	body := strings.TrimSpace(string(b))
	if body == "" {
		return "", fmt.Errorf("note is empty")
	}
	id := uuid.NewString()
	rawPath := filepath.Join(p.cfg.Paths.Sources, id+".txt")
	if err := os.WriteFile(rawPath, []byte(body), 0o644); err != nil {
		return "", err
	}
	return id, p.store.InsertSource(store.Source{
		ID: id, Kind: "note", Ref: abs, RawPath: rawPath,
		FetchedAt: time.Now(),
	})
}

func (p *Pipeline) ingestPaste(_ context.Context, kind, ref, fromFile string) (string, error) {
	body, err := readPaste(fromFile)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(body) == "" {
		return "", fmt.Errorf("paste was empty")
	}
	id := uuid.NewString()
	rawPath := filepath.Join(p.cfg.Paths.Sources, id+".txt")
	if err := os.WriteFile(rawPath, []byte(body), 0o644); err != nil {
		return "", err
	}
	meta, _ := json.Marshal(map[string]any{"chars": len(body), "via": pasteSource(fromFile)})
	return id, p.store.InsertSource(store.Source{
		ID: id, Kind: kind, Ref: ref, RawPath: rawPath,
		FetchedAt: time.Now(), MetaJSON: string(meta),
	})
}

func readPaste(fromFile string) (string, error) {
	if fromFile != "" {
		b, err := os.ReadFile(expand(fromFile))
		if err != nil {
			return "", fmt.Errorf("read %s: %w", fromFile, err)
		}
		return string(b), nil
	}
	if isStdinPiped() {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	fmt.Println("Paste content below. End with Ctrl-D on a blank line.")
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func isStdinPiped() bool {
	st, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (st.Mode() & os.ModeCharDevice) == 0
}

func pasteSource(fromFile string) string {
	if fromFile != "" {
		return "file"
	}
	return "stdin"
}

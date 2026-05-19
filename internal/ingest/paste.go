package ingest

import (
	"context"
	"encoding/json"
	"errors"
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
	looksLikeProfile := strings.Contains(ref, "linkedin.com/in/") ||
		(!strings.Contains(ref, "/") && !strings.Contains(ref, " "))

	if looksLikeProfile {
		// 1. ScrapingDog (recommended; 1000 free credits, simple REST)
		if body, err := FetchLinkedInScrapingDog(ctx, ref); err == nil {
			return p.saveLinkedInBlob(ref, body, "scrapingdog")
		} else if !errors.Is(err, ErrNoScrapingDogKey) {
			fmt.Printf("  scrapingdog failed: %v — trying next provider\n", err)
		}

		// 2. Proxycurl / Enrichlayer (legacy, if key is set)
		if body, err := FetchLinkedInProxycurl(ctx, ref); err == nil {
			return p.saveLinkedInBlob(ref, body, "proxycurl")
		} else if !errors.Is(err, ErrNoProxycurlKey) {
			fmt.Printf("  proxycurl failed: %v — trying next provider\n", err)
		}
	}

	// 3. Caller supplied a paste file or piped stdin → use that
	if fromFile != "" || isStdinPiped() {
		return p.ingestPaste(ctx, "linkedin", ref, fromFile)
	}

	// 4. Try Jina Reader for the URL (last free attempt before asking to paste)
	if looksLikeProfile && strings.HasPrefix(ref, "http") {
		md, jerr := ReadURL(ctx, ref)
		if jerr == nil && looksLikeRealProfile(md) {
			return p.saveLinkedInBlob(ref, md, "jina-reader")
		}
	}

	// 5. Last resort: ask the user to paste.
	return p.ingestPaste(ctx, "linkedin", ref, fromFile)
}

// looksLikeRealProfile rejects login-walls + redirects that come back from
// Jina when LinkedIn refuses to render the public profile.
func looksLikeRealProfile(md string) bool {
	if len(md) < 800 {
		return false
	}
	lower := strings.ToLower(md)
	for _, bad := range []string{
		"sign in to view",
		"join now",
		"this content isn't available",
		"the page you’re looking for",
		"<title>linkedin login",
	} {
		if strings.Contains(lower, bad) {
			return false
		}
	}
	return true
}

func (p *Pipeline) saveLinkedInBlob(ref, body, via string) (string, error) {
	id := uuid.NewString()
	rawPath := filepath.Join(p.cfg.Paths.Sources, id+".txt")
	if err := os.WriteFile(rawPath, []byte(body), 0o644); err != nil {
		return "", err
	}
	meta, _ := json.Marshal(map[string]any{"chars": len(body), "via": via})
	return id, p.store.InsertSource(store.Source{
		ID: id, Kind: "linkedin", Ref: ref, RawPath: rawPath,
		FetchedAt: time.Now(), MetaJSON: string(meta),
	})
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
	fmt.Println()
	fmt.Println("Paste the page content below (Cmd-A → Cmd-C on the source tab,")
	fmt.Println("then Cmd-V here). Press ENTER then Ctrl-D on a blank line when done.")
	fmt.Println("Tip: set SCRAPINGDOG_API_KEY to auto-fetch LinkedIn from now on.")
	fmt.Println()
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

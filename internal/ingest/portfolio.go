package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Prasad-178/reps/internal/store"
	"github.com/google/uuid"
)

// IngestPortfolio dispatches based on what `ref` looks like:
//
//   - http(s)://…   → fetch via Jina Reader, then crawl up to N same-origin
//     pages (sitemap.xml first, then anchor follow).
//   - /abs/path or relative path  → walk the directory, read every text /
//     markdown / html / rst / org file under it (size-capped).
//
// The result is concatenated into one source row with kind="portfolio".
// A logger callback is accepted so the caller (CLI / web ingestion API)
// can surface per-page progress.
func (p *Pipeline) IngestPortfolio(ctx context.Context, ref string) (string, error) {
	return p.ingestPortfolioWithLog(ctx, ref, func(s string) {
		fmt.Println("  " + s)
	})
}

func (p *Pipeline) IngestPortfolioWithLog(ctx context.Context, ref string, log func(string)) (string, error) {
	return p.ingestPortfolioWithLog(ctx, ref, log)
}

func (p *Pipeline) ingestPortfolioWithLog(ctx context.Context, ref string, log func(string)) (string, error) {
	if log == nil {
		log = func(string) {}
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("empty portfolio ref")
	}

	if isURL(ref) {
		return p.ingestPortfolioURL(ctx, ref, log)
	}
	return p.ingestPortfolioFolder(ctx, ref, log)
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// ---- URL flow: Jina Reader + crawl

func (p *Pipeline) ingestPortfolioURL(ctx context.Context, urlRef string, log func(string)) (string, error) {
	log("crawl: discovering pages…")
	pages, err := CrawlSite(ctx, urlRef, defaultMaxPages, log)
	if err != nil {
		return "", err
	}
	log(fmt.Sprintf("fetched %d page(s)", len(pages)))

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Portfolio site: %s\n", urlRef)
	fmt.Fprintf(&sb, "Crawled %d page(s) via Jina Reader.\n\n", len(pages))
	for _, d := range pages {
		fmt.Fprintf(&sb, "\n--- %s ---\n", d.URL)
		body := d.Markdown
		if len(body) > 24000 {
			body = body[:24000] + "\n…(truncated)…"
		}
		sb.WriteString(body)
		sb.WriteString("\n")
	}

	id := uuid.NewString()
	rawPath := filepath.Join(p.cfg.Paths.Sources, id+".txt")
	if err := os.WriteFile(rawPath, []byte(sb.String()), 0o644); err != nil {
		return "", err
	}
	meta, _ := json.Marshal(map[string]any{
		"url": urlRef, "pages": len(pages), "chars": sb.Len(), "via": "jina-reader",
	})
	return id, p.store.InsertSource(store.Source{
		ID: id, Kind: "portfolio", Ref: urlRef, RawPath: rawPath,
		FetchedAt: time.Now(), MetaJSON: string(meta),
	})
}

// ---- folder flow: walk + concat

func (p *Pipeline) ingestPortfolioFolder(ctx context.Context, path string, log func(string)) (string, error) {
	abs, err := filepath.Abs(expand(path))
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("not found: %s", abs)
	}
	log(fmt.Sprintf("folder: scanning %s", abs))
	docs, err := WalkFolder(abs, log)
	if err != nil {
		return "", err
	}
	log(fmt.Sprintf("scanned %d file(s)", len(docs)))

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Portfolio folder: %s\n", abs)
	fmt.Fprintf(&sb, "Read %d file(s).\n\n", len(docs))
	for _, d := range docs {
		fmt.Fprintf(&sb, "\n--- %s ---\n", d.Path)
		sb.WriteString(d.Content)
		sb.WriteString("\n")
	}

	_ = ctx // kept for parity with URL variant
	id := uuid.NewString()
	rawPath := filepath.Join(p.cfg.Paths.Sources, id+".txt")
	if err := os.WriteFile(rawPath, []byte(sb.String()), 0o644); err != nil {
		return "", err
	}
	meta, _ := json.Marshal(map[string]any{
		"root": abs, "files": len(docs), "chars": sb.Len(), "via": "folder-walk",
	})
	return id, p.store.InsertSource(store.Source{
		ID: id, Kind: "portfolio", Ref: abs, RawPath: rawPath,
		FetchedAt: time.Now(), MetaJSON: string(meta),
	})
}

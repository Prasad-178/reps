package ingest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ReadURL fetches the content of `url` via Jina Reader (r.jina.ai), which
// returns clean markdown for any web page. Handles JS-rendered pages. No
// browser needed.
//
// Optional: set JINA_API_KEY for higher rate limits + better quality.
// Without a key, Jina Reader is still free and works for low-volume use.
//
// Docs: https://jina.ai/reader
func ReadURL(ctx context.Context, url string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("empty url")
	}
	endpoint := "https://r.jina.ai/" + url

	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, "GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/plain")
	req.Header.Set("X-Return-Format", "markdown")
	if key := os.Getenv("JINA_API_KEY"); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode == 402 {
		return "", fmt.Errorf("jina reader rate-limited (402) — set JINA_API_KEY for higher quota")
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("jina reader http %d: %s", resp.StatusCode, snip(string(body), 200))
	}
	text := strings.TrimSpace(string(body))
	if len(text) < 80 {
		return "", fmt.Errorf("jina reader returned suspiciously little content (%d chars)", len(text))
	}
	return text, nil
}

func snip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

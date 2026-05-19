package ingest

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	defaultMaxPages = 12
	sitemapTimeout  = 12 * time.Second
)

// CrawlSite returns up to maxPages of same-origin pages starting from rootURL,
// each rendered as markdown via ReadURL. Strategy:
//
//  1. Try /sitemap.xml — fast, authoritative, gets us a deduped URL list.
//  2. Fall back to fetching the root with ReadURL and scraping its anchor
//     hrefs that point to the same origin.
//
// Results are returned in the order they were discovered. Errors on
// individual pages are logged via `log` and skipped.
func CrawlSite(ctx context.Context, rootURL string, maxPages int, log func(string)) ([]PageDoc, error) {
	if maxPages <= 0 {
		maxPages = defaultMaxPages
	}
	root, err := url.Parse(rootURL)
	if err != nil {
		return nil, err
	}
	if root.Scheme == "" || root.Host == "" {
		return nil, fmt.Errorf("invalid url: %s", rootURL)
	}

	visited := map[string]bool{}
	queue := []string{rootURL}

	// Sitemap discovery
	if smURLs, ok := fetchSitemap(ctx, root); ok {
		log(fmt.Sprintf("sitemap.xml → %d urls", len(smURLs)))
		for _, u := range smURLs {
			if sameOrigin(root, u) && !visited[u] {
				queue = append(queue, u)
				visited[u] = true
				if len(queue) >= maxPages {
					break
				}
			}
		}
	}

	var out []PageDoc
	for i := 0; i < len(queue) && len(out) < maxPages; i++ {
		u := queue[i]
		if visited[u] && i > 0 {
			// already counted via sitemap; skip duplicates
		}
		visited[u] = true

		md, err := ReadURL(ctx, u)
		if err != nil {
			log(fmt.Sprintf("skip %s — %v", u, err))
			continue
		}
		out = append(out, PageDoc{URL: u, Markdown: md})

		// On root page (first one), discover more same-origin links to keep crawling
		if i == 0 && len(queue) < maxPages {
			for _, link := range extractLinks(md, root) {
				if !visited[link] && sameOrigin(root, link) {
					queue = append(queue, link)
					visited[link] = true
					if len(queue) >= maxPages {
						break
					}
				}
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no pages crawled (jina failed or robots-blocked)")
	}
	return out, nil
}

type PageDoc struct {
	URL      string
	Markdown string
}

// ---- sitemap.xml fetcher

type sitemapURLSet struct {
	XMLName xml.Name `xml:"urlset"`
	URL     []struct {
		Loc string `xml:"loc"`
	} `xml:"url"`
}

type sitemapIndex struct {
	XMLName xml.Name `xml:"sitemapindex"`
	Sitemap []struct {
		Loc string `xml:"loc"`
	} `xml:"sitemap"`
}

func fetchSitemap(ctx context.Context, root *url.URL) ([]string, bool) {
	base := root.Scheme + "://" + root.Host
	candidates := []string{
		base + "/sitemap.xml",
		base + "/sitemap_index.xml",
		base + "/sitemap.xml.gz",
	}
	for _, c := range candidates {
		urls, ok := fetchSitemapURL(ctx, c)
		if ok {
			return urls, true
		}
	}
	return nil, false
}

func fetchSitemapURL(ctx context.Context, sitemapURL string) ([]string, bool) {
	cctx, cancel := context.WithTimeout(ctx, sitemapTimeout)
	defer cancel()
	req, _ := http.NewRequestWithContext(cctx, "GET", sitemapURL, nil)
	req.Header.Set("User-Agent", "reps-ingest/0.1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			resp.Body.Close()
		}
		return nil, false
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 5<<20))

	// sitemap index?
	var idx sitemapIndex
	if err := xml.Unmarshal(body, &idx); err == nil && len(idx.Sitemap) > 0 {
		var out []string
		for _, sm := range idx.Sitemap {
			if u, ok := fetchSitemapURL(ctx, sm.Loc); ok {
				out = append(out, u...)
				if len(out) > 200 {
					break
				}
			}
		}
		if len(out) > 0 {
			return out, true
		}
	}

	// plain urlset
	var set sitemapURLSet
	if err := xml.Unmarshal(body, &set); err != nil {
		return nil, false
	}
	out := make([]string, 0, len(set.URL))
	for _, u := range set.URL {
		if u.Loc != "" {
			out = append(out, strings.TrimSpace(u.Loc))
		}
	}
	return out, len(out) > 0
}

// ---- link extraction from markdown (anchors of form [text](href))

var mdLinkRe = regexp.MustCompile(`\[[^\]]+\]\(([^)\s]+)`)

func extractLinks(md string, base *url.URL) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range mdLinkRe.FindAllStringSubmatch(md, -1) {
		href := m[1]
		if href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "mailto:") {
			continue
		}
		u, err := url.Parse(href)
		if err != nil {
			continue
		}
		resolved := base.ResolveReference(u).String()
		// strip fragment, normalise
		if i := strings.Index(resolved, "#"); i >= 0 {
			resolved = resolved[:i]
		}
		resolved = strings.TrimSuffix(resolved, "/")
		if seen[resolved] || resolved == strings.TrimSuffix(base.String(), "/") {
			continue
		}
		seen[resolved] = true
		out = append(out, resolved)
	}
	return out
}

func sameOrigin(base *url.URL, raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Host, base.Host)
}

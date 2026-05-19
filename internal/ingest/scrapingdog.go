package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// FetchLinkedInScrapingDog pulls a public LinkedIn profile via ScrapingDog's
// hosted API. ScrapingDog is a stable Proxycurl alternative — 1000 free
// credits to start, 50 credits / profile ≈ $0.005 / profile after that.
//
// Endpoint: https://api.scrapingdog.com/linkedin/?api_key=…&type=profile&linkId=<handle>
// Docs:     https://docs.scrapingdog.com/linkedin-scraper-api
//
// Returns ErrNoScrapingDogKey if SCRAPINGDOG_API_KEY isn't set.
func FetchLinkedInScrapingDog(ctx context.Context, profileURL string) (string, error) {
	key := strings.TrimSpace(os.Getenv("SCRAPINGDOG_API_KEY"))
	if key == "" {
		return "", ErrNoScrapingDogKey
	}
	handle, err := linkedInHandle(profileURL)
	if err != nil {
		return "", err
	}

	q := url.Values{}
	q.Set("api_key", key)
	q.Set("type", "profile")
	q.Set("linkId", handle)
	q.Set("private", "true")
	endpoint := "https://api.scrapingdog.com/linkedin/?" + q.Encode()

	// ScrapingDog may return 202 while scraping; retry a few times.
	for attempt := 0; attempt < 4; attempt++ {
		cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		req, _ := http.NewRequestWithContext(cctx, "GET", endpoint, nil)
		resp, err := http.DefaultClient.Do(req)
		cancel()
		if err != nil {
			return "", err
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
		resp.Body.Close()

		switch {
		case resp.StatusCode == 200:
			return scrapingDogToMarkdown(profileURL, body), nil
		case resp.StatusCode == 202:
			time.Sleep(time.Duration(2+attempt) * time.Second)
			continue
		case resp.StatusCode == 401 || resp.StatusCode == 403:
			return "", fmt.Errorf("scrapingdog auth failed (%d) — check SCRAPINGDOG_API_KEY", resp.StatusCode)
		case resp.StatusCode == 402:
			return "", fmt.Errorf("scrapingdog: out of credits")
		default:
			return "", fmt.Errorf("scrapingdog http %d: %s", resp.StatusCode, snip(string(body), 200))
		}
	}
	return "", fmt.Errorf("scrapingdog: timed out waiting for profile to be scraped")
}

var ErrNoScrapingDogKey = fmt.Errorf("SCRAPINGDOG_API_KEY not set")

// FetchXProfileScrapingDog pulls an X (Twitter) profile via ScrapingDog.
// Endpoint: https://api.scrapingdog.com/x/profile?api_key=…&profileId=<handle>
// Cost: 5 credits per profile.
//
// Returns ErrNoScrapingDogKey when the env var is unset.
func FetchXProfileScrapingDog(ctx context.Context, handleOrURL string) (string, error) {
	key := strings.TrimSpace(os.Getenv("SCRAPINGDOG_API_KEY"))
	if key == "" {
		return "", ErrNoScrapingDogKey
	}
	handle := xHandle(handleOrURL)
	if handle == "" {
		return "", fmt.Errorf("can't extract X handle from %q", handleOrURL)
	}

	q := url.Values{}
	q.Set("api_key", key)
	q.Set("profileId", handle)
	endpoint := "https://api.scrapingdog.com/x/profile?" + q.Encode()

	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(cctx, "GET", endpoint, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 5<<20))

	switch {
	case resp.StatusCode == 200:
		return scrapingDogXToMarkdown(handle, body), nil
	case resp.StatusCode == 401 || resp.StatusCode == 403:
		return "", fmt.Errorf("scrapingdog auth failed (%d) — check SCRAPINGDOG_API_KEY", resp.StatusCode)
	case resp.StatusCode == 402:
		return "", fmt.Errorf("scrapingdog: out of credits")
	default:
		return "", fmt.Errorf("scrapingdog x/profile http %d: %s", resp.StatusCode, snip(string(body), 200))
	}
}

// xHandle pulls the handle from any of:
//
//	@prasadjs178
//	prasadjs178
//	https://x.com/prasadjs178
//	https://twitter.com/prasadjs178/status/123
func xHandle(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "@")
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "/") {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return ""
	}
	return parts[0]
}

func scrapingDogXToMarkdown(handle string, raw []byte) string {
	var sb strings.Builder
	var top map[string]any
	if err := json.Unmarshal(raw, &top); err != nil {
		// Couldn't decode — surface raw for the LLM.
		fmt.Fprintf(&sb, "_Source: x.com/%s (via ScrapingDog)_\n\n```\n", handle)
		sb.Write(raw)
		sb.WriteString("\n```\n")
		return sb.String()
	}
	u, _ := top["user"].(map[string]any)
	if u == nil {
		u = top // some responses are flat
	}

	name := firstString(u, "profile_name", "name")
	bio := firstString(u, "description", "bio")
	loc := firstString(u, "location")
	verified, _ := u["is_blue_verified"].(bool)

	fmt.Fprintf(&sb, "# %s", name)
	if verified {
		sb.WriteString(" ✓")
	}
	sb.WriteString("\n")
	fmt.Fprintf(&sb, "@%s\n", handle)
	fmt.Fprintf(&sb, "_Source: https://x.com/%s (via ScrapingDog)_\n\n", handle)

	if bio != "" {
		sb.WriteString("## Bio\n")
		sb.WriteString(bio)
		sb.WriteString("\n\n")
	}

	if loc != "" {
		fmt.Fprintf(&sb, "Location: %s\n\n", loc)
	}

	// metrics
	type metric struct {
		Label string
		Keys  []string
	}
	metrics := []metric{
		{"Followers", []string{"followers_count"}},
		{"Following", []string{"following_count"}},
		{"Posts", []string{"statuses_count", "tweets_count"}},
		{"Likes", []string{"likes_count", "favourites_count"}},
		{"Media", []string{"media_count"}},
		{"Listed", []string{"listed_count"}},
	}
	hasAny := false
	for _, m := range metrics {
		if v := numField(u, m.Keys...); v != "" {
			if !hasAny {
				sb.WriteString("## Metrics\n")
				hasAny = true
			}
			fmt.Fprintf(&sb, "- %s: %s\n", m.Label, v)
		}
	}
	if hasAny {
		sb.WriteString("\n")
	}

	// always include the raw user JSON so the agent sees fields we didn't map
	sb.WriteString("## Raw\n```json\n")
	pretty, _ := json.MarshalIndent(u, "", "  ")
	sb.Write(pretty)
	sb.WriteString("\n```\n")
	return sb.String()
}

func numField(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case float64:
				return fmt.Sprintf("%d", int64(t))
			case int:
				return fmt.Sprintf("%d", t)
			case string:
				if t != "" {
					return t
				}
			}
		}
	}
	return ""
}

// linkedInHandle extracts the slug from any of these forms:
//
//	https://www.linkedin.com/in/prasadsankarc/
//	https://linkedin.com/in/prasadsankarc?foo=bar
//	prasadsankarc
func linkedInHandle(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty linkedin ref")
	}
	if !strings.Contains(raw, "/") && !strings.Contains(raw, "linkedin.com") {
		return raw, nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i, p := range parts {
		if p == "in" && i+1 < len(parts) {
			return parts[i+1], nil
		}
	}
	return "", fmt.Errorf("can't extract linkedin handle from %s", raw)
}

// scrapingDogToMarkdown renders a *best-effort* markdown profile from
// ScrapingDog's raw JSON. The shape varies — we walk it generically and
// always include the raw JSON at the bottom so nothing is lost.
func scrapingDogToMarkdown(profileURL string, raw []byte) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "_Source: %s (via ScrapingDog)_\n\n", profileURL)

	// ScrapingDog returns either a single object or an array containing one.
	var arr []map[string]any
	var obj map[string]any
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		obj = arr[0]
	} else {
		_ = json.Unmarshal(raw, &obj)
	}
	if obj == nil {
		// Couldn't decode — embed the raw text and let the LLM make sense of it.
		sb.WriteString("```\n")
		sb.WriteString(string(raw))
		sb.WriteString("\n```\n")
		return sb.String()
	}

	name := firstString(obj, "fullName", "name", "full_name")
	if name != "" {
		fmt.Fprintf(&sb, "# %s\n", name)
	}
	if h := firstString(obj, "headline", "title", "occupation"); h != "" {
		fmt.Fprintf(&sb, "%s\n", h)
	}
	if loc := firstString(obj, "location", "addressWithCountry", "geographic_location"); loc != "" {
		fmt.Fprintf(&sb, "_%s_\n", loc)
	}
	sb.WriteString("\n")

	if about := firstString(obj, "about", "summary", "description"); about != "" {
		sb.WriteString("## About\n")
		sb.WriteString(about)
		sb.WriteString("\n\n")
	}

	renderList(&sb, obj, "## Experience", []string{"experience", "experiences", "positions"}, []string{"title", "position"}, []string{"companyName", "company"}, []string{"description", "summary"})
	renderList(&sb, obj, "## Education", []string{"education", "educations"}, []string{"degree", "fieldOfStudy", "field_of_study"}, []string{"schoolName", "school"}, []string{"description"})
	renderList(&sb, obj, "## Projects", []string{"projects"}, []string{"name", "title"}, nil, []string{"description"})
	renderList(&sb, obj, "## Publications", []string{"publications"}, []string{"name", "title"}, []string{"publisher"}, []string{"description"})

	if skills := firstAnyList(obj, "skills"); len(skills) > 0 {
		sb.WriteString("## Skills\n")
		for _, s := range skills {
			if v, _ := s.(string); v != "" {
				fmt.Fprintf(&sb, "- %s\n", v)
			} else if m, ok := s.(map[string]any); ok {
				if v := firstString(m, "name", "title"); v != "" {
					fmt.Fprintf(&sb, "- %s\n", v)
				}
			}
		}
		sb.WriteString("\n")
	}

	// Always include raw JSON for the LLM to dig into.
	sb.WriteString("## Raw\n```json\n")
	pretty, _ := json.MarshalIndent(obj, "", "  ")
	sb.Write(pretty)
	sb.WriteString("\n```\n")
	return sb.String()
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

func firstAnyList(m map[string]any, keys ...string) []any {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if l, ok := v.([]any); ok {
				return l
			}
		}
	}
	return nil
}

func renderList(sb *strings.Builder, obj map[string]any, heading string, keys []string, titleKeys []string, subKeys []string, bodyKeys []string) {
	items := firstAnyList(obj, keys...)
	if len(items) == 0 {
		return
	}
	sb.WriteString(heading)
	sb.WriteString("\n")
	for _, it := range items {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		title := firstString(m, titleKeys...)
		sub := ""
		if subKeys != nil {
			sub = firstString(m, subKeys...)
		}
		body := firstString(m, bodyKeys...)
		switch {
		case title != "" && sub != "":
			fmt.Fprintf(sb, "- **%s** at **%s**", title, sub)
		case title != "":
			fmt.Fprintf(sb, "- **%s**", title)
		case sub != "":
			fmt.Fprintf(sb, "- %s", sub)
		default:
			continue
		}
		// optional date range
		from := firstString(m, "startDate", "starts_at", "from")
		to := firstString(m, "endDate", "ends_at", "to")
		if from != "" || to != "" {
			if to == "" {
				to = "present"
			}
			fmt.Fprintf(sb, " (%s–%s)", from, to)
		}
		sb.WriteString("\n")
		if body != "" {
			fmt.Fprintf(sb, "  %s\n", strings.ReplaceAll(body, "\n", " "))
		}
	}
	sb.WriteString("\n")
}

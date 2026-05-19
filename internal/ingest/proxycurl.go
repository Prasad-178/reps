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

// FetchLinkedInProxycurl pulls a LinkedIn profile via Proxycurl's hosted API.
// LinkedIn aggressively blocks scrapers, so we use a paid API service that
// solves the auth + anti-bot problem properly. Costs ~$0.01/profile.
//
// Requires PROXYCURL_API_KEY env var. Returns a markdown-style blob ready to
// store as a normal source. Returns ErrNoProxycurlKey if the key is missing,
// which lets the caller fall back to manual paste.
//
// API: https://nubela.co/proxycurl/docs#people-api-person-profile-endpoint
func FetchLinkedInProxycurl(ctx context.Context, profileURL string) (string, error) {
	key := strings.TrimSpace(os.Getenv("PROXYCURL_API_KEY"))
	if key == "" {
		return "", ErrNoProxycurlKey
	}
	q := url.Values{}
	q.Set("url", profileURL)
	q.Set("use_cache", "if-present")
	endpoint := "https://nubela.co/proxycurl/api/v2/linkedin?" + q.Encode()

	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, "GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 5<<20))

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return "", fmt.Errorf("proxycurl auth failed (%d) — check PROXYCURL_API_KEY", resp.StatusCode)
	}
	if resp.StatusCode == 402 {
		return "", fmt.Errorf("proxycurl: payment required / out of credits")
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("proxycurl http %d: %s", resp.StatusCode, snip(string(body), 200))
	}

	var profile proxycurlProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return "", fmt.Errorf("proxycurl decode: %w", err)
	}
	return profile.toMarkdown(profileURL), nil
}

var ErrNoProxycurlKey = fmt.Errorf("PROXYCURL_API_KEY not set")

type proxycurlProfile struct {
	FullName    string `json:"full_name"`
	Headline    string `json:"headline"`
	Summary     string `json:"summary"`
	Country     string `json:"country"`
	CountryFullName string `json:"country_full_name"`
	City        string `json:"city"`
	OccupationLabel string `json:"occupation"`

	Experiences []struct {
		Title       string `json:"title"`
		Company     string `json:"company"`
		Description string `json:"description"`
		StartsAt    *struct{ Year int `json:"year"` } `json:"starts_at"`
		EndsAt      *struct{ Year int `json:"year"` } `json:"ends_at"`
		Location    string `json:"location"`
	} `json:"experiences"`

	Education []struct {
		School      string `json:"school"`
		Degree      string `json:"degree_name"`
		Field       string `json:"field_of_study"`
		Description string `json:"description"`
	} `json:"education"`

	AccomplishmentProjects []struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	} `json:"accomplishment_projects"`

	AccomplishmentPublications []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Publisher   string `json:"publisher"`
	} `json:"accomplishment_publications"`

	Skills []string `json:"skills"`

	Certifications []struct {
		Name string `json:"name"`
		Authority string `json:"authority"`
	} `json:"certifications"`
}

func (p proxycurlProfile) toMarkdown(srcURL string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# %s\n", p.FullName)
	if p.Headline != "" {
		fmt.Fprintf(&sb, "%s\n", p.Headline)
	}
	if loc := strings.TrimSpace(strings.Join([]string{p.City, p.CountryFullName}, ", ")); loc != ", " && loc != "" {
		fmt.Fprintf(&sb, "_%s_\n", loc)
	}
	fmt.Fprintf(&sb, "_Source: %s (via Proxycurl)_\n\n", srcURL)

	if p.Summary != "" {
		sb.WriteString("## Summary\n")
		sb.WriteString(p.Summary)
		sb.WriteString("\n\n")
	}

	if len(p.Experiences) > 0 {
		sb.WriteString("## Experience\n")
		for _, e := range p.Experiences {
			fmt.Fprintf(&sb, "- **%s** at **%s**", e.Title, e.Company)
			if e.StartsAt != nil {
				fmt.Fprintf(&sb, " (%d", e.StartsAt.Year)
				if e.EndsAt != nil && e.EndsAt.Year > 0 {
					fmt.Fprintf(&sb, "–%d", e.EndsAt.Year)
				} else {
					sb.WriteString("–present")
				}
				sb.WriteString(")")
			}
			if e.Location != "" {
				fmt.Fprintf(&sb, " — %s", e.Location)
			}
			sb.WriteString("\n")
			if e.Description != "" {
				fmt.Fprintf(&sb, "  %s\n", strings.ReplaceAll(e.Description, "\n", " "))
			}
		}
		sb.WriteString("\n")
	}

	if len(p.Education) > 0 {
		sb.WriteString("## Education\n")
		for _, e := range p.Education {
			fmt.Fprintf(&sb, "- %s — %s %s\n", e.School, e.Degree, e.Field)
			if e.Description != "" {
				fmt.Fprintf(&sb, "  %s\n", strings.ReplaceAll(e.Description, "\n", " "))
			}
		}
		sb.WriteString("\n")
	}

	if len(p.AccomplishmentProjects) > 0 {
		sb.WriteString("## Projects\n")
		for _, pr := range p.AccomplishmentProjects {
			fmt.Fprintf(&sb, "- **%s** — %s\n", pr.Title, strings.ReplaceAll(pr.Description, "\n", " "))
		}
		sb.WriteString("\n")
	}
	if len(p.AccomplishmentPublications) > 0 {
		sb.WriteString("## Publications\n")
		for _, x := range p.AccomplishmentPublications {
			fmt.Fprintf(&sb, "- **%s** (%s) — %s\n", x.Name, x.Publisher, strings.ReplaceAll(x.Description, "\n", " "))
		}
		sb.WriteString("\n")
	}
	if len(p.Skills) > 0 {
		sb.WriteString("## Skills\n")
		sb.WriteString(strings.Join(p.Skills, ", "))
		sb.WriteString("\n\n")
	}
	if len(p.Certifications) > 0 {
		sb.WriteString("## Certifications\n")
		for _, c := range p.Certifications {
			fmt.Fprintf(&sb, "- %s — %s\n", c.Name, c.Authority)
		}
	}
	return sb.String()
}

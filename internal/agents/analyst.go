package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Prasad-178/reps/internal/llm"
)

type AnalystInput struct {
	Profile      string
	JDCards      []JDSummary
	CategoryELO  map[string]int
	WeakTopics   []WeakTopic
	RecentDrills []RecentDrillSummary
}

type RecentDrillSummary struct {
	StartedAt int64    `json:"started_at"`
	Mode      string   `json:"mode"`
	Topics    []string `json:"topics"`
	Ratings   []int    `json:"ratings"`
	Categories []string `json:"categories"`
}

// InsightPanel is one auto-built dashboard card. The frontend renders panels
// purely from this structure — no preset layouts.
type InsightPanel struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Kind       string   `json:"kind"`        // headline | callout | stat-row | sparkline | tag-cloud | list
	Severity   string   `json:"severity"`    // good | warn | bad | info
	Headline   string   `json:"headline"`
	Body       string   `json:"body"`
	Stats      []Stat   `json:"stats,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Items      []string `json:"items,omitempty"`
	Suggestion string   `json:"suggestion,omitempty"`
}

type Stat struct {
	Label string  `json:"label"`
	Value string  `json:"value"`
	Delta float64 `json:"delta,omitempty"`
	Unit  string  `json:"unit,omitempty"`
}

type AnalystOutput struct {
	Panels  []InsightPanel `json:"panels"`
	Summary string         `json:"summary"`
}

type Analyst struct{ Client *llm.Client }

func NewAnalyst(c *llm.Client) *Analyst { return &Analyst{Client: c} }

func (a *Analyst) Insights(ctx context.Context, in AnalystInput) (AnalystOutput, error) {
	jds, _ := json.Marshal(in.JDCards)
	elo, _ := json.Marshal(in.CategoryELO)
	weak, _ := json.Marshal(in.WeakTopics)
	recent, _ := json.Marshal(in.RecentDrills)

	var sb strings.Builder
	sb.WriteString("# Candidate profile\n")
	sb.WriteString(strings.TrimSpace(in.Profile))
	fmt.Fprintf(&sb, "\n\n# Target JDs\n%s\n", string(jds))
	fmt.Fprintf(&sb, "\n# Per-category ELO\n%s\n", string(elo))
	fmt.Fprintf(&sb, "\n# Weakest topics\n%s\n", string(weak))
	fmt.Fprintf(&sb, "\n# Recent drills\n%s\n", string(recent))

	req := llm.ChatRequest{
		Model: a.Client.Model,
		Messages: []llm.Message{
			{Role: "system", Content: analystSystemPrompt},
			{Role: "user", Content: sb.String()},
		},
		Temperature: 0.4,
		MaxTokens:   1500,
	}
	var out AnalystOutput
	if _, err := a.Client.ChatJSON(ctx, req, &out); err != nil {
		return AnalystOutput{}, err
	}
	if len(out.Panels) == 0 {
		return AnalystOutput{}, fmt.Errorf("analyst returned no panels")
	}
	return out, nil
}

const analystSystemPrompt = `You are the Analyst agent for "reps". You generate a *short, specific* set of dashboard
insights about the candidate's drill history. The UI renders your output verbatim as cards.

You receive: profile, target JDs, per-category ELO, weakest topics, recent drills.

Return JSON ONLY in this shape:

{
  "summary": "1-2 sentence overview, terse",
  "panels": [
    {
      "id": "kebab-id",
      "title": "Short panel title (≤ 6 words)",
      "kind": "headline" | "callout" | "stat-row" | "sparkline" | "tag-cloud" | "list",
      "severity": "good" | "warn" | "bad" | "info",
      "headline": "Big one-line takeaway",
      "body": "1-2 sentences of evidence; always cite numbers (ratings, deltas, counts)",
      "stats":      [{"label":"…","value":"…","delta":-12.0,"unit":"%"}],
      "tags":       ["multi-tenant-fhe","pq-key-rotation"],
      "items":      ["bullet 1","bullet 2"],
      "suggestion": "One concrete next action — phrased imperatively"
    }
  ]
}

Hard rules:
- Generate between 3 and 6 panels. Quality > quantity.
- Each panel must be grounded in the data. NEVER invent numbers. If data is thin, say so.
- Pick the panel "kind" that matches the insight:
    headline  = single big claim, no chart
    callout   = important warning or strength (severity != info)
    stat-row  = use when 2-4 numbers tell the story; populate "stats"
    tag-cloud = use when 4+ related topic_tags are the story; populate "tags"
    list      = use for an ordered or unordered list; populate "items"
    sparkline = trend over time; describe in body, no chart data needed
- "severity" maps colour: good=success, warn=warning, bad=destructive, info=primary.
- Be honest. If most ratings are 3/5 and ELO is flat, say "you're plateauing on X" — don't congratulate.
- Always finish each panel with a "suggestion" that's actionable in <30 minutes.

Output JSON only. No prose, no fences.`

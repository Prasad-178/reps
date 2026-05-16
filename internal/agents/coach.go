package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Prasad-178/reps/internal/llm"
)

type CoachInput struct {
	Profile     string
	WindowDays  int
	JDCards     []JDSummary
	Topics      []CoachTopic
}

type CoachTopic struct {
	Tag         string   `json:"tag"`
	Hits        int      `json:"hits"`
	MeanRating  float64  `json:"mean_rating"`
	LastSeenAgo string   `json:"last_seen_ago"`
	Categories  []string `json:"categories"`
}

type Coach struct{ Client *llm.Client }

func NewCoach(c *llm.Client) *Coach { return &Coach{Client: c} }

// Synthesize returns a Markdown study plan. It does NOT use json mode — the
// Coach output is Markdown, not JSON.
func (c *Coach) Synthesize(ctx context.Context, in CoachInput) (string, error) {
	user := buildCoachUserPrompt(in)
	req := llm.ChatRequest{
		Model: c.Client.Model,
		Messages: []llm.Message{
			{Role: "system", Content: coachSystemPrompt},
			{Role: "user", Content: user},
		},
		Temperature: 0.5,
		MaxTokens:   2400,
	}
	out, _, err := c.Client.Chat(ctx, req)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func buildCoachUserPrompt(in CoachInput) string {
	var sb strings.Builder
	sb.WriteString("# Candidate profile\n")
	sb.WriteString(strings.TrimSpace(in.Profile))
	jds, _ := json.Marshal(in.JDCards)
	fmt.Fprintf(&sb, "\n\n# Target JDs\n%s\n", string(jds))
	topics, _ := json.Marshal(in.Topics)
	fmt.Fprintf(&sb, "\n# Topic hits (last %d days)\n%s\n", in.WindowDays, string(topics))
	sb.WriteString("\nGenerate the study plan now.")
	return sb.String()
}

const coachSystemPrompt = `You are the Coach agent for "reps". You receive:
- The candidate's profile (their real shipped work).
- The target JDs.
- A list of topic hits over a recent window: each tag with hit count, mean rating, last-seen, and categories.

Your job: synthesize an ordered study plan that compounds. Output Markdown.

Rules:
- Cluster related tags into themes (e.g. {"multi-tenant-fhe","pq-key-rotation"} → "Encrypted multi-tenancy & key lifecycle"). 4-8 themes max.
- Order themes by priority. Priority = (hit count) * (4 - mean rating) * (JD relevance weight).
  JD relevance weight is higher if the theme matches a tech named in a target JD.
- Per theme write:
    ### <theme name>
    **Why this matters for you + targets:** 1-2 sentences. Tie to a specific JD or a specific past project.
    **Concrete weak signals:** the tag(s), hit count, mean rating.
    **Reading / drills:** 3-5 bullets. Mix of (a) named primary sources (papers, RFCs, docs) and (b) the next drill question to attempt.
- After the themes, append a "## Drill queue (this week)" section listing 5 specific topics to drill, each tied to a theme.
- Be honest, not generous. "Solid on X" goes in the intro only if mean_rating ≥ 4.

Open with a one-paragraph "## Read first" intro setting the priority order.
Close with "## Backlog" listing themes you trimmed and why.

No fluff, no platitudes. The candidate is senior and wants signal, not motivation.

The output is plain Markdown. No JSON, no fenced code blocks around the whole answer.`

// FormatLastSeen returns a compact relative-time string for use in the prompt.
func FormatLastSeen(ts time.Time) string {
	if ts.IsZero() {
		return "never"
	}
	d := time.Since(ts)
	switch {
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dmo", int(d.Hours()/24/30))
	}
}

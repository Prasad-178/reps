package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Prasad-178/reps/internal/llm"
)

// SessionAnalyst produces a deep, cross-question critique of a single drill
// session. Different from the per-question Judge (rates one answer) and the
// Coach (multi-session study plan). This is "what patterns showed up in this
// single 15-minute sitting, and how do I answer better next time?".

type SessionAnalystInput struct {
	Profile  string                       // candidate profile
	Mode     string                       // session mode (web/cli/voice)
	Started  int64                        // unix sec
	Items    []SessionAnalystQuestion     // one per question
}

type SessionAnalystQuestion struct {
	Ord         int                      `json:"ord"`
	Category    string                   `json:"category"`
	Topic       string                   `json:"topic"`
	TargetELO   int                      `json:"target_elo"`
	Turns       []SessionAnalystTurn     `json:"turns"`
	Rating      int                      `json:"rating,omitempty"`     // 0 if ungraded
	Strengths   []string                 `json:"strengths,omitempty"`
	Missed      []string                 `json:"missed,omitempty"`
	BetterSketch string                  `json:"better_sketch,omitempty"`
}

type SessionAnalystTurn struct {
	Speaker string `json:"speaker"` // interviewer | candidate
	Kind    string `json:"kind"`    // opening | followup | answer
	Text    string `json:"text"`
}

// SessionCritique is the structured output rendered by the frontend.
type SessionCritique struct {
	Headline    string             `json:"headline"`    // 1-line summary
	Verdict     string             `json:"verdict"`     // good | mixed | bad
	OverallRating float64          `json:"overall_rating"` // 0..5
	Patterns    []CritiquePattern  `json:"patterns"`     // recurring archetypes
	Strengths   []string           `json:"strengths"`    // 1-3 things you nailed
	GrowthEdge  []GrowthItem       `json:"growth_edge"`  // 2-4 concrete next steps
	DrillAgain  []string           `json:"drill_again"`  // topic phrases for next session
	Reading     []CritiqueReading  `json:"reading"`      // optional curated reads
}

type CritiquePattern struct {
	Name     string `json:"name"`      // archetype name, e.g. "hand-waving on capacity numbers"
	Evidence string `json:"evidence"`  // quote/paraphrase from this session
	Fix      string `json:"fix"`       // concrete how-to-answer-better
}

type GrowthItem struct {
	Action string `json:"action"`  // imperative, e.g. "rehearse PQ recall recovery aloud"
	Why    string `json:"why"`     // tie back to a moment in this session
}

type CritiqueReading struct {
	Topic string `json:"topic"`
	Why   string `json:"why"`
	URL   string `json:"url,omitempty"`
}

type SessionAnalyst struct{ Client *llm.Client }

func NewSessionAnalyst(c *llm.Client) *SessionAnalyst { return &SessionAnalyst{Client: c} }

func (a *SessionAnalyst) Critique(ctx context.Context, in SessionAnalystInput) (SessionCritique, error) {
	itemsJSON, _ := json.Marshal(in.Items)

	var sb strings.Builder
	sb.WriteString("# Candidate profile\n")
	sb.WriteString(strings.TrimSpace(in.Profile))
	sb.WriteString("\n\n# Session metadata\n")
	fmt.Fprintf(&sb, "mode: %s\nstarted: %d (unix)\nquestion_count: %d\n", in.Mode, in.Started, len(in.Items))
	sb.WriteString("\n# Full session (each question with its transcript and judgment)\n")
	sb.Write(itemsJSON)

	req := llm.ChatRequest{
		Model: a.Client.Model,
		Messages: []llm.Message{
			{Role: "system", Content: sessionAnalystSystemPrompt},
			{Role: "user", Content: sb.String()},
		},
		Temperature: 0.4,
		MaxTokens:   1800,
	}

	var out SessionCritique
	if _, err := a.Client.ChatJSON(ctx, req, &out); err != nil {
		return SessionCritique{}, err
	}
	if strings.TrimSpace(out.Headline) == "" {
		return SessionCritique{}, fmt.Errorf("session analyst returned empty critique")
	}
	if out.Verdict == "" {
		out.Verdict = "mixed"
	}
	return out, nil
}

const sessionAnalystSystemPrompt = `You are the Session-Analyst agent for "reps". You critique a single drill session
*across all questions* and produce a focused, blunt, growth-oriented review.

You are NOT re-grading individual answers — the Judge already did that. You synthesize:
- what patterns the candidate showed across questions (good and bad)
- where the same weakness shows up more than once
- one or two strong moments worth keeping
- a small, concrete list of "do this before next session"

Voice: senior engineer mentoring directly. No hype. No softballs. No filler.
- If the session was weak, say so. Cite specific moments.
- If a strength is real, name it precisely.
- Patterns must be archetypal (e.g. "stops at intuition, never quantifies tradeoffs"),
  NOT just restating per-question judge feedback.

Return JSON ONLY in this exact shape:

{
  "headline": "one-sentence summary of the session, ≤ 20 words",
  "verdict": "good" | "mixed" | "bad",
  "overall_rating": number between 0 and 5 (one decimal allowed),
  "patterns": [
    {
      "name": "archetype name, ≤ 8 words",
      "evidence": "1-2 sentences quoting / paraphrasing this session",
      "fix": "1-2 sentence concrete how-to-answer-better next time"
    }
  ],
  "strengths": ["short bullet 1", "short bullet 2"],
  "growth_edge": [
    {"action": "imperative, ≤ 12 words", "why": "tie to a moment from this session"}
  ],
  "drill_again": ["topic phrase 1", "topic phrase 2"],
  "reading": [
    {"topic": "topic name", "why": "1-line why", "url": "optional"}
  ]
}

Hard rules:
- 2 to 4 patterns. Not more. Quality > quantity.
- 1 to 3 strengths. If the session had no genuine strength, return an empty array — do NOT invent one.
- 2 to 4 growth_edge items, each actionable in ≤ 30 minutes.
- 1 to 5 drill_again topic phrases — written exactly as you'd type into a drill prompt.
- Reading is optional (0-3 items). Skip if nothing crisp comes to mind.
- NEVER repeat the per-question Judge feedback verbatim. Synthesize.
- NEVER invent numbers or claims not present in the session transcript.

Output JSON only. No prose outside the JSON. No code fences.`

package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Prasad-178/reps/internal/llm"
)

type PlannerInput struct {
	Profile        string
	JDCards        []JDSummary
	CategoryELO    map[string]int
	RecentTopics   []RecentTopic
	WeakestTopics  []WeakTopic
	SessionTopics  []string // topics/projects already drilled this session — DO NOT REPEAT
	OverrideCat    string   // user-forced category, e.g. via --category
	OverrideTopic  string   // user-forced topic, e.g. via --topic
	OverrideJDID   string   // user-forced JD focus
	DefaultDiff    int      // default rating to start from when no ELO known
}

type JDSummary struct {
	ID      string `json:"id"`
	Company string `json:"company"`
	Role    string `json:"role"`
	Summary string `json:"summary"`
}

type RecentTopic struct {
	Tag      string `json:"tag"`
	Category string `json:"category"`
	Rating   int    `json:"rating"`
}

type WeakTopic struct {
	Tag        string  `json:"tag"`
	Hits       int     `json:"hits"`
	MeanRating float64 `json:"mean_rating"`
}

type PlannerDecision struct {
	Category   string `json:"category"`
	Topic      string `json:"target_topic"`
	Difficulty int    `json:"target_difficulty"`
	Why        string `json:"why"`
	JDID       string `json:"jd_id,omitempty"`
}

type Planner struct{ Client *llm.Client }

func NewPlanner(c *llm.Client) *Planner { return &Planner{Client: c} }

func (p *Planner) Decide(ctx context.Context, in PlannerInput) (PlannerDecision, error) {
	user, err := buildPlannerUserPrompt(in)
	if err != nil {
		return PlannerDecision{}, err
	}
	req := llm.ChatRequest{
		Model: p.Client.Model,
		Messages: []llm.Message{
			{Role: "system", Content: plannerSystemPrompt},
			{Role: "user", Content: user},
		},
		Temperature: 0.6,
		MaxTokens:   400,
	}
	var d PlannerDecision
	if _, err := p.Client.ChatJSON(ctx, req, &d); err != nil {
		return PlannerDecision{}, err
	}
	if !IsValidCategory(d.Category) {
		return d, fmt.Errorf("planner returned invalid category: %s", d.Category)
	}
	if strings.TrimSpace(d.Topic) == "" {
		return d, fmt.Errorf("planner returned empty topic")
	}
	if d.Difficulty < 600 || d.Difficulty > 2800 {
		d.Difficulty = clampELO(in.DefaultDiff)
	}
	return d, nil
}

func clampELO(v int) int {
	if v < 600 {
		return 1200
	}
	if v > 2800 {
		return 2800
	}
	return v
}

func buildPlannerUserPrompt(in PlannerInput) (string, error) {
	jdJSON, _ := json.Marshal(in.JDCards)
	recentJSON, _ := json.Marshal(in.RecentTopics)
	weakJSON, _ := json.Marshal(in.WeakestTopics)
	eloJSON, _ := json.Marshal(in.CategoryELO)

	var sb strings.Builder
	sb.WriteString("Candidate profile:\n")
	sb.WriteString(strings.TrimSpace(in.Profile))
	sb.WriteString("\n\n")
	fmt.Fprintf(&sb, "Per-category ELO (1200 default): %s\n", string(eloJSON))
	fmt.Fprintf(&sb, "Recent 20 drill topics (newest first): %s\n", string(recentJSON))
	fmt.Fprintf(&sb, "Weakest 10 topics (mean rating < 3.5): %s\n", string(weakJSON))
	fmt.Fprintf(&sb, "Target JDs: %s\n", string(jdJSON))

	if len(in.SessionTopics) > 0 {
		stJSON, _ := json.Marshal(in.SessionTopics)
		fmt.Fprintf(&sb, "Topics ALREADY drilled in THIS session (avoid repeating these projects/topics): %s\n", string(stJSON))
	}

	if in.OverrideCat != "" {
		fmt.Fprintf(&sb, "\nUser override: category=%s\n", in.OverrideCat)
	}
	if in.OverrideTopic != "" {
		fmt.Fprintf(&sb, "User override: topic=%s\n", in.OverrideTopic)
	}
	if in.OverrideJDID != "" {
		fmt.Fprintf(&sb, "User override: jd_id=%s\n", in.OverrideJDID)
	}
	return sb.String(), nil
}

const plannerSystemPrompt = `You are the Planner agent for "reps", a personalized interview rehearsal CLI.
Your only job is to pick what question to drill next.

You will receive:
- The candidate's profile (their real projects, skills, work history).
- Per-category ELO ratings (1200 is neutral).
- Recent 20 drill topics with ratings.
- The 10 weakest topics by mean rating.
- A list of target JDs the candidate is preparing for.
- Optional user overrides (category, topic, jd_id).

You must return a single JSON object with this schema and nothing else:

{
  "category": "system-design" | "domain-crypto" | "domain-ml" | "domain-solana" | "jd-specific" | "general",
  "target_topic": string,            // a concrete topic phrase, e.g. "multi-tenant FHE rotation"
  "target_difficulty": integer,      // target ELO at which a 3/5 is the expected outcome, 800..2400
  "why": string,                     // one short sentence explaining the choice
  "jd_id": string                    // optional, set only when category == "jd-specific"
}

How to choose, in priority order:
1. If the user supplied an override (category/topic/jd_id), honor it. Still set target_difficulty intelligently from ELO.
2. Session diversity (HARD RULE): if "Topics ALREADY drilled in THIS session" is non-empty, you MUST pick a topic
   anchored in a DIFFERENT project / area of the candidate's profile. The candidate's resume has many surfaces
   (multiple projects, skills, roles, papers) — spread the questions across them. Never anchor two questions in
   the same project unless the user forced it via override.
3. Spaced repetition: prefer one of the weakest topics every ~2 drills.
4. JD-priority: if some JD has core requirements the candidate hasn't drilled, pick "jd-specific" and set jd_id.
5. Category rotation: don't drill the same category twice in a row unless the user forced it.
6. Exploration: ~15% of the time, pick something orthogonal (a new topic from the profile that has zero recent hits).

Difficulty rule:
- Start from the chosen category's ELO (default 1200 if missing).
- If mean rating of last 5 drills in that category was >= 4, set target_difficulty = ELO + random(50..150).
- If it was <= 2, set target_difficulty = ELO - random(50..150).
- Otherwise target_difficulty = ELO ± random(0..80).

Be specific about target_topic. Generic ("design a chat app") is bad. Reference the candidate's actual work
("multi-tenant FHE rotation on top of your Opaque vector-search engine") is good.

Output JSON only. No prose, no markdown fences.`

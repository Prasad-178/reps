package agents

import (
	"context"
	"fmt"
	"strings"

	"github.com/Prasad-178/reps/internal/llm"
	"github.com/Prasad-178/reps/internal/rubrics"
)

type JudgeInput struct {
	Profile    string
	Decision   PlannerDecision
	Context    []ContextChunk
	Transcript Transcript
	JDCard     string
}

type Reading struct {
	Topic string `json:"topic"`
	Why   string `json:"why"`
	URL   string `json:"optional_url,omitempty"`
}

type Judgment struct {
	Rating             int       `json:"rating"`
	Strengths          []string  `json:"strengths"`
	Missed             []string  `json:"missed"`
	BetterAnswerSketch string    `json:"better_answer_sketch"`
	Reading            []Reading `json:"reading"`
	TopicTags          []string  `json:"topic_tags"`
}

type Judge struct {
	Client *llm.Client
	Model  string // optional override; falls back to Client.JudgeModel then Client.Model
}

func NewJudge(c *llm.Client) *Judge {
	model := c.JudgeModel
	if model == "" {
		model = c.Model
	}
	return &Judge{Client: c, Model: model}
}

func (j *Judge) Grade(ctx context.Context, in JudgeInput) (Judgment, error) {
	rubric, err := rubrics.Load(in.Decision.Category)
	if err != nil {
		return Judgment{}, err
	}
	user := buildJudgeUserPrompt(in, rubric)
	req := llm.ChatRequest{
		Model: j.Model,
		Messages: []llm.Message{
			{Role: "system", Content: judgeSystemPrompt},
			{Role: "user", Content: user},
		},
		Temperature: 0.2,
		MaxTokens:   1200,
	}
	var out Judgment
	resp, err := j.Client.ChatJSON(ctx, req, &out)
	if err != nil {
		// one retry asking the model to fix its output
		retry := llm.ChatRequest{
			Model: j.Model,
			Messages: []llm.Message{
				{Role: "system", Content: judgeSystemPrompt},
				{Role: "user", Content: user},
				{Role: "assistant", Content: "(previous output failed schema validation)"},
				{Role: "user", Content: fmt.Sprintf("Your prior reply failed JSON parsing: %v. Re-emit the JSON only.", err)},
			},
			Temperature: 0.1,
			MaxTokens:   1200,
		}
		if _, err2 := j.Client.ChatJSON(ctx, retry, &out); err2 != nil {
			return Judgment{}, fmt.Errorf("judge JSON failed twice: %w", err2)
		}
	}
	_ = resp
	if out.Rating < 1 || out.Rating > 5 {
		return out, fmt.Errorf("judge returned rating %d, expected 1..5", out.Rating)
	}
	out.TopicTags = normalizeTags(out.TopicTags)
	return out, nil
}

func normalizeTags(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range in {
		t = strings.ToLower(strings.TrimSpace(t))
		t = strings.ReplaceAll(t, " ", "-")
		t = strings.ReplaceAll(t, "_", "-")
		t = strings.Trim(t, "-")
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

func buildJudgeUserPrompt(in JudgeInput, rubric string) string {
	var sb strings.Builder
	sb.WriteString("# Rubric (YAML)\n```yaml\n")
	sb.WriteString(rubric)
	sb.WriteString("\n```\n\n# Candidate profile\n")
	sb.WriteString(strings.TrimSpace(in.Profile))
	sb.WriteString("\n\n# Drill plan\n")
	fmt.Fprintf(&sb, "category: %s\ntarget_topic: %s\ntarget_difficulty: %d\n",
		in.Decision.Category, in.Decision.Topic, in.Decision.Difficulty)
	if in.JDCard != "" {
		sb.WriteString("\n# Target JD card\n")
		sb.WriteString(in.JDCard)
		sb.WriteString("\n")
	}
	sb.WriteString("\n# Grounding context the interviewer had access to\n")
	for i, ch := range in.Context {
		body := ch.Text
		if len(body) > 1200 {
			body = body[:1200] + "..."
		}
		fmt.Fprintf(&sb, "[%d] (%s/%s)\n%s\n\n", i, ch.Kind, ch.Ref, body)
	}
	sb.WriteString("\n# Transcript\n")
	fmt.Fprintf(&sb, "Interviewer (opening): %s\n", in.Transcript.OpeningQ)
	for i, ex := range in.Transcript.Exchanges {
		fmt.Fprintf(&sb, "Candidate (ans %d): %s\n", i+1, ex.Answer)
		if ex.FollowupQ != "" {
			fmt.Fprintf(&sb, "Interviewer (followup %d): %s\n", i+1, ex.FollowupQ)
		}
	}
	sb.WriteString("\nGrade now. Return ONLY the JSON object specified by the system prompt.")
	return sb.String()
}

const judgeSystemPrompt = `You are the Judge agent for "reps". Be honest, not generous.

You receive a rubric YAML (the rubric for this category), the candidate's profile, the
grounding context the interviewer had access to, and the full Q + follow-ups + answers
transcript.

Rules:
- Use the rubric. Score the depth, not the effort.
- Be specific. "Strong on X" without naming what is the same as not saying it.
- The rubric anchors define 1/3/5 — interpolate for 2/4.
- If the candidate hand-waved or was generic, the rating tops out at 2.
- If the candidate referenced their actual project specifics, that's a floor of 3, not a ceiling.
- Topic tags must be lowercase-kebab, 3..6 tags, narrow enough to spot weak themes
  later (e.g. "multi-tenant-fhe" not "crypto").

Output ONLY this JSON object:

{
  "rating": 1..5,
  "strengths": [string],
  "missed": [string],
  "better_answer_sketch": "2-4 sentence sketch of how a strong candidate would have framed it",
  "reading": [{"topic": string, "why": string, "optional_url": string}],
  "topic_tags": [string]
}

No prose outside the JSON. No markdown fences.`

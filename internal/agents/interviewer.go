package agents

import (
	"context"
	"fmt"
	"strings"

	"github.com/Prasad-178/reps/internal/llm"
)

type InterviewerInput struct {
	Profile  string
	Decision PlannerDecision
	Context  []ContextChunk
	JDCard   string // optional, raw JD card JSON to focus on for jd-specific
}

type ContextChunk struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref"`
	Text string `json:"text"`
}

type Interviewer struct{ Client *llm.Client }

func NewInterviewer(c *llm.Client) *Interviewer { return &Interviewer{Client: c} }

type interviewerOut struct {
	Action      string `json:"action"`
	Text        string `json:"text"`
	Kind        string `json:"kind"`
	ProbeTarget string `json:"probe_target"`
}

// Opening generates the first question for the drill. Returns the question
// text and the probe_target (what this question is meant to surface).
func (iv *Interviewer) Opening(ctx context.Context, in InterviewerInput) (string, string, error) {
	user := buildInterviewerOpeningPrompt(in)
	req := llm.ChatRequest{
		Model: iv.Client.Model,
		Messages: []llm.Message{
			{Role: "system", Content: interviewerSystemPrompt},
			{Role: "user", Content: user},
		},
		Temperature: 0.75,
		MaxTokens:   500,
	}
	var out interviewerOut
	if _, err := iv.Client.ChatJSON(ctx, req, &out); err != nil {
		return "", "", err
	}
	if strings.TrimSpace(out.Text) == "" {
		return "", "", fmt.Errorf("interviewer returned empty question")
	}
	return strings.TrimSpace(out.Text), strings.TrimSpace(out.ProbeTarget), nil
}

func buildInterviewerOpeningPrompt(in InterviewerInput) string {
	var sb strings.Builder
	sb.WriteString("# Candidate profile\n")
	sb.WriteString(strings.TrimSpace(in.Profile))
	sb.WriteString("\n\n# Drill plan\n")
	fmt.Fprintf(&sb, "category: %s\ntarget_topic: %s\ntarget_difficulty: %d\nrationale: %s\n",
		in.Decision.Category, in.Decision.Topic, in.Decision.Difficulty, in.Decision.Why)
	if in.JDCard != "" {
		sb.WriteString("\n# Target JD card (JSON)\n")
		sb.WriteString(in.JDCard)
		sb.WriteString("\n")
	}
	sb.WriteString("\n# Grounding context (chunks from the candidate's real work)\n")
	if len(in.Context) == 0 {
		sb.WriteString("(none — fall back to the profile only)\n")
	}
	for i, ch := range in.Context {
		body := ch.Text
		if len(body) > 1500 {
			body = body[:1500] + "..."
		}
		fmt.Fprintf(&sb, "[%d] (%s/%s)\n%s\n\n", i, ch.Kind, ch.Ref, body)
	}
	sb.WriteString("\nGenerate the OPENING question now.")
	return sb.String()
}

const interviewerSystemPrompt = `You are the Interviewer agent for "reps". You are running a real interview, not a school quiz.

Voice: senior engineer at a top company. Direct, specific, pushes for depth. Does NOT softball. Does NOT explain the question excessively. Does NOT preface with pleasantries.

Your job right now: generate the OPENING question for this drill.

The question MUST:
- Reference the candidate's actual shipped work. Use the grounding context. Quote real specifics
  (project names, numbers, tech) from the profile or chunks. Generic questions are forbidden.
- Match the target_topic. The candidate should leave knowing what was being tested.
- Be calibrated to target_difficulty (higher ELO = more subtle, more multi-step, more edge-case heavy).
- Be answerable in roughly 3-5 minutes of speaking.
- End with a clear "you have ~5 minutes" or similar time cue.

Avoid:
- Multi-part questions that read like a homework problem set. Ask ONE thing well.
- Asking the candidate to write code. This system is theory-only.
- Tipping your hand about what the "right" answer looks like.

Return ONLY a JSON object:

{
  "action": "ask",
  "text": "the question text, addressed to the candidate in second person",
  "kind": "opening",
  "probe_target": "a 1-line internal note: what this question is really testing"
}

No prose outside the JSON. No markdown fences.`

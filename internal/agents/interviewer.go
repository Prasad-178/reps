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

type Transcript struct {
	OpeningQ    string
	Exchanges   []Exchange
}

type Exchange struct {
	Answer    string
	FollowupQ string // empty for the final exchange (no follow-up after it)
}

type StepDecision struct {
	Action      string // "ask" | "done"
	Text        string // follow-up text when Action == "ask"
	ProbeTarget string
}

// OpeningStream streams the opening question text token-by-token via onToken.
// Returns the full text and a synthesized probe target on completion. This
// variant uses plain-text completion (no JSON wrapping) so the stream is
// usable directly in a typewriter UI.
func (iv *Interviewer) OpeningStream(ctx context.Context, in InterviewerInput, onToken func(string)) (string, string, error) {
	user := buildInterviewerOpeningPrompt(in) +
		"\n\nReturn ONLY the question text addressed to the candidate. " +
		"No JSON, no preface, no labels — the question itself. " +
		"End with a clear time cue (e.g. 'you have about 5 minutes')."
	req := llm.ChatRequest{
		Model: iv.Client.Model,
		Messages: []llm.Message{
			{Role: "system", Content: interviewerSystemPrompt + "\n\nNOTE: Plain-text-only output mode."},
			{Role: "user", Content: user},
		},
		Temperature: 0.75,
		MaxTokens:   500,
	}
	full, _, err := iv.Client.ChatStream(ctx, req, onToken)
	if err != nil {
		return "", "", err
	}
	text := strings.TrimSpace(full)
	if text == "" {
		return "", "", fmt.Errorf("interviewer returned empty question")
	}
	return text, in.Decision.Topic, nil
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

// Step decides whether to ask a follow-up given the running transcript. It
// returns Action="done" when the candidate's last answer is complete and deep
// enough to grade, or when followupsRemaining <= 0.
func (iv *Interviewer) Step(
	ctx context.Context,
	in InterviewerInput,
	transcript Transcript,
	followupsRemaining int,
) (StepDecision, error) {
	if followupsRemaining <= 0 {
		return StepDecision{Action: "done"}, nil
	}
	user := buildInterviewerStepPrompt(in, transcript, followupsRemaining)
	req := llm.ChatRequest{
		Model: iv.Client.Model,
		Messages: []llm.Message{
			{Role: "system", Content: interviewerStepSystemPrompt},
			{Role: "user", Content: user},
		},
		Temperature: 0.7,
		MaxTokens:   400,
	}
	var out interviewerOut
	if _, err := iv.Client.ChatJSON(ctx, req, &out); err != nil {
		return StepDecision{}, err
	}
	action := strings.ToLower(strings.TrimSpace(out.Action))
	if action != "ask" && action != "done" {
		if strings.Contains(strings.ToLower(out.Text), "<<done>>") || strings.TrimSpace(out.Text) == "" {
			action = "done"
		} else {
			action = "ask"
		}
	}
	if action == "done" {
		return StepDecision{Action: "done"}, nil
	}
	text := strings.TrimSpace(out.Text)
	text = strings.TrimSuffix(text, "<<DONE>>")
	text = strings.TrimSpace(text)
	if text == "" {
		return StepDecision{Action: "done"}, nil
	}
	return StepDecision{
		Action:      "ask",
		Text:        text,
		ProbeTarget: strings.TrimSpace(out.ProbeTarget),
	}, nil
}

func buildInterviewerStepPrompt(in InterviewerInput, t Transcript, remaining int) string {
	var sb strings.Builder
	sb.WriteString("# Candidate profile\n")
	sb.WriteString(strings.TrimSpace(in.Profile))
	sb.WriteString("\n\n# Drill plan\n")
	fmt.Fprintf(&sb, "category: %s\ntarget_topic: %s\ntarget_difficulty: %d\n",
		in.Decision.Category, in.Decision.Topic, in.Decision.Difficulty)
	if in.JDCard != "" {
		sb.WriteString("\n# Target JD card (JSON)\n")
		sb.WriteString(in.JDCard)
		sb.WriteString("\n")
	}
	sb.WriteString("\n# Grounding context\n")
	if len(in.Context) == 0 {
		sb.WriteString("(none — fall back to the profile only)\n")
	}
	for i, ch := range in.Context {
		body := ch.Text
		if len(body) > 1200 {
			body = body[:1200] + "..."
		}
		fmt.Fprintf(&sb, "[%d] (%s/%s)\n%s\n\n", i, ch.Kind, ch.Ref, body)
	}
	sb.WriteString("\n# Transcript so far\n")
	fmt.Fprintf(&sb, "Interviewer (opening): %s\n", t.OpeningQ)
	for i, ex := range t.Exchanges {
		fmt.Fprintf(&sb, "Candidate (ans %d): %s\n", i+1, ex.Answer)
		if ex.FollowupQ != "" {
			fmt.Fprintf(&sb, "Interviewer (followup %d): %s\n", i+1, ex.FollowupQ)
		}
	}
	fmt.Fprintf(&sb, "\nFollow-ups remaining: %d (hard cap).\n", remaining)
	sb.WriteString("\nDecide: ask one more follow-up, or end the question.\n")
	return sb.String()
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

const interviewerStepSystemPrompt = `You are the Interviewer agent for "reps". You are mid-interview, deciding whether
to ask a follow-up to probe deeper or to end the question and let the Judge grade.

Follow-up rule (CRITICAL):
- Ask a follow-up ONLY IF the most recent answer leaves a gap, hand-waves, skips a tradeoff,
  invents specifics not grounded in the candidate's stated work, or its depth is unclear.
- If the answer is already complete, well-grounded, and demonstrates real understanding,
  return action="done" IMMEDIATELY. Do not pad with follow-ups for the sake of it.
- Follow-ups exist to test whether the candidate actually understands the topic, not to fill turns.

Style:
- Real interviewer voice. Terse. No softballs.
- Probe one specific weakness per follow-up. Don't ask two things at once.
- Reference the candidate's actual claim from the prior answer ("you said X — but how does that handle Y?").

Output ONLY this JSON object:

{
  "action": "ask" | "done",
  "text": "follow-up text (only when action=ask); empty when action=done",
  "kind": "followup",
  "probe_target": "1-line note: which gap this follow-up tests"
}

If action="done", "text" must be empty or contain just "<<DONE>>".
No prose outside the JSON. No markdown fences.`

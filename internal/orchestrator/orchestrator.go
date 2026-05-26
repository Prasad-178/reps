package orchestrator

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Prasad-178/reps/internal/agents"
	"github.com/Prasad-178/reps/internal/config"
	"github.com/Prasad-178/reps/internal/elo"
	"github.com/Prasad-178/reps/internal/llm"
	"github.com/Prasad-178/reps/internal/rag"
	"github.com/Prasad-178/reps/internal/store"
	"github.com/Prasad-178/reps/internal/voice"
	"github.com/google/uuid"
)

type Options struct {
	Qs             int
	Voice          bool
	CategoryFilter string
	TopicOverride  string
	JDOverride     string
	DifficultyOver int
}

type Orchestrator struct {
	Cfg      config.Config
	Store    *store.Store
	Client   *llm.Client
	Retriever *rag.Retriever
	Planner  *agents.Planner
	Iv       *agents.Interviewer
	Judge    *agents.Judge
	Voice    *voice.Recorder
	Speaker  *voice.Speaker

	In  io.Reader
	Out io.Writer
}

func New(cfg config.Config, s *store.Store, c *llm.Client) *Orchestrator {
	return &Orchestrator{
		Cfg:       cfg,
		Store:     s,
		Client:    c,
		Retriever: rag.New(s, c),
		Planner:   agents.NewPlanner(c),
		Iv:        agents.NewInterviewer(c),
		Judge:     agents.NewJudge(c),
		Voice:     voice.New(cfg),
		Speaker:   voice.NewSpeaker(cfg),
		In:        os.Stdin,
		Out:       os.Stdout,
	}
}

// Run executes a single drill session. M3: opening Q + one answer, no follow-ups, no judge.
func (o *Orchestrator) Run(ctx context.Context, opt Options) error {
	if err := o.preflight(opt); err != nil {
		return err
	}
	profile, _, _, err := o.Store.GetProfile()
	if err != nil {
		return err
	}
	jds, err := o.Store.ListJDCards()
	if err != nil {
		return err
	}
	if opt.CategoryFilter == "jd-specific" && len(jds) == 0 {
		return fmt.Errorf("no JDs ingested. Run `reps add jd <url>` first.")
	}

	mode := "text"
	if opt.Voice {
		mode = "voice"
	}
	cfgJSON, _ := json.Marshal(opt)
	sess := store.Session{
		ID:         uuid.NewString(),
		StartedAt:  time.Now(),
		Mode:       mode,
		ConfigJSON: string(cfgJSON),
	}
	if err := o.Store.InsertSession(sess); err != nil {
		return err
	}
	defer func() { _ = o.Store.CloseSession(sess.ID, time.Now()) }()

	fmt.Fprintf(o.Out, "session %s started (%s mode, %d Qs)\n\n", sess.ID[:8], mode, opt.Qs)

	sessionTopics := make([]string, 0, opt.Qs)
	for i := 1; i <= opt.Qs; i++ {
		if err := o.runOneQuestion(ctx, sess.ID, i, opt, profile, jds, &sessionTopics); err != nil {
			return fmt.Errorf("Q%d: %w", i, err)
		}
	}
	fmt.Fprintf(o.Out, "\nsession %s complete. (judge + ELO come in M5/M7)\n", sess.ID[:8])
	return nil
}

func (o *Orchestrator) preflight(opt Options) error {
	ok, err := o.Store.HasProfile()
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("no profile yet. Run `reps init` then `reps add ...` and `reps profile --rebuild` first.")
	}
	if opt.Qs < 1 || opt.Qs > 10 {
		return fmt.Errorf("--qs must be in 1..10 (got %d)", opt.Qs)
	}
	if opt.Voice {
		if err := o.Voice.Available(); err != nil {
			fmt.Fprintf(o.Out, "[voice unavailable: %v] — falling back to text input.\n", err)
		}
	}
	return nil
}

func (o *Orchestrator) readUserAnswer(ctx context.Context, useVoice bool) (string, error) {
	if !useVoice {
		return readAnswer(o.In)
	}
	if err := o.Voice.Available(); err != nil {
		fmt.Fprintf(o.Out, "[voice unavailable: %v] — type instead.\n", err)
		return readAnswer(o.In)
	}
	txt, err := o.Voice.RecordAndTranscribe(ctx, o.In, o.Out)
	if err != nil {
		fmt.Fprintf(o.Out, "[voice error: %v] — type instead.\n", err)
		return readAnswer(o.In)
	}
	return txt, nil
}

func (o *Orchestrator) runOneQuestion(
	ctx context.Context, sessID string, ord int, opt Options, profile string, jds []store.JDCard, sessionTopics *[]string,
) error {
	fmt.Fprintf(o.Out, "─── Q%d/%d ──────────────────────────────────────────────\n", ord, opt.Qs)
	fmt.Fprint(o.Out, "Planner deciding...\n")

	elo, err := o.Store.GetAllELO()
	if err != nil {
		return err
	}
	if elo == nil {
		elo = map[string]int{}
	}
	for _, cat := range agents.Categories {
		if _, ok := elo[cat]; !ok {
			elo[cat] = o.Cfg.Elo.StartRating
		}
	}
	recent, err := o.Store.RecentTopics(20)
	if err != nil {
		return err
	}
	weakest, err := o.Store.WeakestTopics(10)
	if err != nil {
		return err
	}

	plannerJDs := make([]agents.JDSummary, 0, len(jds))
	for _, j := range jds {
		summary := j.Role
		if len(j.CardJSON) > 0 {
			var card struct{ Summary string `json:"summary"` }
			_ = json.Unmarshal([]byte(j.CardJSON), &card)
			if card.Summary != "" {
				summary = card.Summary
			}
		}
		plannerJDs = append(plannerJDs, agents.JDSummary{
			ID: j.ID, Company: j.Company, Role: j.Role, Summary: summary,
		})
	}
	recentMapped := make([]agents.RecentTopic, 0, len(recent))
	for _, r := range recent {
		recentMapped = append(recentMapped, agents.RecentTopic{
			Tag: r.Tag, Category: r.Category, Rating: r.Rating,
		})
	}
	weakMapped := make([]agents.WeakTopic, 0, len(weakest))
	for _, w := range weakest {
		weakMapped = append(weakMapped, agents.WeakTopic{
			Tag: w.Tag, Hits: w.Hits, MeanRating: w.MeanRating,
		})
	}

	var st []string
	if sessionTopics != nil {
		st = *sessionTopics
	}
	decision, err := o.Planner.Decide(ctx, agents.PlannerInput{
		Profile:       profile,
		JDCards:       plannerJDs,
		CategoryELO:   elo,
		RecentTopics:  recentMapped,
		WeakestTopics: weakMapped,
		SessionTopics: st,
		OverrideCat:   opt.CategoryFilter,
		OverrideTopic: opt.TopicOverride,
		OverrideJDID:  opt.JDOverride,
		DefaultDiff:   o.Cfg.Elo.StartRating,
	})
	if err != nil {
		return fmt.Errorf("planner: %w", err)
	}
	if opt.DifficultyOver != 0 {
		decision.Difficulty = opt.DifficultyOver
	}
	if sessionTopics != nil {
		*sessionTopics = append(*sessionTopics, decision.Topic)
	}
	fmt.Fprintf(o.Out, "Plan: %s | topic=%q | diff=%d\n  why: %s\n",
		decision.Category, decision.Topic, decision.Difficulty, decision.Why)

	fmt.Fprint(o.Out, "Retrieving context...\n")
	query := decision.Topic + " — " + decision.Why
	hits, err := o.Retriever.Retrieve(ctx, query, 8)
	if err != nil {
		return fmt.Errorf("retrieve: %w", err)
	}
	hits = o.Retriever.Rerank(ctx, query, hits, 3)

	ivContext := make([]agents.ContextChunk, 0, len(hits))
	for _, h := range hits {
		ivContext = append(ivContext, agents.ContextChunk{Kind: h.Kind, Ref: h.Ref, Text: h.Text})
	}

	var jdCardJSON string
	if decision.Category == "jd-specific" && decision.JDID != "" {
		for _, j := range jds {
			if j.ID == decision.JDID {
				jdCardJSON = j.CardJSON
				break
			}
		}
	}

	fmt.Fprint(o.Out, "Interviewer drafting question...\n\n")
	qText, probe, err := o.Iv.Opening(ctx, agents.InterviewerInput{
		Profile:  profile,
		Decision: decision,
		Context:  ivContext,
		JDCard:   jdCardJSON,
	})
	if err != nil {
		return fmt.Errorf("interviewer: %w", err)
	}

	contextJSON, _ := json.Marshal(hitsToRefs(hits))
	qID := uuid.NewString()
	if err := o.Store.InsertQuestion(store.Question{
		ID:                qID,
		SessionID:         sessID,
		Ord:               ord,
		Category:          decision.Category,
		TargetTopic:       decision.Topic,
		TargetELO:         decision.Difficulty,
		Rationale:         decision.Why + " | probe: " + probe,
		ContextChunksJSON: string(contextJSON),
		AskedAt:           time.Now(),
	}); err != nil {
		return err
	}
	if err := o.Store.InsertTurn(store.Turn{
		ID: uuid.NewString(), QuestionID: qID, Ord: 0,
		Speaker: "interviewer", Kind: "opening", Text: qText, Ts: time.Now(),
	}); err != nil {
		return err
	}

	fmt.Fprintln(o.Out, qText)
	o.Speaker.Speak(qText)
	fmt.Fprintln(o.Out)
	if opt.Voice {
		fmt.Fprintln(o.Out, "[voice mode] press Enter to start, Enter again to stop.")
	} else {
		fmt.Fprintln(o.Out, "(type your answer; finish with /end on a new line, or Ctrl-D)")
	}
	answer, err := o.readUserAnswer(ctx, opt.Voice)
	if err != nil {
		return err
	}
	turnOrd := 1
	if err := o.Store.InsertTurn(store.Turn{
		ID: uuid.NewString(), QuestionID: qID, Ord: turnOrd,
		Speaker: "candidate", Kind: "answer", Text: answer, Ts: time.Now(),
	}); err != nil {
		return err
	}

	transcript := agents.Transcript{
		OpeningQ:  qText,
		Exchanges: []agents.Exchange{{Answer: answer}},
	}
	maxFollowups := o.Cfg.Drill.FollowupMax
	if maxFollowups <= 0 {
		maxFollowups = 3
	}
	for fu := 0; fu < maxFollowups; fu++ {
		remaining := maxFollowups - fu
		step, err := o.Iv.Step(ctx, agents.InterviewerInput{
			Profile:  profile,
			Decision: decision,
			Context:  ivContext,
			JDCard:   jdCardJSON,
		}, transcript, remaining)
		if err != nil {
			return fmt.Errorf("interviewer step: %w", err)
		}
		if step.Action == "done" {
			break
		}
		fmt.Fprintf(o.Out, "\nFollow-up %d/%d: %s\n", fu+1, maxFollowups, step.Text)
		o.Speaker.Speak(step.Text)
		turnOrd++
		if err := o.Store.InsertTurn(store.Turn{
			ID: uuid.NewString(), QuestionID: qID, Ord: turnOrd,
			Speaker: "interviewer", Kind: "followup", Text: step.Text, Ts: time.Now(),
		}); err != nil {
			return err
		}
		// patch the running transcript with the follow-up text for the last exchange
		transcript.Exchanges[len(transcript.Exchanges)-1].FollowupQ = step.Text

		if opt.Voice {
			fmt.Fprintln(o.Out, "[voice mode] press Enter to start, Enter again to stop.")
		} else {
			fmt.Fprintln(o.Out, "(answer; /end on a new line to finish)")
		}
		ans, err := o.readUserAnswer(ctx, opt.Voice)
		if err != nil {
			return err
		}
		turnOrd++
		if err := o.Store.InsertTurn(store.Turn{
			ID: uuid.NewString(), QuestionID: qID, Ord: turnOrd,
			Speaker: "candidate", Kind: "answer", Text: ans, Ts: time.Now(),
		}); err != nil {
			return err
		}
		transcript.Exchanges = append(transcript.Exchanges, agents.Exchange{Answer: ans})
	}

	fmt.Fprintf(o.Out, "\nJudging... (%d follow-up(s))\n", len(transcript.Exchanges)-1)
	verdict, err := o.Judge.Grade(ctx, agents.JudgeInput{
		Profile:    profile,
		Decision:   decision,
		Context:    ivContext,
		Transcript: transcript,
		JDCard:     jdCardJSON,
	})
	if err != nil {
		fmt.Fprintf(o.Out, "  judge failed: %v (Q saved without grade)\n\n", err)
		return nil
	}
	if err := o.persistJudgment(qID, decision.Category, verdict); err != nil {
		fmt.Fprintf(o.Out, "  persist judgment failed: %v\n", err)
	}
	renderJudgment(o.Out, verdict)
	if err := o.applyELO(decision, verdict.Rating, qID); err != nil {
		fmt.Fprintf(o.Out, "  ELO update failed: %v\n", err)
	}
	return nil
}

func (o *Orchestrator) applyELO(d agents.PlannerDecision, rating int, qID string) error {
	before, err := o.Store.GetELO(d.Category, o.Cfg.Elo.StartRating)
	if err != nil {
		return err
	}
	score := elo.RatingToScore(rating)
	after, delta := elo.Update(before, d.Difficulty, score, o.Cfg.Elo.KFactor)
	if err := o.Store.UpsertELO(d.Category, after); err != nil {
		return err
	}
	if err := o.Store.InsertELOHistory(d.Category, before, after, delta, qID); err != nil {
		return err
	}
	sign := "+"
	if delta < 0 {
		sign = ""
	}
	fmt.Fprintf(o.Out, "ELO: %s %d → %d (%s%d)\n\n", d.Category, before, after, sign, delta)
	return nil
}

func (o *Orchestrator) persistJudgment(qID, category string, v agents.Judgment) error {
	strengthsJSON, _ := json.Marshal(v.Strengths)
	missedJSON, _ := json.Marshal(v.Missed)
	readingJSON, _ := json.Marshal(v.Reading)
	if err := o.Store.InsertJudgment(store.Judgment{
		QuestionID:    qID,
		Rating:        v.Rating,
		StrengthsJSON: string(strengthsJSON),
		MissedJSON:    string(missedJSON),
		BetterSketch:  v.BetterAnswerSketch,
		ReadingJSON:   string(readingJSON),
		GradedAt:      time.Now(),
		ModelUsed:     o.Client.JudgeModel,
	}); err != nil {
		return err
	}
	for _, tag := range v.TopicTags {
		if err := o.Store.InsertTopicHit(qID, tag, category, v.Rating); err != nil {
			return err
		}
	}
	return nil
}

func renderJudgment(w io.Writer, v agents.Judgment) {
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Rating: %d/5\n", v.Rating)
	if len(v.Strengths) > 0 {
		fmt.Fprintln(w, "Strengths:")
		for _, s := range v.Strengths {
			fmt.Fprintf(w, "  • %s\n", s)
		}
	}
	if len(v.Missed) > 0 {
		fmt.Fprintln(w, "Missed:")
		for _, s := range v.Missed {
			fmt.Fprintf(w, "  • %s\n", s)
		}
	}
	if v.BetterAnswerSketch != "" {
		fmt.Fprintln(w, "Better answer sketch:")
		fmt.Fprintf(w, "  %s\n", v.BetterAnswerSketch)
	}
	if len(v.Reading) > 0 {
		fmt.Fprintln(w, "Reading:")
		for _, r := range v.Reading {
			line := "  • " + r.Topic
			if r.Why != "" {
				line += " — " + r.Why
			}
			if r.URL != "" {
				line += " [" + r.URL + "]"
			}
			fmt.Fprintln(w, line)
		}
	}
	if len(v.TopicTags) > 0 {
		fmt.Fprintf(w, "Topic tags: %s\n", strings.Join(v.TopicTags, ", "))
	}
	fmt.Fprintln(w)
}

func hitsToRefs(hits []rag.Chunk) []map[string]any {
	out := make([]map[string]any, 0, len(hits))
	for _, h := range hits {
		out = append(out, map[string]any{
			"chunk_id": h.ID,
			"kind":     h.Kind,
			"ref":      h.Ref,
			"distance": h.Distance,
		})
	}
	return out
}

// readAnswer reads lines until EOF, a blank line followed by "/end",
// or the user just types "/end" on its own line. Empty answers are allowed.
func readAnswer(r io.Reader) (string, error) {
	rd := bufio.NewReader(r)
	var sb strings.Builder
	for {
		line, err := rd.ReadString('\n')
		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "/end" {
			break
		}
		sb.WriteString(line)
		if err != nil {
			if err == io.EOF {
				break
			}
			return sb.String(), err
		}
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

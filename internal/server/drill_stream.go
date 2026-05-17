package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Prasad-178/reps/internal/agents"
	"github.com/Prasad-178/reps/internal/elo"
	"github.com/Prasad-178/reps/internal/rag"
	"github.com/Prasad-178/reps/internal/store"
	"github.com/google/uuid"
)

// drillSession runs one web drill: Planner → Retrieve → Interviewer → (loop
// follow-ups, gated on HTTP POSTs for answers) → Judge → ELO.
// Communicates progress to the client via an SSE flush closure.

type drillOpts struct {
	Qs         int
	Category   string
	Topic      string
	JD         string
	Difficulty int
}

type drillSession struct {
	s         *Server
	opts      drillOpts
	SessionID string

	// per-question channel of incoming answers (waited on by Run)
	mu     sync.Mutex
	curQID string
	answer chan string
	end    chan struct{}
}

var (
	drillReg   = map[string]*drillSession{}
	drillRegMu sync.RWMutex
)

func registerDrillSession(d *drillSession) {
	drillRegMu.Lock()
	drillReg[d.SessionID] = d
	drillRegMu.Unlock()
}

func unregisterDrillSession(id string) {
	drillRegMu.Lock()
	delete(drillReg, id)
	drillRegMu.Unlock()
}

func getDrillSession(id string) (*drillSession, bool) {
	drillRegMu.RLock()
	d, ok := drillReg[id]
	drillRegMu.RUnlock()
	return d, ok
}

func newDrillSession(s *Server, opts drillOpts) *drillSession {
	return &drillSession{
		s:         s,
		opts:      opts,
		SessionID: uuid.NewString(),
	}
}

func submitAnswer(sess, qID, text string) error {
	d, ok := getDrillSession(sess)
	if !ok {
		return fmt.Errorf("session %s not active", sess)
	}
	d.mu.Lock()
	cur := d.curQID
	ch := d.answer
	d.mu.Unlock()
	if cur != qID {
		return fmt.Errorf("question %s is not the current one (current=%s)", qID, cur)
	}
	select {
	case ch <- text:
		return nil
	case <-time.After(3 * time.Second):
		return fmt.Errorf("no consumer for answer (timeout)")
	}
}

func endQuestion(sess, qID string) error {
	d, ok := getDrillSession(sess)
	if !ok {
		return fmt.Errorf("session %s not active", sess)
	}
	d.mu.Lock()
	cur := d.curQID
	end := d.end
	d.mu.Unlock()
	if cur != qID {
		return fmt.Errorf("question %s is not current", qID)
	}
	close(end)
	return nil
}

type flushFn func(event, data string)

func emit(flush flushFn, event string, v any) {
	b, _ := json.Marshal(v)
	flush(event, string(b))
}

func (d *drillSession) Run(ctx context.Context, flush flushFn) error {
	s := d.s

	// preflight: must have profile
	ok, err := s.Store.HasProfile()
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("no profile yet — run `reps init` then add sources and rebuild")
	}
	profile, _, _, _ := s.Store.GetProfile()
	jds, err := s.Store.ListJDCards()
	if err != nil {
		return err
	}
	if d.opts.Category == "jd-specific" && len(jds) == 0 {
		return fmt.Errorf("no JDs ingested for jd-specific drills")
	}

	// persist session
	cfgJSON, _ := json.Marshal(d.opts)
	sess := store.Session{
		ID: d.SessionID, StartedAt: time.Now(),
		Mode: "web", ConfigJSON: string(cfgJSON),
	}
	if err := s.Store.InsertSession(sess); err != nil {
		return err
	}
	defer func() { _ = s.Store.CloseSession(d.SessionID, time.Now()) }()

	retriever := rag.New(s.Store, s.Client)
	planner := agents.NewPlanner(s.Client)
	iv := agents.NewInterviewer(s.Client)
	judge := agents.NewJudge(s.Client)

	for i := 1; i <= d.opts.Qs; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		emit(flush, "question:start", map[string]any{"ord": i, "total": d.opts.Qs})

		eloMap, err := s.Store.GetAllELO()
		if err != nil {
			return err
		}
		if eloMap == nil {
			eloMap = map[string]int{}
		}
		for _, c := range agents.Categories {
			if _, ok := eloMap[c]; !ok {
				eloMap[c] = s.Cfg.Elo.StartRating
			}
		}
		recent, _ := s.Store.RecentTopics(20)
		weakest, _ := s.Store.WeakestTopics(10)

		plannerJDs := make([]agents.JDSummary, 0, len(jds))
		for _, j := range jds {
			plannerJDs = append(plannerJDs, agents.JDSummary{
				ID: j.ID, Company: j.Company, Role: j.Role,
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

		emit(flush, "planner:thinking", map[string]any{})
		decision, err := planner.Decide(ctx, agents.PlannerInput{
			Profile: profile, JDCards: plannerJDs, CategoryELO: eloMap,
			RecentTopics: recentMapped, WeakestTopics: weakMapped,
			OverrideCat: d.opts.Category, OverrideTopic: d.opts.Topic,
			OverrideJDID: d.opts.JD, DefaultDiff: s.Cfg.Elo.StartRating,
		})
		if err != nil {
			return fmt.Errorf("planner: %w", err)
		}
		if d.opts.Difficulty > 0 {
			decision.Difficulty = d.opts.Difficulty
		}
		emit(flush, "planner:decision", decision)

		emit(flush, "rag:retrieve", map[string]any{})
		query := decision.Topic + " — " + decision.Why
		hits, err := retriever.Retrieve(ctx, query, 8)
		if err != nil {
			return err
		}
		hits = retriever.Rerank(ctx, query, hits, 3)
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

		emit(flush, "interviewer:thinking", map[string]any{})
		qText, probe, err := iv.Opening(ctx, agents.InterviewerInput{
			Profile: profile, Decision: decision, Context: ivContext, JDCard: jdCardJSON,
		})
		if err != nil {
			return fmt.Errorf("interviewer: %w", err)
		}

		contextRefs := make([]map[string]any, 0, len(hits))
		for _, h := range hits {
			contextRefs = append(contextRefs, map[string]any{
				"kind": h.Kind, "ref": h.Ref, "chunk_id": h.ID, "distance": h.Distance,
			})
		}
		contextJSON, _ := json.Marshal(contextRefs)
		qID := uuid.NewString()
		if err := s.Store.InsertQuestion(store.Question{
			ID: qID, SessionID: d.SessionID, Ord: i,
			Category: decision.Category, TargetTopic: decision.Topic,
			TargetELO: decision.Difficulty,
			Rationale: decision.Why + " | probe: " + probe,
			ContextChunksJSON: string(contextJSON), AskedAt: time.Now(),
		}); err != nil {
			return err
		}
		if err := s.Store.InsertTurn(store.Turn{
			ID: uuid.NewString(), QuestionID: qID, Ord: 0,
			Speaker: "interviewer", Kind: "opening", Text: qText, Ts: time.Now(),
		}); err != nil {
			return err
		}

		emit(flush, "interviewer:opening", map[string]any{
			"question_id": qID,
			"text":        qText,
			"context":     contextRefs,
		})

		// register answer channel for this question
		d.mu.Lock()
		d.curQID = qID
		d.answer = make(chan string, 1)
		d.end = make(chan struct{})
		d.mu.Unlock()

		transcript := agents.Transcript{
			OpeningQ:  qText,
			Exchanges: []agents.Exchange{},
		}

		maxFu := s.Cfg.Drill.FollowupMax
		if maxFu <= 0 {
			maxFu = 3
		}

		// wait for first answer
		answer, err := d.waitAnswer(ctx)
		if err != nil {
			return err
		}
		if err := s.Store.InsertTurn(store.Turn{
			ID: uuid.NewString(), QuestionID: qID, Ord: 1,
			Speaker: "candidate", Kind: "answer", Text: answer, Ts: time.Now(),
		}); err != nil {
			return err
		}
		transcript.Exchanges = append(transcript.Exchanges, agents.Exchange{Answer: answer})

		turnOrd := 1
		for fu := 0; fu < maxFu; fu++ {
			emit(flush, "interviewer:deciding", map[string]any{"followups_remaining": maxFu - fu})
			step, err := iv.Step(ctx, agents.InterviewerInput{
				Profile: profile, Decision: decision, Context: ivContext, JDCard: jdCardJSON,
			}, transcript, maxFu-fu)
			if err != nil {
				return fmt.Errorf("interviewer step: %w", err)
			}
			if step.Action == "done" {
				emit(flush, "interviewer:done_with_question", map[string]any{})
				break
			}
			turnOrd++
			if err := s.Store.InsertTurn(store.Turn{
				ID: uuid.NewString(), QuestionID: qID, Ord: turnOrd,
				Speaker: "interviewer", Kind: "followup", Text: step.Text, Ts: time.Now(),
			}); err != nil {
				return err
			}
			transcript.Exchanges[len(transcript.Exchanges)-1].FollowupQ = step.Text
			emit(flush, "interviewer:followup", map[string]any{
				"index": fu + 1, "total": maxFu, "text": step.Text,
			})

			// wait for next answer
			a2, err := d.waitAnswer(ctx)
			if err != nil {
				return err
			}
			turnOrd++
			if err := s.Store.InsertTurn(store.Turn{
				ID: uuid.NewString(), QuestionID: qID, Ord: turnOrd,
				Speaker: "candidate", Kind: "answer", Text: a2, Ts: time.Now(),
			}); err != nil {
				return err
			}
			transcript.Exchanges = append(transcript.Exchanges, agents.Exchange{Answer: a2})
		}

		emit(flush, "judge:grading", map[string]any{})
		v, err := judge.Grade(ctx, agents.JudgeInput{
			Profile: profile, Decision: decision, Context: ivContext,
			Transcript: transcript, JDCard: jdCardJSON,
		})
		if err != nil {
			emit(flush, "judge:error", map[string]any{"message": err.Error()})
		} else {
			persistJudgment(s.Store, qID, decision.Category, v)
			emit(flush, "judge:verdict", v)

			before, _ := s.Store.GetELO(decision.Category, s.Cfg.Elo.StartRating)
			score := elo.RatingToScore(v.Rating)
			after, delta := elo.Update(before, decision.Difficulty, score, s.Cfg.Elo.KFactor)
			_ = s.Store.UpsertELO(decision.Category, after)
			_ = s.Store.InsertELOHistory(decision.Category, before, after, delta, qID)
			emit(flush, "elo:update", map[string]any{
				"category": decision.Category, "before": before, "after": after, "delta": delta,
			})
		}

		emit(flush, "question:end", map[string]any{"ord": i})
	}
	return nil
}

// waitAnswer blocks until an answer arrives via submitAnswer or the user ends
// the question via endQuestion, in which case the prior partial input
// (drained from the channel if any) is returned.
func (d *drillSession) waitAnswer(ctx context.Context) (string, error) {
	d.mu.Lock()
	ch := d.answer
	end := d.end
	d.mu.Unlock()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case a := <-ch:
		return a, nil
	case <-end:
		// drained partial answer if any
		select {
		case a := <-ch:
			return a, nil
		default:
			return "", nil
		}
	}
}

func persistJudgment(s *store.Store, qID, category string, v agents.Judgment) {
	st, _ := json.Marshal(v.Strengths)
	mi, _ := json.Marshal(v.Missed)
	rd, _ := json.Marshal(v.Reading)
	_ = s.InsertJudgment(store.Judgment{
		QuestionID: qID, Rating: v.Rating,
		StrengthsJSON: string(st), MissedJSON: string(mi),
		BetterSketch: v.BetterAnswerSketch, ReadingJSON: string(rd),
		GradedAt: time.Now(),
	})
	for _, tag := range v.TopicTags {
		_ = s.InsertTopicHit(qID, tag, category, v.Rating)
	}
}

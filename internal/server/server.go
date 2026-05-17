package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Prasad-178/reps/internal/agents"
	"github.com/Prasad-178/reps/internal/config"
	"github.com/Prasad-178/reps/internal/llm"
	"github.com/Prasad-178/reps/internal/orchestrator"
	"github.com/Prasad-178/reps/internal/store"
)

type Server struct {
	Cfg    config.Config
	Store  *store.Store
	Client *llm.Client
	Orch   *orchestrator.Orchestrator

	// CORS allowlist (origins). Defaults to localhost dev origins.
	Origins []string
}

func New(cfg config.Config, s *store.Store, c *llm.Client) *Server {
	return &Server{
		Cfg:    cfg,
		Store:  s,
		Client: c,
		Orch:   orchestrator.New(cfg, s, c),
		Origins: []string{
			"http://localhost:3000",
			"http://127.0.0.1:3000",
			"http://localhost:3001",
		},
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /api/stats", s.stats)
	mux.HandleFunc("GET /api/elo", s.eloHistory)
	mux.HandleFunc("GET /api/sources", s.sources)
	mux.HandleFunc("GET /api/jds", s.jds)
	mux.HandleFunc("GET /api/profile", s.profile)
	mux.HandleFunc("GET /api/sessions", s.sessions)
	mux.HandleFunc("GET /api/sessions/{id}", s.replay)
	mux.HandleFunc("GET /api/plans", s.plans)
	mux.HandleFunc("GET /api/plans/latest", s.latestPlan)

	mux.HandleFunc("GET /api/drill/stream", s.drillStream)
	mux.HandleFunc("POST /api/drill/{session}/{question}/answer", s.drillAnswer)
	mux.HandleFunc("POST /api/drill/{session}/{question}/end", s.drillEnd)

	return s.withCORS(s.withLog(mux))
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && s.originAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) originAllowed(o string) bool {
	for _, a := range s.Origins {
		if a == o {
			return true
		}
	}
	return false
}

func (s *Server) withLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%-4s %-40s %s", r.Method, r.URL.Path, time.Since(t))
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

func writeErr(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

// ---- handlers

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]any{"ok": true, "version": "0.0.1"})
}

func (s *Server) stats(w http.ResponseWriter, _ *http.Request) {
	elo, err := s.Store.GetAllELO()
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	delta, err := s.Store.ELODeltaSince(time.Now().Add(-7 * 24 * time.Hour))
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	weak, err := s.Store.WeakestTopics(10)
	if err != nil {
		writeErr(w, 500, err)
		return
	}

	type cat struct {
		Rating  int `json:"rating"`
		Delta7d int `json:"delta_7d"`
	}
	byCat := map[string]cat{}
	overall := 0
	n := 0
	for _, c := range agents.Categories {
		r, ok := elo[c]
		if !ok {
			r = s.Cfg.Elo.StartRating
		}
		byCat[c] = cat{Rating: r, Delta7d: delta[c]}
		overall += r
		n++
	}
	avg := 0
	if n > 0 {
		avg = overall / n
	}
	type weakTopic struct {
		Tag        string  `json:"tag"`
		Hits       int     `json:"hits"`
		MeanRating float64 `json:"mean_rating"`
	}
	ws := make([]weakTopic, 0, len(weak))
	for _, w := range weak {
		ws = append(ws, weakTopic{Tag: w.Tag, Hits: w.Hits, MeanRating: w.MeanRating})
	}
	writeJSON(w, 200, map[string]any{
		"overall":     avg,
		"by_category": byCat,
		"weakest":     ws,
	})
}

func (s *Server) eloHistory(w http.ResponseWriter, r *http.Request) {
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 {
		days = 30
	}
	rows, err := s.Store.DB.Query(`SELECT at_ts, category, rating_after
		FROM elo_history WHERE at_ts >= ? ORDER BY at_ts`,
		time.Now().Add(-time.Duration(days)*24*time.Hour).Unix())
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	defer rows.Close()
	type point struct {
		At       int64  `json:"at"`
		Category string `json:"category"`
		Rating   int    `json:"rating"`
	}
	out := []point{}
	for rows.Next() {
		var p point
		if err := rows.Scan(&p.At, &p.Category, &p.Rating); err != nil {
			writeErr(w, 500, err)
			return
		}
		out = append(out, p)
	}
	writeJSON(w, 200, out)
}

func (s *Server) sources(w http.ResponseWriter, _ *http.Request) {
	src, err := s.Store.ListSources()
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	type o struct {
		ID        string `json:"id"`
		Kind      string `json:"kind"`
		Ref       string `json:"ref"`
		FetchedAt int64  `json:"fetched_at"`
	}
	out := make([]o, 0, len(src))
	for _, s := range src {
		out = append(out, o{ID: s.ID, Kind: s.Kind, Ref: s.Ref, FetchedAt: s.FetchedAt.Unix()})
	}
	writeJSON(w, 200, out)
}

func (s *Server) jds(w http.ResponseWriter, _ *http.Request) {
	jds, err := s.Store.ListJDCards()
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	type o struct {
		ID      string         `json:"id"`
		Company string         `json:"company"`
		Role    string         `json:"role"`
		Card    map[string]any `json:"card"`
	}
	out := make([]o, 0, len(jds))
	for _, j := range jds {
		var card map[string]any
		_ = json.Unmarshal([]byte(j.CardJSON), &card)
		out = append(out, o{ID: j.ID, Company: j.Company, Role: j.Role, Card: card})
	}
	writeJSON(w, 200, out)
}

func (s *Server) profile(w http.ResponseWriter, _ *http.Request) {
	md, built, model, err := s.Store.GetProfile()
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{
		"markdown":   md,
		"built_at":   built.Unix(),
		"model_used": model,
	})
}

func (s *Server) sessions(w http.ResponseWriter, _ *http.Request) {
	ss, err := s.Store.RecentSessions(50)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	type o struct {
		ID         string `json:"id"`
		StartedAt  int64  `json:"started_at"`
		EndedAt    *int64 `json:"ended_at"`
		Mode       string `json:"mode"`
		QCount     int    `json:"q_count"`
		MeanRating float64 `json:"mean_rating"`
	}
	out := make([]o, 0, len(ss))
	for _, s := range ss {
		var end *int64
		if s.EndedAt != nil {
			t := s.EndedAt.Unix()
			end = &t
		}
		out = append(out, o{
			ID: s.ID, StartedAt: s.StartedAt.Unix(), EndedAt: end,
			Mode: s.Mode, QCount: s.QCount, MeanRating: s.MeanRate,
		})
	}
	writeJSON(w, 200, out)
}

func (s *Server) replay(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeErr(w, 400, fmt.Errorf("missing session id"))
		return
	}
	// resolve prefix
	full := id
	if len(id) < 36 {
		row := s.Store.DB.QueryRow(`SELECT id FROM sessions WHERE id LIKE ?`, id+"%")
		if err := row.Scan(&full); err != nil {
			writeErr(w, 404, fmt.Errorf("session %q not found", id))
			return
		}
	}

	var sess struct {
		ID         string  `json:"id"`
		StartedAt  int64   `json:"started_at"`
		EndedAt    *int64  `json:"ended_at"`
		Mode       string  `json:"mode"`
		QCount     int     `json:"q_count"`
		MeanRating float64 `json:"mean_rating"`
	}
	var endedAt *int64
	row := s.Store.DB.QueryRow(`
		SELECT s.id, s.started_at, s.ended_at, COALESCE(s.mode,''),
		  (SELECT COUNT(*) FROM questions q WHERE q.session_id = s.id),
		  COALESCE((SELECT AVG(rating) FROM judgments j JOIN questions q ON q.id=j.question_id WHERE q.session_id = s.id), 0)
		FROM sessions s WHERE s.id = ?`, full)
	if err := row.Scan(&sess.ID, &sess.StartedAt, &endedAt, &sess.Mode, &sess.QCount, &sess.MeanRating); err != nil {
		writeErr(w, 500, err)
		return
	}
	sess.EndedAt = endedAt

	qs, err := s.Store.QuestionsBySession(full)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	type turn struct {
		Ord     int    `json:"ord"`
		Speaker string `json:"speaker"`
		Kind    string `json:"kind"`
		Text    string `json:"text"`
	}
	type judgmentOut struct {
		Rating       int                      `json:"rating"`
		Strengths    []string                 `json:"strengths"`
		Missed       []string                 `json:"missed"`
		BetterSketch string                   `json:"better_sketch"`
		Reading      []map[string]string      `json:"reading"`
		TopicTags    []string                 `json:"topic_tags"`
	}
	type qOut struct {
		Ord       int          `json:"ord"`
		Category  string       `json:"category"`
		Topic     string       `json:"topic"`
		TargetELO int          `json:"target_elo"`
		Rationale string       `json:"rationale"`
		Turns     []turn       `json:"turns"`
		Judgment  *judgmentOut `json:"judgment,omitempty"`
	}
	out := make([]qOut, 0, len(qs))
	for _, q := range qs {
		o := qOut{Ord: q.Ord, Category: q.Category, Topic: q.TargetTopic,
			TargetELO: q.TargetELO, Rationale: q.Rationale}
		turns, _ := s.Store.ListTurnsForQuestion(q.ID)
		for _, t := range turns {
			o.Turns = append(o.Turns, turn{Ord: t.Ord, Speaker: t.Speaker, Kind: t.Kind, Text: t.Text})
		}
		if j, ok, _ := s.Store.GetJudgment(q.ID); ok {
			jo := judgmentOut{Rating: j.Rating, BetterSketch: j.BetterSketch}
			_ = json.Unmarshal([]byte(j.StrengthsJSON), &jo.Strengths)
			_ = json.Unmarshal([]byte(j.MissedJSON), &jo.Missed)
			var reading []map[string]string
			_ = json.Unmarshal([]byte(j.ReadingJSON), &reading)
			jo.Reading = reading
			o.Judgment = &jo
		}
		out = append(out, o)
	}
	writeJSON(w, 200, map[string]any{
		"session":   sess,
		"questions": out,
	})
}

func (s *Server) plans(w http.ResponseWriter, _ *http.Request) {
	rows, err := s.Store.DB.Query(`SELECT id,generated_at,window_days,markdown FROM plans ORDER BY generated_at DESC LIMIT 50`)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	defer rows.Close()
	type o struct {
		ID          string `json:"id"`
		GeneratedAt int64  `json:"generated_at"`
		WindowDays  int    `json:"window_days"`
		Markdown    string `json:"markdown"`
	}
	out := []o{}
	for rows.Next() {
		var p o
		if err := rows.Scan(&p.ID, &p.GeneratedAt, &p.WindowDays, &p.Markdown); err != nil {
			writeErr(w, 500, err)
			return
		}
		out = append(out, p)
	}
	writeJSON(w, 200, out)
}

func (s *Server) latestPlan(w http.ResponseWriter, _ *http.Request) {
	var p struct {
		ID          string `json:"id"`
		GeneratedAt int64  `json:"generated_at"`
		WindowDays  int    `json:"window_days"`
		Markdown    string `json:"markdown"`
	}
	err := s.Store.DB.QueryRow(`SELECT id,generated_at,window_days,markdown FROM plans ORDER BY generated_at DESC LIMIT 1`).
		Scan(&p.ID, &p.GeneratedAt, &p.WindowDays, &p.Markdown)
	if err != nil {
		writeJSON(w, 200, nil)
		return
	}
	writeJSON(w, 200, p)
}

// ---- drill stream (SSE) — placeholder. M3-M5 orchestrator is interactive
// and reads stdin; for the web flow we run a stripped-down variant that
// posts answers via HTTP instead of stdin. Wired in drill_stream.go.

func (s *Server) drillStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, 500, fmt.Errorf("streaming unsupported"))
		return
	}
	flush := func(event, data string) {
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
		flusher.Flush()
	}

	q := r.URL.Query()
	qs, _ := strconv.Atoi(q.Get("qs"))
	if qs == 0 {
		qs = s.Cfg.Drill.DefaultQs
	}
	cat := q.Get("category")
	topic := q.Get("topic")
	jd := q.Get("jd")
	diff, _ := strconv.Atoi(q.Get("difficulty"))

	flush("hello", fmt.Sprintf(`{"qs":%d}`, qs))

	ds := newDrillSession(s, drillOpts{
		Qs: qs, Category: cat, Topic: topic, JD: jd, Difficulty: diff,
	})
	registerDrillSession(ds)
	defer unregisterDrillSession(ds.SessionID)

	flush("session", fmt.Sprintf(`{"session_id":%q}`, ds.SessionID))

	ctx := r.Context()
	if err := ds.Run(ctx, flush); err != nil {
		flush("error", fmt.Sprintf(`{"message":%q}`, err.Error()))
		return
	}
	flush("done", "{}")
}

func (s *Server) drillAnswer(w http.ResponseWriter, r *http.Request) {
	sess := r.PathValue("session")
	qID := r.PathValue("question")
	var body struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeErr(w, 400, err)
		return
	}
	if err := submitAnswer(sess, qID, body.Text); err != nil {
		writeErr(w, 404, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) drillEnd(w http.ResponseWriter, r *http.Request) {
	sess := r.PathValue("session")
	qID := r.PathValue("question")
	if err := endQuestion(sess, qID); err != nil {
		writeErr(w, 404, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}


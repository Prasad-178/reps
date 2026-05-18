package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/Prasad-178/reps/internal/agents"
)

var (
	insightsMu      sync.Mutex
	insightsCache   agents.AnalystOutput
	insightsBuiltAt time.Time
)

const insightsTTL = 10 * time.Minute

func (s *Server) insights(w http.ResponseWriter, r *http.Request) {
	force := r.URL.Query().Get("force") == "1"

	insightsMu.Lock()
	if !force && len(insightsCache.Panels) > 0 && time.Since(insightsBuiltAt) < insightsTTL {
		out := insightsCache
		builtAt := insightsBuiltAt.Unix()
		insightsMu.Unlock()
		writeJSON(w, 200, map[string]any{
			"summary":  out.Summary,
			"panels":   out.Panels,
			"built_at": builtAt,
			"cached":   true,
		})
		return
	}
	insightsMu.Unlock()

	if s.Client.APIKey == "" {
		writeJSON(w, 200, map[string]any{
			"summary": "",
			"panels":  []agents.InsightPanel{},
			"error":   "no API key set",
		})
		return
	}

	// gather inputs
	profile, _, _, _ := s.Store.GetProfile()
	jds, _ := s.Store.ListJDCards()
	elo, _ := s.Store.GetAllELO()
	if elo == nil {
		elo = map[string]int{}
	}
	for _, c := range agents.Categories {
		if _, ok := elo[c]; !ok {
			elo[c] = s.Cfg.Elo.StartRating
		}
	}
	weak, _ := s.Store.WeakestTopics(15)

	jdSum := make([]agents.JDSummary, 0, len(jds))
	for _, j := range jds {
		var card struct{ Summary string `json:"summary"` }
		_ = json.Unmarshal([]byte(j.CardJSON), &card)
		jdSum = append(jdSum, agents.JDSummary{ID: j.ID, Company: j.Company, Role: j.Role, Summary: card.Summary})
	}
	weakMapped := make([]agents.WeakTopic, 0, len(weak))
	for _, w := range weak {
		weakMapped = append(weakMapped, agents.WeakTopic{Tag: w.Tag, Hits: w.Hits, MeanRating: w.MeanRating})
	}

	recents, _ := s.Store.RecentSessions(20)
	recentSum := make([]agents.RecentDrillSummary, 0, len(recents))
	for _, sn := range recents {
		qs, _ := s.Store.QuestionsBySession(sn.ID)
		topics := make([]string, 0, len(qs))
		ratings := make([]int, 0, len(qs))
		cats := make([]string, 0, len(qs))
		for _, q := range qs {
			topics = append(topics, q.TargetTopic)
			cats = append(cats, q.Category)
			if j, ok, _ := s.Store.GetJudgment(q.ID); ok {
				ratings = append(ratings, j.Rating)
			} else {
				ratings = append(ratings, 0)
			}
		}
		recentSum = append(recentSum, agents.RecentDrillSummary{
			StartedAt: sn.StartedAt.Unix(), Mode: sn.Mode,
			Topics: topics, Ratings: ratings, Categories: cats,
		})
	}

	analyst := agents.NewAnalyst(s.Client)
	out, err := analyst.Insights(r.Context(), agents.AnalystInput{
		Profile:      profile,
		JDCards:      jdSum,
		CategoryELO:  elo,
		WeakTopics:   weakMapped,
		RecentDrills: recentSum,
	})
	if err != nil {
		writeErr(w, 500, err)
		return
	}

	insightsMu.Lock()
	insightsCache = out
	insightsBuiltAt = time.Now()
	insightsMu.Unlock()

	writeJSON(w, 200, map[string]any{
		"summary":  out.Summary,
		"panels":   out.Panels,
		"built_at": insightsBuiltAt.Unix(),
		"cached":   false,
	})
}

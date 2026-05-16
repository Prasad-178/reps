package repscli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Prasad-178/reps/internal/store"
	"github.com/urfave/cli/v3"
)

func exportAction(ctx context.Context, c *cli.Command) error {
	_, s, _, err := openCtx(ctx)
	if err != nil {
		return err
	}
	defer s.Close()
	if c.Bool("json") {
		return exportJSON(s)
	}
	return exportMarkdown(s)
}

func exportJSON(s *store.Store) error {
	type qOut struct {
		Ord       int                 `json:"ord"`
		Category  string              `json:"category"`
		Topic     string              `json:"topic"`
		TargetELO int                 `json:"target_elo"`
		Rationale string              `json:"rationale"`
		Turns     []map[string]string `json:"turns"`
		Judgment  map[string]any      `json:"judgment,omitempty"`
	}
	type sOut struct {
		ID        string `json:"id"`
		StartedAt int64  `json:"started_at"`
		Mode      string `json:"mode"`
		Questions []qOut `json:"questions"`
	}
	type all struct {
		Profile  string `json:"profile,omitempty"`
		Sessions []sOut `json:"sessions"`
	}
	var payload all
	if md, _, _, err := s.GetProfile(); err == nil {
		payload.Profile = md
	}
	sessions, err := s.RecentSessions(10000)
	if err != nil {
		return err
	}
	for _, ss := range sessions {
		out := sOut{ID: ss.ID, StartedAt: ss.StartedAt.Unix(), Mode: ss.Mode}
		qs, err := s.QuestionsBySession(ss.ID)
		if err != nil {
			return err
		}
		for _, q := range qs {
			eq := qOut{
				Ord: q.Ord, Category: q.Category, Topic: q.TargetTopic,
				TargetELO: q.TargetELO, Rationale: q.Rationale,
			}
			turns, _ := s.ListTurnsForQuestion(q.ID)
			for _, t := range turns {
				eq.Turns = append(eq.Turns, map[string]string{
					"speaker": t.Speaker, "kind": t.Kind, "text": t.Text,
				})
			}
			if j, ok, _ := s.GetJudgment(q.ID); ok {
				var st, ms []string
				_ = json.Unmarshal([]byte(j.StrengthsJSON), &st)
				_ = json.Unmarshal([]byte(j.MissedJSON), &ms)
				eq.Judgment = map[string]any{
					"rating": j.Rating, "strengths": st, "missed": ms,
					"better": j.BetterSketch,
				}
			}
			out.Questions = append(out.Questions, eq)
		}
		payload.Sessions = append(payload.Sessions, out)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func exportMarkdown(s *store.Store) error {
	if md, built, model, err := s.GetProfile(); err == nil && md != "" {
		fmt.Printf("# Profile (built %s, model %s)\n\n%s\n\n---\n\n",
			built.Format("2006-01-02"), model, md)
	}
	sessions, err := s.RecentSessions(10000)
	if err != nil {
		return err
	}
	for _, ss := range sessions {
		fmt.Printf("# Session %s — %s (%s)\n\n",
			ss.ID[:8], ss.StartedAt.Format("2006-01-02 15:04"), ss.Mode)
		qs, err := s.QuestionsBySession(ss.ID)
		if err != nil {
			return err
		}
		for _, q := range qs {
			fmt.Printf("## Q%d — %s | %s (target ELO %d)\n",
				q.Ord, q.Category, q.TargetTopic, q.TargetELO)
			fmt.Printf("> %s\n\n", q.Rationale)
			turns, _ := s.ListTurnsForQuestion(q.ID)
			for _, t := range turns {
				role := titleCase(t.Speaker)
				if t.Kind != "" {
					role += " (" + t.Kind + ")"
				}
				fmt.Printf("**%s:** %s\n\n", role, t.Text)
			}
			if j, ok, _ := s.GetJudgment(q.ID); ok {
				var st, ms []string
				_ = json.Unmarshal([]byte(j.StrengthsJSON), &st)
				_ = json.Unmarshal([]byte(j.MissedJSON), &ms)
				fmt.Printf("**Judgment %d/5**\n\n", j.Rating)
				if len(st) > 0 {
					fmt.Println("Strengths:")
					for _, x := range st {
						fmt.Printf("- %s\n", x)
					}
				}
				if len(ms) > 0 {
					fmt.Println("Missed:")
					for _, x := range ms {
						fmt.Printf("- %s\n", x)
					}
				}
				if j.BetterSketch != "" {
					fmt.Printf("\n_Better answer sketch:_ %s\n\n", j.BetterSketch)
				}
			}
		}
		fmt.Println()
	}
	return nil
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

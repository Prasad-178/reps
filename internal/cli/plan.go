package repscli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Prasad-178/reps/internal/agents"
	"github.com/google/uuid"
	"github.com/urfave/cli/v3"
)

func planAction(ctx context.Context, c *cli.Command) error {
	cfg, s, client, err := openCtx(ctx)
	if err != nil {
		return err
	}
	defer s.Close()
	if cfg.LLM.APIKey == "" {
		return fmt.Errorf("OPENROUTER_API_KEY not set")
	}
	days := int(c.Int("days"))
	if days <= 0 {
		days = 30
	}
	since := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	topics, err := s.AggregateTopics(since)
	if err != nil {
		return err
	}
	if len(topics) == 0 {
		fmt.Println("(no topic hits in window — drill a few sessions first)")
		return nil
	}
	profile, _, _, err := s.GetProfile()
	if err != nil {
		return err
	}
	jds, err := s.ListJDCards()
	if err != nil {
		return err
	}
	jdSum := make([]agents.JDSummary, 0, len(jds))
	for _, j := range jds {
		var card struct{ Summary string `json:"summary"` }
		_ = json.Unmarshal([]byte(j.CardJSON), &card)
		jdSum = append(jdSum, agents.JDSummary{
			ID: j.ID, Company: j.Company, Role: j.Role, Summary: card.Summary,
		})
	}
	coachTopics := make([]agents.CoachTopic, 0, len(topics))
	for _, t := range topics {
		coachTopics = append(coachTopics, agents.CoachTopic{
			Tag: t.Tag, Hits: t.Hits, MeanRating: t.MeanRating,
			LastSeenAgo: agents.FormatLastSeen(t.LastSeen), Categories: t.Categories,
		})
	}
	coach := agents.NewCoach(client)
	fmt.Printf("Coach synthesizing plan over last %d days (%d topics)...\n", days, len(coachTopics))
	md, err := coach.Synthesize(ctx, agents.CoachInput{
		Profile: profile, WindowDays: days, JDCards: jdSum, Topics: coachTopics,
	})
	if err != nil {
		return err
	}
	id := uuid.NewString()
	if err := s.InsertPlan(id, md, days); err != nil {
		return err
	}
	planPath := filepath.Join(cfg.Paths.Plans, time.Now().Format("2006-01-02")+"-"+id[:8]+".md")
	if err := os.WriteFile(planPath, []byte(md), 0o644); err != nil {
		return err
	}
	fmt.Printf("\n%s\n", md)
	fmt.Printf("\nSaved to %s (id=%s)\n", planPath, id[:8])
	return nil
}

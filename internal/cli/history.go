package repscli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Prasad-178/reps/internal/config"
	"github.com/Prasad-178/reps/internal/llm"
	"github.com/Prasad-178/reps/internal/store"
	"github.com/urfave/cli/v3"
)

func historyAction(ctx context.Context, c *cli.Command) error {
	cfg, s, _, err := openStoreOnly(ctx)
	if err != nil {
		return err
	}
	defer s.Close()
	_ = cfg
	n := int(c.Int("last"))
	if n <= 0 {
		n = 10
	}
	sessions, err := s.RecentSessions(n)
	if err != nil {
		return err
	}
	if len(sessions) == 0 {
		fmt.Println("(no drills yet — run `reps drill`)")
		return nil
	}
	fmt.Printf("%-10s  %-19s  %-5s  %-4s  %s\n", "id", "started", "mode", "qs", "mean")
	for _, ss := range sessions {
		mean := "-"
		if ss.MeanRate > 0 {
			mean = fmt.Sprintf("%.2f", ss.MeanRate)
		}
		fmt.Printf("%-10s  %-19s  %-5s  %-4d  %s\n",
			ss.ID[:8],
			ss.StartedAt.Format("2006-01-02 15:04:05"),
			ss.Mode, ss.QCount, mean)
	}
	return nil
}

func replayAction(ctx context.Context, c *cli.Command) error {
	id := c.Args().First()
	if id == "" {
		return fmt.Errorf("usage: reps replay <session-id> (use prefix of id from `reps history`)")
	}
	_, s, _, err := openStoreOnly(ctx)
	if err != nil {
		return err
	}
	defer s.Close()
	full, err := resolveSessionID(s, id)
	if err != nil {
		return err
	}
	qs, err := s.QuestionsBySession(full)
	if err != nil {
		return err
	}
	if len(qs) == 0 {
		fmt.Println("(no questions in this session)")
		return nil
	}
	for _, q := range qs {
		fmt.Printf("─── Q%d ── %s | topic=%q | target_elo=%d\n", q.Ord, q.Category, q.TargetTopic, q.TargetELO)
		fmt.Printf("  why: %s\n", q.Rationale)
		turns, err := s.ListTurnsForQuestion(q.ID)
		if err != nil {
			return err
		}
		for _, t := range turns {
			fmt.Printf("  [%s/%s] %s\n", t.Speaker, t.Kind, indent(t.Text, "    "))
		}
		j, ok, err := s.GetJudgment(q.ID)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("  (no judgment)")
			fmt.Println()
			continue
		}
		var strengths, missed []string
		_ = json.Unmarshal([]byte(j.StrengthsJSON), &strengths)
		_ = json.Unmarshal([]byte(j.MissedJSON), &missed)
		fmt.Printf("  Rating: %d/5\n", j.Rating)
		for _, st := range strengths {
			fmt.Printf("    + %s\n", st)
		}
		for _, m := range missed {
			fmt.Printf("    - %s\n", m)
		}
		if j.BetterSketch != "" {
			fmt.Printf("    Better: %s\n", j.BetterSketch)
		}
		fmt.Println()
	}
	return nil
}

func resolveSessionID(s *store.Store, prefix string) (string, error) {
	if len(prefix) >= 36 {
		return prefix, nil
	}
	rows, err := s.DB.Query(`SELECT id FROM sessions WHERE id LIKE ?`, prefix+"%")
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var matches []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", err
		}
		matches = append(matches, id)
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no session matching %q", prefix)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("%d sessions match %q — please use more characters", len(matches), prefix)
	}
}

func indent(s, pad string) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) == 1 {
		return lines[0]
	}
	for i := 1; i < len(lines); i++ {
		lines[i] = pad + lines[i]
	}
	return strings.Join(lines, "\n")
}

func openStoreOnly(ctx context.Context) (*config.Config, *store.Store, *llm.Client, error) {
	return openCtx(ctx)
}

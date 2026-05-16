package repscli

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/Prasad-178/reps/internal/agents"
	"github.com/urfave/cli/v3"
)

func statsAction(ctx context.Context, c *cli.Command) error {
	cfg, s, _, err := openCtx(ctx)
	if err != nil {
		return err
	}
	defer s.Close()
	elo, err := s.GetAllELO()
	if err != nil {
		return err
	}
	delta7, err := s.ELODeltaSince(time.Now().Add(-7 * 24 * time.Hour))
	if err != nil {
		return err
	}
	cats := agents.Categories
	overall := 0
	count := 0
	fmt.Println("ELO by category (7d delta):")
	rows := make([][3]string, 0, len(cats))
	for _, cat := range cats {
		r, ok := elo[cat]
		if !ok {
			r = cfg.Elo.StartRating
		}
		overall += r
		count++
		d := delta7[cat]
		sign := "+"
		if d < 0 {
			sign = ""
		}
		rows = append(rows, [3]string{cat, fmt.Sprintf("%d", r), fmt.Sprintf("%s%d", sign, d)})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i][0] < rows[j][0] })
	for _, r := range rows {
		fmt.Printf("  %-15s  %s   (7d: %s)\n", r[0], r[1], r[2])
	}
	if count > 0 {
		fmt.Printf("\nOverall ELO: %d (mean across %d categories)\n", overall/count, count)
	}
	weak, err := s.WeakestTopics(5)
	if err != nil {
		return err
	}
	if len(weak) > 0 {
		fmt.Println("\nWeakest topics (mean rating < 3.5 or ≥3 hits < 4):")
		for _, w := range weak {
			fmt.Printf("  • %-32s  hits=%d  mean=%.2f\n", w.Tag, w.Hits, w.MeanRating)
		}
	}
	return nil
}


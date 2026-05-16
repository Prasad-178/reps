package repscli

import (
	"context"
	"fmt"

	"github.com/Prasad-178/reps/internal/agents"
	"github.com/Prasad-178/reps/internal/orchestrator"
	"github.com/urfave/cli/v3"
)

func drillAction(ctx context.Context, c *cli.Command) error {
	cfg, s, client, err := openCtx(ctx)
	if err != nil {
		return err
	}
	defer s.Close()
	if cfg.LLM.APIKey == "" {
		return fmt.Errorf("OPENROUTER_API_KEY not set. Run `reps config llm.api_key sk-or-...` or export the env var.")
	}

	qs := int(c.Int("qs"))
	if qs == 0 {
		qs = cfg.Drill.DefaultQs
	}
	cat := c.String("category")
	if cat != "" && !agents.IsValidCategory(cat) {
		return fmt.Errorf("invalid --category %q. Valid: %v", cat, agents.Categories)
	}
	o := orchestrator.New(*cfg, s, client)
	return o.Run(ctx, orchestrator.Options{
		Qs:             qs,
		Voice:          c.Bool("voice"),
		CategoryFilter: cat,
		TopicOverride:  c.String("topic"),
		JDOverride:     c.String("jd"),
		DifficultyOver: int(c.Int("difficulty")),
	})
}

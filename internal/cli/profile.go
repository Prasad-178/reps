package repscli

import (
	"context"
	"fmt"

	"github.com/Prasad-178/reps/internal/config"
	"github.com/Prasad-178/reps/internal/ingest"
	"github.com/Prasad-178/reps/internal/llm"
	"github.com/Prasad-178/reps/internal/store"
	"github.com/urfave/cli/v3"
)

func ProfileCmd() *cli.Command {
	return &cli.Command{
		Name:  "profile",
		Usage: "print synthesized profile",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "rebuild", Usage: "re-ingest + re-summarize all sources"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := config.EnsureDirs(cfg); err != nil {
				return err
			}
			dim := llm.EmbedDim(cfg.LLM.EmbedModel)
			if dim == 0 {
				dim = 1536
			}
			s, err := store.Open(config.DBPath(cfg), dim)
			if err != nil {
				return err
			}
			defer s.Close()

			if c.Bool("rebuild") {
				if cfg.LLM.APIKey == "" {
					return fmt.Errorf("OPENROUTER_API_KEY not set")
				}
				client := llm.New(cfg.LLM.APIKey, cfg.LLM.Model, cfg.LLM.EmbedModel,
					cfg.LLM.JudgeModel, cfg.LLM.RerankModel)
				p := ingest.NewPipeline(cfg, s, client)
				if err := p.Rebuild(ctx); err != nil {
					return err
				}
			}

			md, built, model, err := s.GetProfile()
			if err != nil {
				return err
			}
			if md == "" {
				fmt.Println("(no profile yet — run `reps add ...` then `reps profile --rebuild`)")
				return nil
			}
			fmt.Printf("# Profile (built %s, model %s)\n\n", built.Format("2006-01-02 15:04"), model)
			fmt.Println(md)
			return nil
		},
	}
}

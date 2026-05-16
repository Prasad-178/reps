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

func openCtx(ctx context.Context) (*config.Config, *store.Store, *llm.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, nil, err
	}
	if err := config.EnsureDirs(cfg); err != nil {
		return nil, nil, nil, err
	}
	dim := llm.EmbedDim(cfg.LLM.EmbedModel)
	if dim == 0 {
		dim = 1536
	}
	s, err := store.Open(config.DBPath(cfg), dim)
	if err != nil {
		return nil, nil, nil, err
	}
	client := llm.New(cfg.LLM.APIKey, cfg.LLM.Model, cfg.LLM.EmbedModel,
		cfg.LLM.JudgeModel, cfg.LLM.RerankModel)
	return &cfg, s, client, nil
}

func addResumeAction(ctx context.Context, c *cli.Command) error {
	path := c.Args().First()
	if path == "" {
		return fmt.Errorf("usage: reps add resume <path>")
	}
	cfg, s, client, err := openCtx(ctx)
	if err != nil {
		return err
	}
	defer s.Close()
	p := ingest.NewPipeline(*cfg, s, client)
	id, err := p.IngestResume(ctx, path)
	if err != nil {
		return err
	}
	fmt.Printf("✓ resume ingested as source %s\n", id)
	fmt.Println("  next: run `reps profile --rebuild` to refresh embeddings + profile")
	return nil
}

func addGitHubAction(ctx context.Context, c *cli.Command) error {
	user := c.Args().First()
	if user == "" {
		return fmt.Errorf("usage: reps add github <user>")
	}
	cfg, s, client, err := openCtx(ctx)
	if err != nil {
		return err
	}
	defer s.Close()
	p := ingest.NewPipeline(*cfg, s, client)
	id, err := p.IngestGitHub(ctx, user)
	if err != nil {
		return err
	}
	fmt.Printf("✓ github '%s' ingested as source %s\n", user, id)
	fmt.Println("  next: run `reps profile --rebuild` to refresh embeddings + profile")
	return nil
}

func addPortfolioAction(ctx context.Context, c *cli.Command) error {
	url := c.Args().First()
	if url == "" {
		return fmt.Errorf("usage: reps add portfolio <url>")
	}
	cfg, s, client, err := openCtx(ctx)
	if err != nil {
		return err
	}
	defer s.Close()
	p := ingest.NewPipeline(*cfg, s, client)
	fmt.Printf("Fetching %s ...\n", url)
	id, err := p.IngestPortfolio(ctx, url)
	if err != nil {
		return err
	}
	fmt.Printf("✓ portfolio ingested as source %s\n", id)
	fmt.Println("  next: run `reps profile --rebuild` to refresh embeddings + profile")
	return nil
}

func addJDAction(ctx context.Context, c *cli.Command) error {
	url := c.Args().First()
	if url == "" {
		return fmt.Errorf("usage: reps add jd <url>")
	}
	cfg, s, client, err := openCtx(ctx)
	if err != nil {
		return err
	}
	defer s.Close()
	p := ingest.NewPipeline(*cfg, s, client)
	fmt.Printf("Fetching %s ...\n", url)
	id, err := p.IngestJD(ctx, url)
	if err != nil {
		return err
	}
	fmt.Printf("✓ JD ingested as source %s\n", id)
	fmt.Println("  next: run `reps profile --rebuild` to refresh embeddings + profile")
	return nil
}

func addLinkedInAction(ctx context.Context, c *cli.Command) error {
	ref := c.Args().First()
	if ref == "" {
		return fmt.Errorf("usage: reps add linkedin <url|@handle> [--from-file <path>]")
	}
	cfg, s, client, err := openCtx(ctx)
	if err != nil {
		return err
	}
	defer s.Close()
	p := ingest.NewPipeline(*cfg, s, client)
	id, err := p.IngestLinkedIn(ctx, ref, c.String("from-file"))
	if err != nil {
		return err
	}
	fmt.Printf("✓ linkedin paste ingested as source %s\n", id)
	fmt.Println("  next: run `reps profile --rebuild` to refresh embeddings + profile")
	return nil
}

func addXAction(ctx context.Context, c *cli.Command) error {
	handle := c.Args().First()
	if handle == "" {
		return fmt.Errorf("usage: reps add x <handle> [--from-file <path>]")
	}
	cfg, s, client, err := openCtx(ctx)
	if err != nil {
		return err
	}
	defer s.Close()
	p := ingest.NewPipeline(*cfg, s, client)
	id, err := p.IngestX(ctx, handle, c.String("from-file"))
	if err != nil {
		return err
	}
	fmt.Printf("✓ x paste ingested as source %s\n", id)
	fmt.Println("  next: run `reps profile --rebuild` to refresh embeddings + profile")
	return nil
}

func addNoteAction(ctx context.Context, c *cli.Command) error {
	path := c.Args().First()
	if path == "" {
		return fmt.Errorf("usage: reps add note <path>")
	}
	cfg, s, client, err := openCtx(ctx)
	if err != nil {
		return err
	}
	defer s.Close()
	p := ingest.NewPipeline(*cfg, s, client)
	id, err := p.IngestNote(ctx, path)
	if err != nil {
		return err
	}
	fmt.Printf("✓ note ingested as source %s\n", id)
	fmt.Println("  next: run `reps profile --rebuild` to refresh embeddings + profile")
	return nil
}

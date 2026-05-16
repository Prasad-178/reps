package repscli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Prasad-178/reps/internal/config"
	"github.com/Prasad-178/reps/internal/llm"
	"github.com/Prasad-178/reps/internal/store"
	"github.com/urfave/cli/v3"
)

func InitCmd() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "interactive personalization wizard",
		Action: func(ctx context.Context, _ *cli.Command) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := config.EnsureDirs(cfg); err != nil {
				return err
			}
			fmt.Println("reps init")
			fmt.Println("---------")
			fmt.Printf("Config:  %s\n", config.Path())
			fmt.Printf("Home:    %s\n", cfg.Paths.Home)
			fmt.Println()

			if cfg.LLM.APIKey == "" {
				fmt.Println("OPENROUTER_API_KEY not set. Set it in env or via:")
				fmt.Println("  reps config llm.api_key sk-or-...")
			}

			dim := llm.EmbedDim(cfg.LLM.EmbedModel)
			if dim == 0 {
				dim = 1536
				fmt.Printf("Unknown embed model '%s', defaulting dim=1536. Override with `reps config llm.embed_model ...`.\n", cfg.LLM.EmbedModel)
			}
			s, err := store.Open(config.DBPath(cfg), dim)
			if err != nil {
				return err
			}
			defer s.Close()
			fmt.Printf("DB:      %s (embed dim %d)\n", config.DBPath(cfg), dim)

			if err := config.Save(cfg); err != nil {
				return err
			}

			fmt.Println()
			fmt.Println("Add your sources to personalize the agent. Each prompt accepts an empty line to skip.")
			rd := bufio.NewReader(os.Stdin)

			collect := func(label string) string {
				fmt.Printf("%s: ", label)
				line, _ := rd.ReadString('\n')
				return strings.TrimSpace(line)
			}

			if v := collect("Resume PDF path"); v != "" {
				fmt.Printf("  (later) reps add resume %s\n", v)
			}
			if v := collect("Portfolio URL"); v != "" {
				fmt.Printf("  (later) reps add portfolio %s\n", v)
			}
			if v := collect("GitHub user"); v != "" {
				fmt.Printf("  (later) reps add github %s\n", v)
			}
			if v := collect("LinkedIn URL or @"); v != "" {
				fmt.Printf("  (later) reps add linkedin %s\n", v)
			}
			if v := collect("X handle"); v != "" {
				fmt.Printf("  (later) reps add x %s\n", v)
			}
			for {
				v := collect("JD URL (blank to finish)")
				if v == "" {
					break
				}
				fmt.Printf("  (later) reps add jd %s\n", v)
			}

			fmt.Println()
			fmt.Println("Init scaffold done. Run `reps add ...` to ingest, then `reps profile --rebuild`.")
			return nil
		},
	}
}

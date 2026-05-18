package repscli

import (
	"context"

	"github.com/Prasad-178/reps/internal/tui"
	"github.com/urfave/cli/v3"
)

func InitCmd() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "interactive setup wizard",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "reset",
				Usage: "wipe ~/.reps/* before running the wizard",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			return tui.RunInit(ctx, c.Root().Version, c.Bool("reset"))
		},
	}
}

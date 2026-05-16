package repscli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func stub(name string) cli.ActionFunc {
	return func(_ context.Context, _ *cli.Command) error {
		return fmt.Errorf("%s: not implemented yet", name)
	}
}

func AddCmd() *cli.Command {
	pasteFlag := &cli.StringFlag{Name: "from-file", Usage: "read pasted content from a file instead of stdin"}
	return &cli.Command{
		Name:  "add",
		Usage: "add a source to your profile (resume, portfolio, github, linkedin, x, jd, note)",
		Commands: []*cli.Command{
			{Name: "resume", Usage: "add resume PDF", ArgsUsage: "<path>", Action: addResumeAction},
			{Name: "portfolio", Usage: "add portfolio URL", ArgsUsage: "<url>", Action: addPortfolioAction},
			{Name: "github", Usage: "add GitHub user", ArgsUsage: "<user>", Action: addGitHubAction},
			{Name: "linkedin", Usage: "add LinkedIn (paste fallback — site blocks scrapers)",
				ArgsUsage: "<url|@>", Flags: []cli.Flag{pasteFlag}, Action: addLinkedInAction},
			{Name: "x", Usage: "add X handle (paste fallback)", ArgsUsage: "<handle>",
				Flags: []cli.Flag{pasteFlag}, Action: addXAction},
			{Name: "jd", Usage: "add job description URL", ArgsUsage: "<url>", Action: addJDAction},
			{Name: "note", Usage: "add a markdown note", ArgsUsage: "<path>", Action: addNoteAction},
		},
	}
}

func DrillCmd() *cli.Command {
	return &cli.Command{
		Name:  "drill",
		Usage: "run a drill session",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "voice", Usage: "use mic input via whisper.cpp (M9)"},
			&cli.StringFlag{Name: "category", Usage: "system-design|domain-crypto|domain-ml|domain-solana|jd-specific|general"},
			&cli.StringFlag{Name: "topic", Usage: "force a topic"},
			&cli.StringFlag{Name: "jd", Usage: "focus on one JD id"},
			&cli.IntFlag{Name: "qs", Usage: "number of questions (1-10)", Value: 0},
			&cli.IntFlag{Name: "difficulty", Usage: "override target ELO"},
		},
		Action: drillAction,
	}
}

func StatsCmd() *cli.Command {
	return &cli.Command{Name: "stats", Usage: "ELO + category breakdown", Action: statsAction}
}

func HistoryCmd() *cli.Command {
	return &cli.Command{
		Name:  "history",
		Usage: "recent drills",
		Flags: []cli.Flag{&cli.IntFlag{Name: "last", Usage: "show last N", Value: 10}},
		Action: historyAction,
	}
}

func PlanCmd() *cli.Command {
	return &cli.Command{
		Name:  "plan",
		Usage: "generate study plan",
		Flags: []cli.Flag{&cli.IntFlag{Name: "days", Usage: "lookback window", Value: 30}},
		Action: planAction,
	}
}

func ExportCmd() *cli.Command {
	return &cli.Command{
		Name:  "export",
		Usage: "dump corpus + drills",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "md"},
			&cli.BoolFlag{Name: "json"},
		},
		Action: exportAction,
	}
}

func ReplayCmd() *cli.Command {
	return &cli.Command{Name: "replay", Usage: "re-print a past session", ArgsUsage: "<session-id>", Action: replayAction}
}

func ResetCmd() *cli.Command {
	return &cli.Command{
		Name:  "reset",
		Usage: "wipe local data (requires --yes)",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "yes"},
			&cli.BoolFlag{Name: "all"},
			&cli.BoolFlag{Name: "data"},
			&cli.BoolFlag{Name: "sources"},
		},
		Action: resetAction,
	}
}

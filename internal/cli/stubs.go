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
	return &cli.Command{
		Name:  "add",
		Usage: "add a source to your profile (resume, portfolio, github, linkedin, x, jd, note)",
		Commands: []*cli.Command{
			{Name: "resume", Usage: "add resume PDF", ArgsUsage: "<path>", Action: addResumeAction},
			{Name: "portfolio", Usage: "add portfolio URL", ArgsUsage: "<url>", Action: stub("add portfolio")},
			{Name: "github", Usage: "add GitHub user", ArgsUsage: "<user>", Action: addGitHubAction},
			{Name: "linkedin", Usage: "add LinkedIn URL or paste", ArgsUsage: "<url|@>", Action: stub("add linkedin")},
			{Name: "x", Usage: "add X handle", ArgsUsage: "<handle>", Action: stub("add x")},
			{Name: "jd", Usage: "add job description URL", ArgsUsage: "<url>", Action: stub("add jd")},
			{Name: "note", Usage: "add a markdown note", ArgsUsage: "<path>", Action: stub("add note")},
		},
	}
}

func DrillCmd() *cli.Command {
	return &cli.Command{
		Name:  "drill",
		Usage: "run a drill session",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "voice", Usage: "use mic input via whisper.cpp"},
			&cli.StringFlag{Name: "category", Usage: "system-design|domain|jd|general"},
			&cli.StringFlag{Name: "topic", Usage: "force a topic"},
			&cli.StringFlag{Name: "jd", Usage: "focus on one JD id"},
			&cli.IntFlag{Name: "qs", Usage: "number of questions (1-10)", Value: 0},
			&cli.IntFlag{Name: "difficulty", Usage: "override target ELO"},
		},
		Action: stub("drill"),
	}
}

func StatsCmd() *cli.Command {
	return &cli.Command{Name: "stats", Usage: "ELO + category breakdown", Action: stub("stats")}
}

func HistoryCmd() *cli.Command {
	return &cli.Command{
		Name:  "history",
		Usage: "recent drills",
		Flags: []cli.Flag{&cli.IntFlag{Name: "last", Usage: "show last N", Value: 10}},
		Action: stub("history"),
	}
}

func PlanCmd() *cli.Command {
	return &cli.Command{
		Name:  "plan",
		Usage: "generate study plan",
		Flags: []cli.Flag{&cli.IntFlag{Name: "days", Usage: "lookback window", Value: 30}},
		Action: stub("plan"),
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
		Action: stub("export"),
	}
}

func ReplayCmd() *cli.Command {
	return &cli.Command{Name: "replay", Usage: "re-print a past session", ArgsUsage: "<session-id>", Action: stub("replay")}
}

func ResetCmd() *cli.Command {
	return &cli.Command{
		Name:  "reset",
		Usage: "wipe local data",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "yes"},
			&cli.BoolFlag{Name: "all"},
			&cli.BoolFlag{Name: "data"},
			&cli.BoolFlag{Name: "sources"},
		},
		Action: stub("reset"),
	}
}

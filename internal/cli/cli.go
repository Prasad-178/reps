package repscli

import "github.com/urfave/cli/v3"

func Commands() []*cli.Command {
	return []*cli.Command{
		InitCmd(),
		AddCmd(),
		ProfileCmd(),
		DrillCmd(),
		StatsCmd(),
		HistoryCmd(),
		PlanCmd(),
		ExportCmd(),
		ReplayCmd(),
		ConfigCmd(),
		ResetCmd(),
		ServeCmd(),
	}
}

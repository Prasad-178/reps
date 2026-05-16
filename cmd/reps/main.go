package main

import (
	"context"
	"fmt"
	"os"

	repscli "github.com/Prasad-178/reps/internal/cli"
	"github.com/urfave/cli/v3"
)

var (
	version = "0.0.1"
	commit  = "dev"
)

func main() {
	app := &cli.Command{
		Name:    "reps",
		Usage:   "personalized, agentic interview rehearsal CLI",
		Version: fmt.Sprintf("%s (%s)", version, commit),
		Commands: repscli.Commands(),
	}
	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

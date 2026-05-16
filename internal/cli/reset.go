package repscli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Prasad-178/reps/internal/config"
	"github.com/urfave/cli/v3"
)

func resetAction(ctx context.Context, c *cli.Command) error {
	if !c.Bool("yes") {
		return fmt.Errorf("reset requires --yes (this is destructive)")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if !looksLikeRepsHome(cfg.Paths.Home) {
		return fmt.Errorf("refusing to reset suspicious REPS_HOME=%q", cfg.Paths.Home)
	}

	all := c.Bool("all")
	wipeData := all || c.Bool("data")
	wipeSources := all || c.Bool("sources")
	if !wipeData && !wipeSources {
		return fmt.Errorf("nothing to do — pass --all, --data, or --sources")
	}

	if wipeData {
		dbPath := config.DBPath(cfg)
		fmt.Printf("Removing DB: %s\n", dbPath)
		if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		for _, ext := range []string{"-wal", "-shm"} {
			_ = os.Remove(dbPath + ext)
		}
	}
	if wipeSources {
		fmt.Printf("Removing sources dir: %s\n", cfg.Paths.Sources)
		if err := os.RemoveAll(cfg.Paths.Sources); err != nil {
			return err
		}
		_ = os.MkdirAll(cfg.Paths.Sources, 0o755)
	}
	if all {
		fmt.Printf("Removing plans dir: %s\n", cfg.Paths.Plans)
		_ = os.RemoveAll(cfg.Paths.Plans)
		_ = os.MkdirAll(cfg.Paths.Plans, 0o755)
		fmt.Printf("Removing tmp dir: %s\n", cfg.Paths.Tmp)
		_ = os.RemoveAll(cfg.Paths.Tmp)
		_ = os.MkdirAll(cfg.Paths.Tmp, 0o755)
	}
	fmt.Println("Done.")
	return nil
}

func looksLikeRepsHome(p string) bool {
	abs, err := filepath.Abs(p)
	if err != nil {
		return false
	}
	suspicious := []string{"/", "/usr", "/home", "/Users", "/Users/", "/etc", "/var", "/tmp", "."}
	for _, s := range suspicious {
		if abs == s {
			return false
		}
	}
	if abs == "" {
		return false
	}
	base := strings.ToLower(filepath.Base(abs))
	return strings.Contains(base, "reps")
}

package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// LoadDotenv loads KEY=VALUE pairs from .env-style files into the process
// environment, but never overwrites an existing env var (so explicit
// `export` and shell wrappers always win).
//
// Search order (first hit wins per key):
//  1. $REPS_ENV_FILE (explicit override)
//  2. ./.env in the current working directory
//  3. ~/.reps/.env (or $REPS_HOME/.env)
//
// Comments (#…) and blank lines are ignored. Optional surrounding quotes
// on the value are stripped. `export KEY=VAL` is tolerated.
//
// Errors are silent on purpose — a missing .env is the common case.
func LoadDotenv() {
	for _, p := range candidates() {
		if p == "" {
			continue
		}
		if err := loadFile(p); err == nil {
			// keep going so later files can fill keys the earlier ones missed
			continue
		}
	}
}

func candidates() []string {
	out := []string{}
	if v := os.Getenv("REPS_ENV_FILE"); v != "" {
		out = append(out, expandDotenv(v))
	}
	if cwd, err := os.Getwd(); err == nil {
		out = append(out, filepath.Join(cwd, ".env"))
	}
	out = append(out, filepath.Join(defaultHome(), ".env"))
	return out
}

func expandDotenv(p string) string {
	if strings.HasPrefix(p, "~/") {
		if h, err := os.UserHomeDir(); err == nil {
			return filepath.Join(h, p[2:])
		}
	}
	return p
}

func loadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		val = stripQuotes(val)
		if key == "" {
			continue
		}
		if _, ok := os.LookupEnv(key); ok {
			continue
		}
		_ = os.Setenv(key, val)
	}
	return sc.Err()
}

func stripQuotes(s string) string {
	if len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

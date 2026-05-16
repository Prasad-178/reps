package rubrics

import (
	"embed"
	"fmt"
	"path"
	"strings"
)

//go:embed *.yaml
var fs embed.FS

// Load returns the raw rubric YAML for a category. The Judge prompt receives
// it verbatim — no need to parse it server-side.
func Load(category string) (string, error) {
	name := normalize(category) + ".yaml"
	b, err := fs.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("rubric for %q not found", category)
	}
	return string(b), nil
}

// Available lists rubric basenames found in the embed.
func Available() ([]string, error) {
	entries, err := fs.ReadDir(".")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		n := e.Name()
		if strings.HasSuffix(n, ".yaml") {
			out = append(out, strings.TrimSuffix(path.Base(n), ".yaml"))
		}
	}
	return out, nil
}

func normalize(c string) string {
	c = strings.ToLower(strings.TrimSpace(c))
	c = strings.ReplaceAll(c, "_", "-")
	return c
}

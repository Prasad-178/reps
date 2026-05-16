package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Prasad-178/reps/internal/store"
	"github.com/google/uuid"
)

type ghRepo struct {
	Name        string `json:"name"`
	NameWithOwner string `json:"nameWithOwner"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Stars       int    `json:"stargazerCount"`
	Lang        struct {
		Name string `json:"name"`
	} `json:"primaryLanguage"`
	IsFork    bool      `json:"isFork"`
	IsArchived bool     `json:"isArchived"`
	PushedAt  time.Time `json:"pushedAt"`
}

func (p *Pipeline) IngestGitHub(ctx context.Context, user string) (string, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return "", fmt.Errorf("`gh` not on PATH — install GitHub CLI and `gh auth login`")
	}
	repos, err := ghListRepos(ctx, user)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "# GitHub: %s\n\n", user)
	for _, r := range repos {
		if r.IsFork || r.IsArchived {
			continue
		}
		fmt.Fprintf(&sb, "## %s\n", r.NameWithOwner)
		fmt.Fprintf(&sb, "URL: %s\n", r.URL)
		if r.Lang.Name != "" {
			fmt.Fprintf(&sb, "Language: %s\n", r.Lang.Name)
		}
		fmt.Fprintf(&sb, "Stars: %d\n", r.Stars)
		if r.Description != "" {
			fmt.Fprintf(&sb, "Description: %s\n", r.Description)
		}
		readme, err := ghFetchReadme(ctx, r.NameWithOwner)
		if err == nil && readme != "" {
			sb.WriteString("\n--- README ---\n")
			sb.WriteString(readme)
			sb.WriteString("\n--- /README ---\n")
		}
		sb.WriteString("\n")
	}
	id := uuid.NewString()
	rawPath := filepath.Join(p.cfg.Paths.Sources, id+".txt")
	if err := os.WriteFile(rawPath, []byte(sb.String()), 0o644); err != nil {
		return "", err
	}
	meta, _ := json.Marshal(map[string]any{"repo_count": len(repos)})
	return id, p.store.InsertSource(store.Source{
		ID: id, Kind: "github", Ref: user, RawPath: rawPath,
		FetchedAt: time.Now(), MetaJSON: string(meta),
	})
}

func ghListRepos(ctx context.Context, user string) ([]ghRepo, error) {
	cmd := exec.CommandContext(ctx, "gh", "repo", "list", user,
		"--limit", "100",
		"--json", "name,nameWithOwner,description,url,stargazerCount,primaryLanguage,isFork,isArchived,pushedAt")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh repo list: %w", err)
	}
	var repos []ghRepo
	if err := json.Unmarshal(out, &repos); err != nil {
		return nil, fmt.Errorf("decode repo list: %w", err)
	}
	return repos, nil
}

func ghFetchReadme(ctx context.Context, nameWithOwner string) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", "api",
		fmt.Sprintf("repos/%s/readme", nameWithOwner),
		"-H", "Accept: application/vnd.github.raw",
	)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

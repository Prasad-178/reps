package ingest

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Prasad-178/reps/internal/llm"
	"github.com/Prasad-178/reps/internal/store"

	"github.com/google/uuid"
)

const (
	chunkMaxTokens = 700
	embedBatchSize = 64
)

func (p *Pipeline) Rebuild(ctx context.Context) error {
	dim := llm.EmbedDim(p.cfg.LLM.EmbedModel)
	if dim != 0 && dim != p.store.EmbedDim() {
		fmt.Printf("Embed model dim changed (%d → %d). Rebuilding vec table.\n", p.store.EmbedDim(), dim)
		if err := p.store.RebuildVecTable(dim); err != nil {
			return err
		}
	}

	sources, err := p.store.ListSources()
	if err != nil {
		return err
	}
	if len(sources) == 0 {
		return fmt.Errorf("no sources ingested. Run `reps add resume <path>` or `reps add github <user>` first.")
	}

	fmt.Printf("Re-embedding %d source(s)...\n", len(sources))
	for _, src := range sources {
		if err := p.store.DeleteChunksBySource(src.ID); err != nil {
			return fmt.Errorf("clear chunks for %s: %w", src.ID, err)
		}
		raw, err := os.ReadFile(src.RawPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", src.RawPath, err)
		}
		chunks := chunkBySemanticBoundaries(string(raw), chunkMaxTokens)
		fmt.Printf("  %s [%s] → %d chunks\n", src.Kind, src.Ref, len(chunks))
		if err := p.embedAndStore(ctx, src.ID, chunks); err != nil {
			return err
		}
	}

	fmt.Println("Synthesizing profile...")
	md, err := p.synthesizeProfile(ctx, sources)
	if err != nil {
		return err
	}
	if err := p.store.UpsertProfile(md, p.client.Model); err != nil {
		return err
	}
	fmt.Println("✓ profile written.")
	return nil
}

func (p *Pipeline) embedAndStore(ctx context.Context, sourceID string, chunks []string) error {
	for batchStart := 0; batchStart < len(chunks); batchStart += embedBatchSize {
		end := batchStart + embedBatchSize
		if end > len(chunks) {
			end = len(chunks)
		}
		batch := chunks[batchStart:end]
		vecs, err := p.client.Embed(ctx, batch)
		if err != nil {
			return fmt.Errorf("embed batch %d-%d: %w", batchStart, end, err)
		}
		if len(vecs) > 0 && len(vecs[0]) != p.store.EmbedDim() {
			newDim := len(vecs[0])
			fmt.Printf("Embed returned dim %d, expected %d. Rebuilding vec table.\n", newDim, p.store.EmbedDim())
			if err := p.store.RebuildVecTable(newDim); err != nil {
				return err
			}
		}
		for i, text := range batch {
			c := store.Chunk{
				ID:       uuid.NewString(),
				SourceID: sourceID,
				Ord:      batchStart + i,
				Text:     text,
			}
			if err := p.store.InsertChunk(c, vecs[i]); err != nil {
				return fmt.Errorf("insert chunk: %w", err)
			}
		}
	}
	return nil
}

func (p *Pipeline) synthesizeProfile(ctx context.Context, sources []store.Source) (string, error) {
	var sb strings.Builder
	for _, src := range sources {
		raw, err := os.ReadFile(src.RawPath)
		if err != nil {
			continue
		}
		body := string(raw)
		if len(body) > 12000 {
			body = body[:12000] + "\n...(truncated)..."
		}
		fmt.Fprintf(&sb, "## Source: %s (%s)\n%s\n\n", src.Kind, src.Ref, body)
	}

	req := llm.ChatRequest{
		Model: p.client.Model,
		Messages: []llm.Message{
			{Role: "system", Content: profileSystemPrompt},
			{Role: "user", Content: "Sources:\n\n" + sb.String()},
		},
		Temperature: 0.3,
		MaxTokens:   2000,
	}
	out, _, err := p.client.Chat(ctx, req)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

const profileSystemPrompt = `You are building a concise candidate profile that will be loaded into every
interview-rehearsal LLM prompt. The profile must be specific, technical, and
short (~1.5k tokens max). It will be used by an interviewer agent to ask
questions that reference the candidate's real work.

Output Markdown with these sections:

# Profile

## Who they are
One paragraph. Role, seniority, primary domains.

## Top projects (5)
For each: name, one-line crisp summary, key result/metric if any, tech stack.
Pull verbatim metrics (e.g. "99.8% R@10", "357ms p99") where the source provides them.

## Claimed skills
Tight bulleted list. No fluff. Include depth indicator if signal exists (e.g. "Rust — 3 production projects").

## Work history
Compact bullets. Company, role, dates, one-line scope.

## Target roles / interests
What kinds of roles they are aiming at, inferred from sources.

## Voice / tone
2-3 lines describing how they communicate (terse, exhaustive, formal, builder-tone, etc.) so the interviewer can match register.

Rules:
- Be specific, not generic. "Built a vector DB in Rust" is better than "worked on databases".
- Quote real numbers. Don't fabricate.
- Don't hedge. If a source claims something, state it.
- Cut anything you can't ground in the provided sources.
`

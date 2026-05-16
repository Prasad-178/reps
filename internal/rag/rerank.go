package rag

import (
	"context"
	"fmt"
	"strings"

	"github.com/Prasad-178/reps/internal/llm"
)

// Rerank takes the top-k vector hits and asks a cheap LLM to pick the best
// keep=3 for grounding the interviewer prompt. Returns the chosen subset in
// rank order. On error or empty result, falls back to the first `keep` of
// the input.
func (r *Retriever) Rerank(ctx context.Context, query string, hits []Chunk, keep int) []Chunk {
	if keep <= 0 {
		keep = 3
	}
	if len(hits) <= keep {
		return hits
	}
	var sb strings.Builder
	for i, h := range hits {
		body := h.Text
		if len(body) > 600 {
			body = body[:600] + "..."
		}
		fmt.Fprintf(&sb, "[%d] (%s/%s)\n%s\n\n", i, h.Kind, h.Ref, body)
	}
	model := r.Client.RerankModel
	if model == "" {
		model = r.Client.Model
	}
	prompt := fmt.Sprintf(
		"Query: %s\n\nCandidates:\n%s\nReturn ONLY a JSON object of the form {\"picks\":[i,j,k]} containing the %d most relevant candidate indices. No prose.",
		query, sb.String(), keep,
	)
	var resp struct {
		Picks []int `json:"picks"`
	}
	_, err := r.Client.ChatJSON(ctx, llm.ChatRequest{
		Model:       model,
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		Temperature: 0,
		MaxTokens:   60,
	}, &resp)
	picks := resp.Picks
	if err != nil || len(picks) == 0 {
		if keep > len(hits) {
			keep = len(hits)
		}
		return hits[:keep]
	}
	out := make([]Chunk, 0, keep)
	seen := map[int]bool{}
	for _, idx := range picks {
		if idx < 0 || idx >= len(hits) || seen[idx] {
			continue
		}
		seen[idx] = true
		out = append(out, hits[idx])
		if len(out) >= keep {
			break
		}
	}
	if len(out) == 0 {
		if keep > len(hits) {
			keep = len(hits)
		}
		return hits[:keep]
	}
	return out
}

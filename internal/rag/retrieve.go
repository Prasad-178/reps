package rag

import (
	"context"
	"fmt"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"

	"github.com/Prasad-178/reps/internal/llm"
	"github.com/Prasad-178/reps/internal/store"
)

type Chunk struct {
	ID       string
	SourceID string
	Kind     string
	Ref      string
	Text     string
	Distance float64
}

type Retriever struct {
	Store  *store.Store
	Client *llm.Client
}

func New(s *store.Store, c *llm.Client) *Retriever {
	return &Retriever{Store: s, Client: c}
}

// Retrieve embeds the query and returns top-k chunks by cosine distance.
func (r *Retriever) Retrieve(ctx context.Context, query string, k int) ([]Chunk, error) {
	if k <= 0 {
		k = 8
	}
	vecs, err := r.Client.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	if len(vecs) == 0 || len(vecs[0]) == 0 {
		return nil, fmt.Errorf("embed query returned empty vector")
	}
	if got, want := len(vecs[0]), r.Store.EmbedDim(); got != want {
		return nil, fmt.Errorf("query embed dim %d != schema dim %d (run `reps profile --rebuild`)", got, want)
	}
	blob, err := sqlite_vec.SerializeFloat32(vecs[0])
	if err != nil {
		return nil, err
	}
	rows, err := r.Store.DB.QueryContext(ctx, `
		SELECT v.chunk_id, c.source_id, s.kind, s.ref, c.text, v.distance
		FROM chunks_vec v
		JOIN chunks  c ON c.id = v.chunk_id
		JOIN sources s ON s.id = c.source_id
		WHERE v.embedding MATCH ?
		  AND v.k = ?
		ORDER BY v.distance
	`, blob, k)
	if err != nil {
		return nil, fmt.Errorf("vec query: %w", err)
	}
	defer rows.Close()
	var out []Chunk
	for rows.Next() {
		var ch Chunk
		if err := rows.Scan(&ch.ID, &ch.SourceID, &ch.Kind, &ch.Ref, &ch.Text, &ch.Distance); err != nil {
			return nil, err
		}
		out = append(out, ch)
	}
	return out, rows.Err()
}

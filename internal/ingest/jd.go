package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Prasad-178/reps/internal/llm"
	"github.com/Prasad-178/reps/internal/store"
	"github.com/google/uuid"
)

type JDCard struct {
	Company   string   `json:"company"`
	Role      string   `json:"role"`
	Location  string   `json:"location"`
	Level     string   `json:"level"`
	MustHaves []string `json:"must_haves"`
	NiceToHaves []string `json:"nice_to_haves"`
	Culture   []string `json:"culture"`
	Tech      []string `json:"tech"`
	Summary   string   `json:"summary"`
}

func (p *Pipeline) IngestJD(ctx context.Context, url string) (string, error) {
	text, err := ReadURL(ctx, url)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", url, err)
	}
	if len(text) < 200 {
		return "", fmt.Errorf("JD body too small (%d chars) — page may have blocked us", len(text))
	}

	id := uuid.NewString()
	rawPath := filepath.Join(p.cfg.Paths.Sources, id+".txt")
	if err := os.WriteFile(rawPath, []byte(text), 0o644); err != nil {
		return "", err
	}
	meta, _ := json.Marshal(map[string]any{"url": url, "chars": len(text)})
	if err := p.store.InsertSource(store.Source{
		ID: id, Kind: "jd", Ref: url, RawPath: rawPath,
		FetchedAt: time.Now(), MetaJSON: string(meta),
	}); err != nil {
		return "", err
	}

	if p.client == nil || p.client.APIKey == "" {
		fmt.Println("  (skipping JD card extraction — no API key)")
		return id, nil
	}

	body := text
	if len(body) > 16000 {
		body = body[:16000]
	}
	req := llm.ChatRequest{
		Model: p.client.Model,
		Messages: []llm.Message{
			{Role: "system", Content: jdCardSystemPrompt},
			{Role: "user", Content: body},
		},
		Temperature: 0.2,
		MaxTokens:   900,
	}
	var card JDCard
	if _, err := p.client.ChatJSON(ctx, req, &card); err != nil {
		fmt.Printf("  (JD card extraction failed: %v — raw JD still saved)\n", err)
		return id, nil
	}
	cardJSON, _ := json.Marshal(card)
	if err := p.store.InsertJDCard(store.JDCard{
		ID: uuid.NewString(), SourceID: id, Company: card.Company, Role: card.Role,
		CardJSON: string(cardJSON),
	}); err != nil {
		return id, fmt.Errorf("save jd_card: %w", err)
	}
	fmt.Printf("  parsed: %s — %s\n", card.Company, card.Role)
	return id, nil
}

const jdCardSystemPrompt = `You are extracting a structured job-description card from raw scraped text.
Return JSON only, matching this schema exactly:

{
  "company": string,
  "role": string,
  "location": string,
  "level": string,            // junior | mid | senior | staff | principal | unknown
  "must_haves": [string],     // hard requirements, terse
  "nice_to_haves": [string],
  "culture": [string],        // mission/values cues, work style
  "tech": [string],           // specific technologies named
  "summary": string           // 2-3 sentence summary of the role
}

Rules:
- Be terse. Each list item ≤ 12 words.
- Don't invent — if the JD doesn't say something, leave it empty.
- "level" is your best guess based on years/seniority cues.
`

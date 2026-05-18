package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ProbeKey makes a cheap chat call to verify the API key works for the given
// model. Returns nil on success, error on auth/quota/model issues. Times out
// after 12s.
func ProbeKey(ctx context.Context, apiKey, model string) error {
	if apiKey == "" {
		return fmt.Errorf("API key is empty")
	}
	if model == "" {
		model = "google/gemini-2.0-flash-001"
	}
	cctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	body, _ := json.Marshal(map[string]any{
		"model":      model,
		"max_tokens": 1,
		"messages": []map[string]string{
			{"role": "user", "content": "ping"},
		},
	})
	req, err := http.NewRequestWithContext(cctx, "POST", chatURL, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/Prasad-178/reps")
	req.Header.Set("X-Title", "reps")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	buf, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 200 {
		return nil
	}
	// Try to surface the OpenRouter error message verbatim.
	var e struct {
		Error struct {
			Message string `json:"message"`
			Code    any    `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(buf, &e)
	msg := strings.TrimSpace(e.Error.Message)
	switch resp.StatusCode {
	case 401:
		if msg == "" {
			msg = "unauthorized"
		}
		return fmt.Errorf("auth failed: %s", msg)
	case 402:
		return fmt.Errorf("payment required (out of credits): %s", msg)
	case 404:
		return fmt.Errorf("model not found: %s", model)
	case 429:
		return fmt.Errorf("rate-limited — try again")
	default:
		if msg == "" {
			msg = strings.TrimSpace(string(buf))
		}
		if len(msg) > 200 {
			msg = msg[:200] + "..."
		}
		return fmt.Errorf("openrouter http %d: %s", resp.StatusCode, msg)
	}
}

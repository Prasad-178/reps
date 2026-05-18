package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ChatStream emits text deltas as they arrive. `onToken` is called for each
// content delta; the final return value is the full concatenated text.
// Reads OpenAI-compatible SSE deltas from OpenRouter.
func (c *Client) ChatStream(
	ctx context.Context,
	req ChatRequest,
	onToken func(string),
) (string, ChatResponse, error) {
	if req.Model == "" {
		req.Model = c.Model
	}
	payload := map[string]any{
		"model":       req.Model,
		"messages":    req.Messages,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
		"stream":      true,
	}
	if req.ResponseFormat != nil {
		payload["response_format"] = req.ResponseFormat
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", ChatResponse{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", chatURL, bytes.NewReader(body))
	if err != nil {
		return "", ChatResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("HTTP-Referer", "https://github.com/Prasad-178/reps")
	httpReq.Header.Set("X-Title", "reps")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return "", ChatResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		buf, _ := io.ReadAll(resp.Body)
		return "", ChatResponse{}, fmt.Errorf("stream http %d: %s", resp.StatusCode, truncate(string(buf), 200))
	}

	var full strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		data = strings.TrimSpace(data)
		if data == "" || data == "[DONE]" {
			continue
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
					Role    string `json:"role"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Error *apiError `json:"error,omitempty"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if chunk.Error != nil {
			return full.String(), ChatResponse{}, chunk.Error
		}
		for _, ch := range chunk.Choices {
			if ch.Delta.Content == "" {
				continue
			}
			full.WriteString(ch.Delta.Content)
			if onToken != nil {
				onToken(ch.Delta.Content)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return full.String(), ChatResponse{}, err
	}
	return full.String(), ChatResponse{}, nil
}

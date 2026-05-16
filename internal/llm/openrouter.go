package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
)

const (
	chatURL  = "https://openrouter.ai/api/v1/chat/completions"
	embedURL = "https://openrouter.ai/api/v1/embeddings"
)

type Client struct {
	APIKey      string
	Model       string
	EmbedModel  string
	JudgeModel  string
	RerankModel string
	HTTP        *http.Client
}

func New(apiKey, model, embedModel, judgeModel, rerankModel string) *Client {
	return &Client{
		APIKey:      apiKey,
		Model:       model,
		EmbedModel:  embedModel,
		JudgeModel:  judgeModel,
		RerankModel: rerankModel,
		HTTP:        &http.Client{Timeout: 120 * time.Second},
	}
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model          string         `json:"model"`
	Messages       []Message      `json:"messages"`
	Temperature    float64        `json:"temperature,omitempty"`
	MaxTokens      int            `json:"max_tokens,omitempty"`
	ResponseFormat map[string]any `json:"response_format,omitempty"`
}

type ChatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message Message `json:"message"`
		Finish  string  `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *apiError `json:"error,omitempty"`
}

type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    any    `json:"code"`
}

func (e *apiError) Error() string {
	return fmt.Sprintf("openrouter: %s (type=%s code=%v)", e.Message, e.Type, e.Code)
}

func (c *Client) Chat(ctx context.Context, req ChatRequest) (string, ChatResponse, error) {
	if req.Model == "" {
		req.Model = c.Model
	}
	body, err := json.Marshal(req)
	if err != nil {
		return "", ChatResponse{}, err
	}
	var out ChatResponse
	err = c.doRetry(ctx, "POST", chatURL, body, &out)
	if err != nil {
		return "", out, err
	}
	if out.Error != nil {
		return "", out, out.Error
	}
	if len(out.Choices) == 0 {
		return "", out, errors.New("openrouter: empty choices")
	}
	return out.Choices[0].Message.Content, out, nil
}

func (c *Client) ChatJSON(ctx context.Context, req ChatRequest, target any) (ChatResponse, error) {
	if req.ResponseFormat == nil {
		req.ResponseFormat = map[string]any{"type": "json_object"}
	}
	content, resp, err := c.Chat(ctx, req)
	if err != nil {
		return resp, err
	}
	content = extractJSON(content)
	if err := json.Unmarshal([]byte(content), target); err != nil {
		return resp, fmt.Errorf("decode model JSON: %w (raw=%s)", err, truncate(content, 400))
	}
	return resp, nil
}

type EmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type EmbedResponse struct {
	Model string `json:"model"`
	Data  []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	Error *apiError `json:"error,omitempty"`
}

func (c *Client) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	req := EmbedRequest{Model: c.EmbedModel, Input: inputs}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	var out EmbedResponse
	if err := c.doRetry(ctx, "POST", embedURL, body, &out); err != nil {
		return nil, err
	}
	if out.Error != nil {
		return nil, out.Error
	}
	vecs := make([][]float32, len(out.Data))
	for _, d := range out.Data {
		if d.Index < 0 || d.Index >= len(vecs) {
			return nil, fmt.Errorf("embed: index %d out of range", d.Index)
		}
		vecs[d.Index] = d.Embedding
	}
	for i, v := range vecs {
		if v == nil {
			return nil, fmt.Errorf("embed: missing vector at %d", i)
		}
	}
	return vecs, nil
}

func (c *Client) doRetry(ctx context.Context, method, url string, body []byte, out any) error {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 500 * time.Millisecond
	bo.MaxInterval = 8 * time.Second
	bo.MaxElapsedTime = 60 * time.Second
	op := func() error {
		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
		if err != nil {
			return backoff.Permanent(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("HTTP-Referer", "https://github.com/Prasad-178/reps")
		req.Header.Set("X-Title", "reps")
		resp, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		buf, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if resp.StatusCode >= 500 || resp.StatusCode == 429 {
			return fmt.Errorf("openrouter http %d: %s", resp.StatusCode, truncate(string(buf), 200))
		}
		if resp.StatusCode >= 400 {
			return backoff.Permanent(fmt.Errorf("openrouter http %d: %s", resp.StatusCode, truncate(string(buf), 200)))
		}
		return json.Unmarshal(buf, out)
	}
	return backoff.Retry(op, backoff.WithContext(bo, ctx))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// extractJSON pulls the first JSON object/array from a response that may have
// markdown fences or leading prose. Cheap and good enough for v1.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```")
		if i := strings.LastIndex(s, "```"); i >= 0 {
			s = s[:i]
		}
		s = strings.TrimSpace(s)
	}
	start := strings.IndexAny(s, "{[")
	if start < 0 {
		return s
	}
	end := strings.LastIndexAny(s, "}]")
	if end < start {
		return s
	}
	return s[start : end+1]
}

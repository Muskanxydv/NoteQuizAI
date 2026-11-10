package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ErrBlocked means Gemini returned no usable text (safety block or empty answer).
// Retrying the same prompt will not help.
var ErrBlocked = errors.New("llm: response blocked or empty")

// APIError is a non-200 reply from the Gemini API.
type APIError struct {
	StatusCode int
	Message    string
	RetryAfter time.Duration // from the Retry-After header, if present
}

func (e *APIError) Error() string {
	return fmt.Sprintf("gemini API error (status %d): %s", e.StatusCode, e.Message)
}

// Retryable is true for rate limits (429) and server-side errors (5xx).
func (e *APIError) Retryable() bool {
	return e.StatusCode == http.StatusTooManyRequests || e.StatusCode >= 500
}

// Request / response shapes for POST /models/{model}:generateContent.
type part struct {
	Text    string `json:"text"`
	Thought bool   `json:"thought,omitempty"`
}

type content struct {
	Parts []part `json:"parts"`
}

type generationConfig struct {
	ResponseMimeType string  `json:"responseMimeType"`
	Temperature      float64 `json:"temperature"`
	MaxOutputTokens  int     `json:"maxOutputTokens,omitempty"`
}

type genRequest struct {
	Contents         []content        `json:"contents"`
	GenerationConfig generationConfig `json:"generationConfig"`
}

type genResponse struct {
	Candidates []struct {
		Content      content `json:"content"`
		FinishReason string  `json:"finishReason"`
	} `json:"candidates"`
	PromptFeedback struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
}

// geminiModel is one Gemini model with its own rate limiter.
type geminiModel struct {
	name    string
	apiKey  string
	baseURL string
	http    *http.Client
	timeout time.Duration
	limiter *Limiter
}

// call makes a single request (no retries, no limiter) and returns the JSON text.
func (m *geminiModel) call(ctx context.Context, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	body, err := json.Marshal(genRequest{
		Contents: []content{{Parts: []part{{Text: prompt}}}},
		GenerationConfig: generationConfig{
			ResponseMimeType: "application/json",
			Temperature:      0.4,
			MaxOutputTokens:  8192,
		},
	})
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/models/%s:generateContent", m.baseURL, m.name)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", m.apiKey) // header, so the key never appears in URLs or logs

	resp, err := m.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", fmt.Errorf("reading gemini response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", newAPIError(resp, raw)
	}

	var out genResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("decoding gemini response: %w", err)
	}
	if reason := out.PromptFeedback.BlockReason; reason != "" {
		return "", fmt.Errorf("%w: prompt blocked (%s)", ErrBlocked, reason)
	}
	if len(out.Candidates) == 0 {
		return "", fmt.Errorf("%w: no candidates returned", ErrBlocked)
	}

	cand := out.Candidates[0]
	var sb strings.Builder
	for _, p := range cand.Content.Parts {
		if !p.Thought {
			sb.WriteString(p.Text)
		}
	}
	text := strings.TrimSpace(sb.String())
	if text == "" {
		return "", fmt.Errorf("%w: empty text (finish reason %q)", ErrBlocked, cand.FinishReason)
	}
	return text, nil
}

func newAPIError(resp *http.Response, raw []byte) *APIError {
	msg := strings.TrimSpace(string(raw))
	var env struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &env) == nil && env.Error.Message != "" {
		msg = env.Error.Message
	}
	if len(msg) > 300 {
		msg = msg[:300] + "..."
	}
	apiErr := &APIError{StatusCode: resp.StatusCode, Message: msg}
	if secs, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && secs > 0 {
		apiErr.RetryAfter = time.Duration(secs) * time.Second
	}
	return apiErr
}

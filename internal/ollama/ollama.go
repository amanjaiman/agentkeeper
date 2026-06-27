// Package ollama is a minimal client for a local Ollama server's /api/generate
// endpoint. It is provider-pluggable behind a tiny interface so LM Studio or
// other local backends can slot in later. Everything stays on the machine.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// DefaultBaseURL is used when neither config nor OLLAMA_HOST specifies one.
const DefaultBaseURL = "http://localhost:11434"

// Client talks to an Ollama server.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// New returns a Client. baseURL falls back to $OLLAMA_HOST, then DefaultBaseURL.
func New(baseURL string) *Client {
	if baseURL == "" {
		baseURL = os.Getenv("OLLAMA_HOST")
	}
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if !strings.Contains(baseURL, "://") {
		baseURL = "http://" + baseURL
	}
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Timeout: 90 * time.Second},
	}
}

type genRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type genResponse struct {
	Response string `json:"response"`
	Error    string `json:"error"`
}

// Generate sends a single non-streaming completion request and returns the
// model's response text.
func (c *Client) Generate(ctx context.Context, model, prompt string) (string, error) {
	body, err := json.Marshal(genRequest{Model: model, Prompt: prompt, Stream: false})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama unreachable at %s: %w", c.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("ollama returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var gr genResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return "", fmt.Errorf("decode ollama response: %w", err)
	}
	if gr.Error != "" {
		return "", fmt.Errorf("ollama error: %s", gr.Error)
	}
	return gr.Response, nil
}

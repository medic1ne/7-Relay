package router

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

// SSEProxy handles streaming responses between providers and clients
type SSEProxy struct{}

func NewSSEProxy() *SSEProxy {
	return &SSEProxy{}
}

// StreamRequest holds state for a streaming request
type StreamRequest struct {
	Request    ChatRequest
	Provider   *ProviderPool
	ModelName  string
	ClaudeMode bool
	GeminiMode bool
}

// ProxyStream forwards an OpenAI-style streaming request to the provider
// and relays SSE chunks back to the client
func (sp *SSEProxy) ProxyStream(ctx context.Context, w http.ResponseWriter, sr StreamRequest) error {
	// Build the upstream request body
	body, err := json.Marshal(sr.Request)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	url := fmt.Sprintf("%s/chat/completions", sr.Provider.config.BaseURL)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request error: %w", err)
	}

	sr.Provider.setHeaders(httpReq)

	// Execute
	resp, err := sr.Provider.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("provider request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("provider returned %d: %s", resp.StatusCode, string(respBody))
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("response writer doesn't support flushing")
	}

	// Read SSE stream and relay to client
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	var lastChunk []byte

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// Empty line = end of SSE event
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			if data == "[DONE]" {
				fmt.Fprintf(w, "data: [DONE]\n\n")
				flusher.Flush()
				break
			}

			// Parse and relay the chunk
			lastChunk = []byte(data)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}

	// Update token usage from final chunk
	if lastChunk != nil {
		sp.extractUsage(sr, lastChunk)
	}

	return nil
}

// extractUsage parses the final SSE chunk for usage stats
func (sp *SSEProxy) extractUsage(sr StreamRequest, chunk []byte) {
	var parsed struct {
		Usage *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(chunk, &parsed); err == nil && parsed.Usage != nil {
		// Usage stats available - caller can track these
		_ = parsed.Usage
	}
}

package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type ProviderPool struct {
	config    ProviderConfig
	client    *http.Client
	mu        sync.RWMutex
	keys      []string // multi-account support
	currentKey int
}

func NewProviderPool(cfg ProviderConfig) *ProviderPool {
	return &ProviderPool{
		config: cfg,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (p *ProviderPool) Forward(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Marshal request
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build URL
	url := fmt.Sprintf("%s/chat/completions", p.config.BaseURL)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers based on provider type
	p.setHeaders(httpReq)

	// Send request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("provider request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("provider returned %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &chatResp, nil
}

func (p *ProviderPool) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")

	switch p.config.Type {
	case "openai", "custom":
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))
	case "claude":
		req.Header.Set("x-api-key", p.config.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	case "gemini":
		// Gemini uses API key in URL query param
		q := req.URL.Query()
		q.Set("key", p.config.APIKey)
		req.URL.RawQuery = q.Encode()
	}

	// Apply custom headers
	for k, v := range p.config.Headers {
		req.Header.Set(k, v)
	}
}

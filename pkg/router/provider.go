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
	config       ProviderConfig
	client       *http.Client
	translator   *FormatTranslator
	circuit      *CircuitBreaker
	rateLimiter  *RateLimiter
	mu           sync.RWMutex
	keys         []string // multi-account support
	currentKey   int
	lastHealth   time.Time
	healthy      bool
}

func NewProviderPool(cfg ProviderConfig) *ProviderPool {
	// Default rate limits based on tier
	maxTokens := 60.0
	refillRate := 1.0 // 1 request/second
	switch cfg.Tier {
	case 1:
		maxTokens = 100
		refillRate = 2.0
	case 2:
		maxTokens = 60
		refillRate = 1.0
	case 3:
		maxTokens = 20
		refillRate = 0.5
	}

	return &ProviderPool{
		config:      cfg,
		client:      &http.Client{Timeout: 120 * time.Second},
		translator:  NewFormatTranslator(),
		circuit:     NewCircuitBreaker(5, 60*time.Second), // 5 failures → 60s cooldown
		rateLimiter: NewRateLimiter(maxTokens, refillRate),
		healthy:     true,
	}
}

func (p *ProviderPool) Forward(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Check circuit breaker
	if !p.circuit.Allow() {
		return nil, fmt.Errorf("circuit breaker open for provider %s", p.config.Name)
	}

	// Check rate limiter
	if !p.rateLimiter.Allow() {
		return nil, fmt.Errorf("rate limit exceeded for provider %s", p.config.Name)
	}

	// Marshal request
	body, err := json.Marshal(req)
	if err != nil {
		p.circuit.RecordFailure()
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build URL
	url := fmt.Sprintf("%s/chat/completions", p.config.BaseURL)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		p.circuit.RecordFailure()
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	p.setHeaders(httpReq)

	// Retry logic with exponential backoff
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(1<<uint(attempt)) * time.Second)
		}

		resp, err := p.client.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("provider request failed: %w", err)
			p.circuit.RecordFailure()
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			p.circuit.RecordSuccess()
			var chatResp ChatResponse
			if err := json.Unmarshal(respBody, &chatResp); err != nil {
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			return &chatResp, nil
		}

		// Rate limited — back off
		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("rate limited by provider %s", p.config.Name)
			p.circuit.RecordFailure()
			continue
		}

		// Other error
		lastErr = fmt.Errorf("provider returned %d: %s", resp.StatusCode, string(respBody))
		p.circuit.RecordFailure()
	}

	return nil, lastErr
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
		q := req.URL.Query()
		q.Set("key", p.config.APIKey)
		req.URL.RawQuery = q.Encode()
	}

	for k, v := range p.config.Headers {
		req.Header.Set(k, v)
	}
}

// HealthCheck pings the provider's models endpoint
func (p *ProviderPool) HealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/models", p.config.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		p.mu.Lock()
		p.healthy = false
		p.mu.Unlock()
		return
	}

	p.setHeaders(httpReq)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		p.mu.Lock()
		p.healthy = false
		p.lastHealth = time.Now()
		p.mu.Unlock()
		return
	}
	resp.Body.Close()

	p.mu.Lock()
	p.healthy = resp.StatusCode == http.StatusOK
	p.lastHealth = time.Now()
	p.mu.Unlock()
}

func (p *ProviderPool) IsHealthy() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.healthy && p.circuit.State() == CircuitClosed
}

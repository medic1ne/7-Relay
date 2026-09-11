package router

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Router struct {
	cfg        *Config
	server     *http.Server
	providers  map[string]*ProviderPool
	mu         sync.RWMutex
	stats      *Stats
	comboMap   map[string][]string
	translator *FormatTranslator
	sseProxy   *SSEProxy
	health     *HealthChecker
}

type Stats struct {
	Requests   int64            `json:"requests"`
	TokensUsed int64            `json:"tokens_used"`
	Costs      float64          `json:"costs"`
	Errors     int64            `json:"errors"`
	ByProvider map[string]int64 `json:"by_provider"`
	mu         sync.RWMutex
}

type ChatRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	Stream    bool      `json:"stream,omitempty"`
	Tools     []Tool    `json:"tools,omitempty"`
	MaxTokens *int      `json:"max_tokens,omitempty"`
}

type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func New(cfg *Config) (*Router, error) {
	r := &Router{
		cfg:        cfg,
		providers:  make(map[string]*ProviderPool),
		comboMap:   make(map[string][]string),
		translator: NewFormatTranslator(),
		sseProxy:   NewSSEProxy(),
		stats: &Stats{
			ByProvider: make(map[string]int64),
		},
	}

	// Initialize providers
	for _, pc := range cfg.Providers {
		if pc.Enabled {
			r.providers[pc.Name] = NewProviderPool(pc)
		}
	}

	// Build combo map
	for _, combo := range cfg.Combos {
		r.comboMap[combo.Name] = combo.Models
	}

	// Start health checker
	r.health = NewHealthChecker(30 * time.Second)
	r.health.Start(r.providers)

	// HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", r.handleChatCompletions)
	mux.HandleFunc("/v1/models", r.handleModels)
	mux.HandleFunc("/v1/embeddings", r.handleEmbeddings)
	mux.HandleFunc("/health", r.handleHealth)
	mux.HandleFunc("/v1/stats", r.handleStats)
	mux.HandleFunc("/", r.handleRoot)

	r.server = &http.Server{
		Handler:      r.loggingMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 5 * time.Minute,
	}

	return r, nil
}

func (r *Router) ListenAndServe(addr string) error {
	r.server.Addr = addr
	return r.server.ListenAndServe()
}

func (r *Router) Shutdown() {
	r.health.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	r.server.Shutdown(ctx)
}

func (r *Router) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, req)
		if r.cfg.Logging.Enabled {
			log.Printf("[%s] %s %s %v", req.Method, req.URL.Path, req.RemoteAddr, time.Since(start))
		}
	})
}

func (r *Router) handleHealth(w http.ResponseWriter, req *http.Request) {
	providerStatus := make(map[string]string)
	r.mu.RLock()
	for name, pool := range r.providers {
		if pool.IsHealthy() {
			providerStatus[name] = "healthy"
		} else {
			providerStatus[name] = "unhealthy"
		}
	}
	r.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"version":   "0.1.0",
		"providers": providerStatus,
	})
}

func (r *Router) handleStats(w http.ResponseWriter, req *http.Request) {
	r.stats.mu.RLock()
	defer r.stats.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"requests":    r.stats.Requests,
		"tokens_used": r.stats.TokensUsed,
		"errors":      r.stats.Errors,
		"by_provider": r.stats.ByProvider,
	})
}

func (r *Router) handleRoot(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":    "7-Relay",
		"version": "0.1.0",
		"endpoints": []string{
			"POST /v1/chat/completions",
			"GET  /v1/models",
			"POST /v1/embeddings",
			"GET  /health",
			"GET  /v1/stats",
		},
	})
}

func (r *Router) handleModels(w http.ResponseWriter, req *http.Request) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var models []map[string]string
	for name, pool := range r.providers {
		for _, m := range pool.config.Models {
			models = append(models, map[string]string{
				"id":       fmt.Sprintf("%s/%s", name, m),
				"object":   "model",
				"owned_by": name,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"object": "list",
		"data":   models,
	})
}

func (r *Router) handleChatCompletions(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var chatReq ChatRequest
	if err := json.Unmarshal(body, &chatReq); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Track request
	r.stats.mu.Lock()
	r.stats.Requests++
	r.stats.mu.Unlock()

	// Resolve provider and model
	provider, modelName := r.resolveProvider(chatReq.Model)
	if provider == nil {
		// Try combo fallback
		provider, modelName = r.resolveCombo(chatReq.Model)
	}
	if provider == nil {
		http.Error(w, fmt.Sprintf("No provider available for model: %s", chatReq.Model), http.StatusBadGateway)
		return
	}

	// Rewrite model name
	chatReq.Model = modelName

	// Handle streaming
	if chatReq.Stream {
		sr := StreamRequest{
			Request:   chatReq,
			Provider:  provider,
			ModelName: modelName,
		}
		if err := r.sseProxy.ProxyStream(req.Context(), w, sr); err != nil {
			http.Error(w, fmt.Sprintf("Stream error: %v", err), http.StatusBadGateway)
		}
		return
	}

	// Non-streaming with fallback
	resp, err := provider.Forward(req.Context(), chatReq)
	if err != nil {
		r.stats.mu.Lock()
		r.stats.Errors++
		r.stats.mu.Unlock()

		// Fallback to next tier
		if fallback := r.findFallback(provider); fallback != nil {
			log.Printf("[fallback] %s failed, trying %s", provider.config.Name, fallback.config.Name)
			resp, err = fallback.Forward(req.Context(), chatReq)
			if err != nil {
				http.Error(w, fmt.Sprintf("All providers failed: %v", err), http.StatusBadGateway)
				return
			}
			provider = fallback
		} else {
			http.Error(w, fmt.Sprintf("Provider error: %v", err), http.StatusBadGateway)
			return
		}
	}

	// Update stats
	r.stats.mu.Lock()
	r.stats.TokensUsed += int64(resp.Usage.TotalTokens)
	r.stats.ByProvider[provider.config.Name]++
	r.stats.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (r *Router) handleEmbeddings(w http.ResponseWriter, req *http.Request) {
	// Forward to OpenAI-compatible embedding endpoint
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	provider, _ := r.resolveProvider("openai/text-embedding-3-small")
	if provider == nil {
		// Find any provider
		r.mu.RLock()
		for _, p := range r.providers {
			provider = p
			break
		}
		r.mu.RUnlock()
	}
	if provider == nil {
		http.Error(w, "No embedding provider available", http.StatusBadGateway)
		return
	}

	url := fmt.Sprintf("%s/embeddings", provider.config.BaseURL)
	httpReq, err := http.NewRequest("POST", url, strings.NewReader(string(body)))
	if err != nil {
		http.Error(w, "Request creation failed", http.StatusInternalServerError)
		return
	}
	provider.setHeaders(httpReq)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := provider.client.Do(httpReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Provider error: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (r *Router) resolveProvider(model string) (*ProviderPool, string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check explicit provider prefix (e.g., "openai/gpt-4o")
	if idx := strings.Index(model, "/"); idx > 0 {
		providerName := model[:idx]
		modelName := model[idx+1:]
		if pool, ok := r.providers[providerName]; ok {
			return pool, modelName
		}
	}

	// Match by model name
	for _, pool := range r.providers {
		for _, m := range pool.config.Models {
			if m == model {
				return pool, model
			}
		}
	}

	return nil, ""
}

func (r *Router) resolveCombo(model string) (*ProviderPool, string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check if model matches a combo name
	comboModels, ok := r.comboMap[model]
	if !ok {
		return nil, ""
	}

	// Try each model in combo order
	for _, cm := range comboModels {
		for _, pool := range r.providers {
			if pool.IsHealthy() {
				for _, m := range pool.config.Models {
					if m == cm {
						return pool, m
					}
				}
			}
		}
	}

	return nil, ""
}

func (r *Router) findFallback(current *ProviderPool) *ProviderPool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Find provider with higher tier number (cheaper/free)
	for _, pool := range r.providers {
		if pool.config.Name != current.config.Name && pool.IsHealthy() {
			if pool.config.Tier > current.config.Tier {
				return pool
			}
		}
	}

	// If current is already highest tier, try any healthy provider
	for _, pool := range r.providers {
		if pool.config.Name != current.config.Name && pool.IsHealthy() {
			return pool
		}
	}

	return nil
}

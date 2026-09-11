package router

import (
	"sync"
	"time"
)

// ==================== Circuit Breaker ====================

type CircuitState int

const (
	CircuitClosed   CircuitState = iota // Normal — requests flow through
	CircuitOpen                         // Tripped — requests blocked
	CircuitHalfOpen                     // Testing — allow one probe
)

type CircuitBreaker struct {
	state        CircuitState
	failures     int
	successes    int
	lastFail     time.Time
	threshold    int
	resetTimeout time.Duration
	halfOpenMax  int
	mu           sync.RWMutex
}

func NewCircuitBreaker(threshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:        CircuitClosed,
		threshold:    threshold,
		resetTimeout: resetTimeout,
		halfOpenMax:  1,
	}
}

func (cb *CircuitBreaker) Allow() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if reset timeout has elapsed
		if time.Since(cb.lastFail) > cb.resetTimeout {
			return true // Allow probe
		}
		return false
	case CircuitHalfOpen:
		return cb.successes < cb.halfOpenMax
	}
	return false
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		cb.failures = 0
	case CircuitHalfOpen:
		cb.successes++
		if cb.successes >= cb.halfOpenMax {
			cb.state = CircuitClosed
			cb.failures = 0
			cb.successes = 0
		}
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFail = time.Now()

	switch cb.state {
	case CircuitClosed:
		if cb.failures >= cb.threshold {
			cb.state = CircuitOpen
		}
	case CircuitHalfOpen:
		cb.state = CircuitOpen
		cb.successes = 0
	}
}

func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// ==================== Rate Limiter (Token Bucket) ====================

type RateLimiter struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

func NewRateLimiter(maxTokens float64, refillRate float64) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refill()
	if rl.tokens >= 1 {
		rl.tokens--
		return true
	}
	return false
}

func (rl *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	rl.tokens += elapsed * rl.refillRate
	if rl.tokens > rl.maxTokens {
		rl.tokens = rl.maxTokens
	}
	rl.lastRefill = now
}

// ==================== Health Checker ====================

type HealthChecker struct {
	interval time.Duration
	stop     chan struct{}
}

func NewHealthChecker(interval time.Duration) *HealthChecker {
	return &HealthChecker{
		interval: interval,
		stop:     make(chan struct{}),
	}
}

func (hc *HealthChecker) Start(pools map[string]*ProviderPool) {
	go func() {
		ticker := time.NewTicker(hc.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				for _, pool := range pools {
					go pool.HealthCheck()
				}
			case <-hc.stop:
				return
			}
		}
	}()
}

func (hc *HealthChecker) Stop() {
	close(hc.stop)
}

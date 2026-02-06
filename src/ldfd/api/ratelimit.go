package api

import (
	"sync"
	"time"
)

// RateLimitConfig holds configuration for the rate limiter.
type RateLimitConfig struct {
	// Enabled controls whether rate limiting is active.
	Enabled bool
	// AuthRequestsPerMin is the max requests per minute for auth endpoints (login/create/refresh).
	AuthRequestsPerMin int
	// APIRequestsPerMin is the max requests per minute for general API endpoints.
	APIRequestsPerMin int
	// TrustProxy enables trusting X-Forwarded-For for client IP detection.
	TrustProxy bool
}

// DefaultRateLimitConfig returns sensible defaults for rate limiting.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Enabled:            true,
		AuthRequestsPerMin: 10,
		APIRequestsPerMin:  120,
		TrustProxy:         false,
	}
}

// window tracks request count within a time window.
type window struct {
	count     int
	expiresAt time.Time
}

// RateLimiter implements a sliding-window rate limiter keyed by arbitrary string.
type RateLimiter struct {
	mu      sync.Mutex
	windows map[string]*window
	config  RateLimitConfig
	stopCh  chan struct{}
}

// NewRateLimiter creates a new rate limiter and starts the background cleanup goroutine.
func NewRateLimiter(cfg RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		windows: make(map[string]*window),
		config:  cfg,
		stopCh:  make(chan struct{}),
	}
	go rl.cleanup()
	return rl
}

// Allow checks whether a request for the given key should be allowed under the specified limit.
// Returns true if allowed (and increments the counter), false if rate limit exceeded.
func (rl *RateLimiter) Allow(key string, limit int) bool {
	if !rl.config.Enabled || limit <= 0 {
		return true
	}

	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()

	w, exists := rl.windows[key]
	if !exists || now.After(w.expiresAt) {
		// Start a new 1-minute window
		rl.windows[key] = &window{
			count:     1,
			expiresAt: now.Add(time.Minute),
		}
		return true
	}

	if w.count >= limit {
		return false
	}

	w.count++
	return true
}

// cleanup periodically removes expired windows to prevent memory leaks.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			rl.mu.Lock()
			for key, w := range rl.windows {
				if now.After(w.expiresAt) {
					delete(rl.windows, key)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopCh:
			return
		}
	}
}

// Stop terminates the background cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.stopCh)
}

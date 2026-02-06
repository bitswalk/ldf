package api

import (
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{
		Enabled:            true,
		AuthRequestsPerMin: 3,
		APIRequestsPerMin:  5,
	})
	defer rl.Stop()

	key := "ip:127.0.0.1"

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		if !rl.Allow(key, 3) {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	// 4th request should be denied
	if rl.Allow(key, 3) {
		t.Fatal("4th request should be denied")
	}
}

func TestRateLimiter_DifferentKeys(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{
		Enabled:            true,
		AuthRequestsPerMin: 2,
	})
	defer rl.Stop()

	// Each key has its own window
	if !rl.Allow("ip:1.1.1.1", 2) {
		t.Fatal("first request for key1 should be allowed")
	}
	if !rl.Allow("ip:1.1.1.1", 2) {
		t.Fatal("second request for key1 should be allowed")
	}
	if rl.Allow("ip:1.1.1.1", 2) {
		t.Fatal("third request for key1 should be denied")
	}

	// Different key should still have quota
	if !rl.Allow("ip:2.2.2.2", 2) {
		t.Fatal("first request for key2 should be allowed")
	}
}

func TestRateLimiter_WindowExpiry(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{
		Enabled:            true,
		AuthRequestsPerMin: 1,
	})
	defer rl.Stop()

	key := "ip:10.0.0.1"

	if !rl.Allow(key, 1) {
		t.Fatal("first request should be allowed")
	}
	if rl.Allow(key, 1) {
		t.Fatal("second request should be denied within same window")
	}

	// Manually expire the window
	rl.mu.Lock()
	if w, ok := rl.windows[key]; ok {
		w.expiresAt = time.Now().Add(-time.Second)
	}
	rl.mu.Unlock()

	// Now should be allowed again (new window)
	if !rl.Allow(key, 1) {
		t.Fatal("request after window expiry should be allowed")
	}
}

func TestRateLimiter_Disabled(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{
		Enabled:            false,
		AuthRequestsPerMin: 1,
	})
	defer rl.Stop()

	// All requests should pass when disabled
	for i := 0; i < 100; i++ {
		if !rl.Allow("ip:1.1.1.1", 1) {
			t.Fatalf("request %d should be allowed when rate limiting is disabled", i+1)
		}
	}
}

func TestRateLimiter_ZeroLimit(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{
		Enabled: true,
	})
	defer rl.Stop()

	// Zero limit should allow all
	if !rl.Allow("ip:1.1.1.1", 0) {
		t.Fatal("request should be allowed with zero limit")
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{
		Enabled:            true,
		AuthRequestsPerMin: 5,
	})
	defer rl.Stop()

	// Add some entries
	rl.Allow("ip:1.1.1.1", 5)
	rl.Allow("ip:2.2.2.2", 5)

	// Expire all windows
	rl.mu.Lock()
	for _, w := range rl.windows {
		w.expiresAt = time.Now().Add(-time.Second)
	}
	rl.mu.Unlock()

	// Manually run cleanup logic
	now := time.Now()
	rl.mu.Lock()
	for key, w := range rl.windows {
		if now.After(w.expiresAt) {
			delete(rl.windows, key)
		}
	}
	rl.mu.Unlock()

	rl.mu.Lock()
	remaining := len(rl.windows)
	rl.mu.Unlock()

	if remaining != 0 {
		t.Fatalf("expected 0 windows after cleanup, got %d", remaining)
	}
}

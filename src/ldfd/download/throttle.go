package download

import (
	"context"
	"io"
	"sync"
	"time"
)

// ThrottleConfig holds bandwidth throttling configuration
type ThrottleConfig struct {
	PerWorkerBytesPerSec int64 // Per-worker limit (0 = unlimited)
	GlobalBytesPerSec    int64 // Global limit across all workers (0 = unlimited)
}

// DefaultThrottleConfig returns default throttle configuration (unlimited)
func DefaultThrottleConfig() ThrottleConfig {
	return ThrottleConfig{}
}

// rateLimiter implements a token bucket rate limiter
type rateLimiter struct {
	bytesPerSec int64
	tokens      int64
	maxTokens   int64
	lastRefill  time.Time
	mu          sync.Mutex
}

// newRateLimiter creates a new rate limiter with the given bytes-per-second limit
func newRateLimiter(bytesPerSec int64) *rateLimiter {
	// Max burst: 1 second worth of tokens (or 64KB minimum)
	maxTokens := bytesPerSec
	if maxTokens < 65536 {
		maxTokens = 65536
	}
	return &rateLimiter{
		bytesPerSec: bytesPerSec,
		tokens:      maxTokens,
		maxTokens:   maxTokens,
		lastRefill:  time.Now(),
	}
}

// Wait blocks until n bytes worth of tokens are available.
// Returns immediately if the limiter is unlimited (bytesPerSec <= 0).
func (rl *rateLimiter) Wait(ctx context.Context, n int) error {
	if rl.bytesPerSec <= 0 {
		return nil
	}

	for {
		rl.mu.Lock()
		// Refill tokens based on elapsed time
		now := time.Now()
		elapsed := now.Sub(rl.lastRefill)
		newTokens := int64(elapsed.Seconds() * float64(rl.bytesPerSec))
		if newTokens > 0 {
			rl.tokens += newTokens
			if rl.tokens > rl.maxTokens {
				rl.tokens = rl.maxTokens
			}
			rl.lastRefill = now
		}

		// Check if we have enough tokens
		needed := int64(n)
		if rl.tokens >= needed {
			rl.tokens -= needed
			rl.mu.Unlock()
			return nil
		}

		// Calculate how long to wait for enough tokens
		deficit := needed - rl.tokens
		waitDuration := time.Duration(float64(deficit) / float64(rl.bytesPerSec) * float64(time.Second))
		rl.mu.Unlock()

		// Wait with context awareness
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitDuration):
		}
	}
}

// throttledReader wraps an io.Reader with rate limiting
type throttledReader struct {
	reader   io.Reader
	limiters []*rateLimiter // multiple limiters (per-worker + global)
	ctx      context.Context
}

// newThrottledReader creates a reader that respects all provided rate limiters.
// Pass nil limiters to skip them (they are filtered out).
func newThrottledReader(ctx context.Context, r io.Reader, limiters ...*rateLimiter) *throttledReader {
	var active []*rateLimiter
	for _, l := range limiters {
		if l != nil {
			active = append(active, l)
		}
	}
	return &throttledReader{
		reader:   r,
		limiters: active,
		ctx:      ctx,
	}
}

// Read reads from the underlying reader, throttled by all configured rate limiters
func (tr *throttledReader) Read(p []byte) (int, error) {
	// If no limiters, read directly
	if len(tr.limiters) == 0 {
		return tr.reader.Read(p)
	}

	// Limit read size to avoid holding tokens for too long (max 32KB per read)
	maxRead := len(p)
	if maxRead > 32768 {
		maxRead = 32768
	}

	n, err := tr.reader.Read(p[:maxRead])
	if n > 0 {
		// Wait for tokens from all limiters
		for _, limiter := range tr.limiters {
			if waitErr := limiter.Wait(tr.ctx, n); waitErr != nil {
				return n, waitErr
			}
		}
	}
	return n, err
}

package download

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
)

func TestRateLimiter_UnlimitedReturnsImmediately(t *testing.T) {
	rl := newRateLimiter(0)
	start := time.Now()
	if err := rl.Wait(context.Background(), 1024*1024); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed > 10*time.Millisecond {
		t.Errorf("unlimited limiter should return immediately, took %v", elapsed)
	}
}

func TestRateLimiter_ContextCancellation(t *testing.T) {
	rl := newRateLimiter(100) // 100 bytes/sec
	// Drain all tokens (maxTokens is max(bytesPerSec, 65536) = 65536 for 100 B/s)
	_ = rl.Wait(context.Background(), int(rl.maxTokens))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Request more tokens than can refill in 50ms (100 B/s * 0.05s = 5 bytes)
	err := rl.Wait(ctx, 1000)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestRateLimiter_TokenRefill(t *testing.T) {
	rl := newRateLimiter(10000) // 10KB/sec
	// Drain tokens
	_ = rl.Wait(context.Background(), int(rl.maxTokens))

	// Wait a bit for refill
	time.Sleep(200 * time.Millisecond)

	start := time.Now()
	err := rl.Wait(context.Background(), 1000) // Ask for 1KB after refill
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should be fast since tokens have refilled (200ms * 10KB/s = ~2KB tokens)
	if elapsed > 100*time.Millisecond {
		t.Errorf("expected fast return after refill, took %v", elapsed)
	}
}

func TestThrottledReader_NoLimiters(t *testing.T) {
	data := bytes.Repeat([]byte("x"), 1024)
	reader := bytes.NewReader(data)
	tr := newThrottledReader(context.Background(), reader)

	out, err := io.ReadAll(tr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1024 {
		t.Errorf("expected 1024 bytes, got %d", len(out))
	}
}

func TestThrottledReader_WithLimiter(t *testing.T) {
	// 50KB/sec limiter, read 10KB — should complete quickly since burst allows it
	data := bytes.Repeat([]byte("a"), 10*1024)
	reader := bytes.NewReader(data)

	limiter := newRateLimiter(50 * 1024) // 50KB/s
	tr := newThrottledReader(context.Background(), reader, limiter)

	out, err := io.ReadAll(tr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 10*1024 {
		t.Errorf("expected %d bytes, got %d", 10*1024, len(out))
	}
}

func TestThrottledReader_NilLimitersFiltered(t *testing.T) {
	data := []byte("hello")
	reader := bytes.NewReader(data)
	tr := newThrottledReader(context.Background(), reader, nil, nil)

	if len(tr.limiters) != 0 {
		t.Errorf("expected 0 active limiters, got %d", len(tr.limiters))
	}

	out, err := io.ReadAll(tr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "hello" {
		t.Errorf("expected 'hello', got %q", string(out))
	}
}

func TestThrottledReader_MultipleLimiters(t *testing.T) {
	data := bytes.Repeat([]byte("b"), 5*1024)
	reader := bytes.NewReader(data)

	limiter1 := newRateLimiter(100 * 1024) // 100KB/s
	limiter2 := newRateLimiter(200 * 1024) // 200KB/s
	tr := newThrottledReader(context.Background(), reader, limiter1, limiter2)

	if len(tr.limiters) != 2 {
		t.Errorf("expected 2 active limiters, got %d", len(tr.limiters))
	}

	out, err := io.ReadAll(tr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 5*1024 {
		t.Errorf("expected %d bytes, got %d", 5*1024, len(out))
	}
}

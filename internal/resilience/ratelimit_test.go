package resilience

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestRateLimiter_AllowsWithinRate(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{Rate: 100, Burst: 10})

	// Should allow up to burst without blocking.
	for range 10 {
		if !rl.Allow() {
			t.Fatal("should allow within burst")
		}
	}
}

func TestRateLimiter_ThrottlesBeyondBurst(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{Rate: 1, Burst: 1})

	// First should pass.
	if !rl.Allow() {
		t.Fatal("first should be allowed")
	}

	// Second should be rejected (1 RPS, burst 1).
	if rl.Allow() {
		t.Fatal("second should be throttled")
	}
}

func TestRateLimiter_WaitBlocks(t *testing.T) {
	var throttled atomic.Int32
	rl := NewRateLimiter(RateLimitConfig{
		Rate:    1000, // high rate for fast test
		Burst:   1,
		OnLimit: func() { throttled.Add(1) },
	})

	// Exhaust burst.
	rl.Allow()

	// Wait should eventually succeed.
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := rl.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
}

func TestRateLimiter_WaitRespectsContext(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{Rate: 0.001, Burst: 1}) // very slow

	// Exhaust burst.
	rl.Allow()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := rl.Wait(ctx)
	if err == nil {
		t.Fatal("should timeout")
	}
}

func TestRateLimiter_WaitsCounter(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{Rate: 1000, Burst: 1})

	if rl.Waits() != 0 {
		t.Fatalf("initial waits = %d", rl.Waits())
	}

	rl.Allow() // exhaust burst

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	rl.Wait(ctx)

	if rl.Waits() != 1 {
		t.Fatalf("waits = %d, want 1", rl.Waits())
	}
}

func TestRateLimiter_DefaultConfig(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{})
	// Should not panic, defaults applied.
	if !rl.Allow() {
		t.Fatal("default config should allow first call")
	}
}

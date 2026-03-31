package guard

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	bd "github.com/dpopsuev/jericho/dispatch"
)

type countingDispatcher struct {
	calls atomic.Int64
}

func (d *countingDispatcher) Dispatch(_ context.Context, _ bd.Context) ([]byte, error) { //nolint:gocritic
	d.calls.Add(1)
	return []byte(`ok`), nil
}

func TestRateLimiter_BurstAllowed(t *testing.T) {
	inner := &countingDispatcher{}
	rl := NewRateLimitDispatcher(inner, RateLimitConfig{
		Rate:  1000,
		Burst: 5,
	})

	ctx := bd.Context{Step: "test"}
	for i := 0; i < 5; i++ {
		_, err := rl.Dispatch(context.Background(), ctx)
		if err != nil {
			t.Fatalf("burst dispatch %d failed: %v", i, err)
		}
	}

	if inner.calls.Load() != 5 {
		t.Errorf("expected 5 inner calls, got %d", inner.calls.Load())
	}
}

func TestRateLimiter_DelaysBeyondBurst(t *testing.T) {
	inner := &countingDispatcher{}
	var hookCalls atomic.Int64
	rl := NewRateLimitDispatcher(inner, RateLimitConfig{
		Rate:  100,
		Burst: 1,
		OnLimit: func() {
			hookCalls.Add(1)
		},
	})

	ctx := bd.Context{Step: "test"}
	start := time.Now()

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = rl.Dispatch(context.Background(), ctx)
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	if inner.calls.Load() != 3 {
		t.Errorf("expected 3 inner calls, got %d", inner.calls.Load())
	}

	if hookCalls.Load() == 0 {
		t.Error("expected at least one rate limit hook call")
	}

	if rl.Waits() == 0 {
		t.Error("expected at least one wait recorded")
	}

	if elapsed > 2*time.Second {
		t.Errorf("rate limiter took too long: %v", elapsed)
	}
}

func TestRateLimiter_Unwrapper(t *testing.T) {
	inner := &countingDispatcher{}
	rl := NewRateLimitDispatcher(inner, RateLimitConfig{Rate: 10, Burst: 1})
	if rl.Inner() != bd.Dispatcher(inner) {
		t.Error("Inner() should return the wrapped dispatcher")
	}
}

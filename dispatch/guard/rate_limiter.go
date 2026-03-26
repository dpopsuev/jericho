package guard

import (
	"context"

	bd "github.com/dpopsuev/bugle/dispatch"
	"github.com/dpopsuev/bugle/resilience"
)

// RateLimitHook is called each time a dispatch is delayed by the rate limiter.
type RateLimitHook = func()

// RateLimitConfig configures a RateLimitDispatcher.
type RateLimitConfig = resilience.RateLimitConfig

// RateLimitDispatcher wraps a bd.Dispatcher with token bucket rate limiting.
// Delegates to resilience.RateLimiter for the token bucket.
type RateLimitDispatcher struct {
	inner   bd.Dispatcher
	limiter *resilience.RateLimiter
}

// NewRateLimitDispatcher wraps inner with rate limiting.
func NewRateLimitDispatcher(inner bd.Dispatcher, cfg RateLimitConfig) *RateLimitDispatcher {
	return &RateLimitDispatcher{
		inner:   inner,
		limiter: resilience.NewRateLimiter(cfg),
	}
}

// Dispatch waits for a rate limit token, then delegates to the inner dispatcher.
func (d *RateLimitDispatcher) Dispatch(ctx context.Context, dc bd.Context) ([]byte, error) {
	if err := d.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	return d.inner.Dispatch(ctx, dc)
}

// Waits returns the total number of times a dispatch was delayed.
func (d *RateLimitDispatcher) Waits() int64 { return d.limiter.Waits() }

// Inner returns the wrapped dispatcher.
func (d *RateLimitDispatcher) Inner() bd.Dispatcher { return d.inner }

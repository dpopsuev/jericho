package resilience

import (
	"context"
	"sync/atomic"

	"golang.org/x/time/rate"
)

// RateLimitConfig configures a RateLimiter.
type RateLimitConfig struct {
	Rate    float64 // requests per second (default 10)
	Burst   int     // max burst capacity (default 1)
	OnLimit func()  // called when throttled (optional)
}

// RateLimiter implements token bucket rate limiting.
type RateLimiter struct {
	limiter *rate.Limiter
	onLimit func()
	waits   atomic.Int64
}

// NewRateLimiter creates a rate limiter with the given config.
func NewRateLimiter(cfg RateLimitConfig) *RateLimiter {
	r := cfg.Rate
	if r <= 0 {
		r = 10
	}
	burst := cfg.Burst
	if burst <= 0 {
		burst = 1
	}
	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(r), burst),
		onLimit: cfg.OnLimit,
	}
}

// Allow checks if a request is allowed immediately (non-blocking).
func (rl *RateLimiter) Allow() bool {
	return rl.limiter.Allow()
}

// Wait blocks until a token is available or the context is cancelled.
// Returns immediately if a token is available. Calls OnLimit when throttled.
func (rl *RateLimiter) Wait(ctx context.Context) error {
	if rl.limiter.Allow() {
		return nil
	}
	rl.waits.Add(1)
	if rl.onLimit != nil {
		rl.onLimit()
	}
	return rl.limiter.Wait(ctx)
}

// Waits returns the total number of times a call was throttled.
func (rl *RateLimiter) Waits() int64 { return rl.waits.Load() }

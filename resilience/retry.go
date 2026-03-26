// Package resilience provides generic production resilience primitives:
// retry with exponential backoff, circuit breaker, and rate limiter.
//
// These are protocol-agnostic — they wrap any function, not a specific
// transport interface. Both ACP clients and dispatch.Dispatcher use them.
package resilience

import (
	"context"
	"math"
	"math/rand/v2"
	"time"
)

// RetryConfig controls retry behavior.
type RetryConfig struct {
	MaxAttempts int                // default 3
	BaseDelay   time.Duration     // default 500ms
	MaxDelay    time.Duration     // default 30s
	Jitter      bool              // default true — adds ±25% randomness
	Retryable   func(error) bool  // classify errors; nil = retry all
}

func (c RetryConfig) defaults() RetryConfig {
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = 3
	}
	if c.BaseDelay <= 0 {
		c.BaseDelay = 500 * time.Millisecond
	}
	if c.MaxDelay <= 0 {
		c.MaxDelay = 30 * time.Second
	}
	return c
}

// Retry calls fn up to MaxAttempts times with exponential backoff.
// Returns the last error if all attempts fail. Respects context cancellation.
func Retry(ctx context.Context, cfg RetryConfig, fn func() error) error {
	cfg = cfg.defaults()

	var lastErr error
	for attempt := range cfg.MaxAttempts {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		// Check if error is retryable.
		if cfg.Retryable != nil && !cfg.Retryable(lastErr) {
			return lastErr
		}

		// Don't sleep after last attempt.
		if attempt == cfg.MaxAttempts-1 {
			break
		}

		delay := backoff(attempt, cfg.BaseDelay, cfg.MaxDelay, cfg.Jitter)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return lastErr
}

// backoff computes the delay for the given attempt using exponential backoff.
func backoff(attempt int, base, max time.Duration, jitter bool) time.Duration {
	delay := time.Duration(float64(base) * math.Pow(2, float64(attempt)))
	if delay > max {
		delay = max
	}
	if jitter {
		// ±25% jitter
		factor := 0.75 + rand.Float64()*0.5
		delay = time.Duration(float64(delay) * factor)
	}
	return delay
}

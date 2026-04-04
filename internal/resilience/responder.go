package resilience

import (
	"context"

	"github.com/dpopsuev/troupe/internal/protocol"
)

// CircuitBreakerResponder wraps a protocol.Responder with circuit breaker protection.
type CircuitBreakerResponder struct {
	inner   protocol.Responder
	breaker *CircuitBreaker
}

// NewCircuitBreakerResponder wraps inner with circuit breaker protection.
func NewCircuitBreakerResponder(inner protocol.Responder, cfg CircuitConfig) *CircuitBreakerResponder {
	return &CircuitBreakerResponder{
		inner:   inner,
		breaker: NewCircuitBreaker(cfg),
	}
}

// Respond delegates to the inner responder if the circuit allows it.
func (r *CircuitBreakerResponder) RespondTo(ctx context.Context, prompt string) (string, error) {
	var result string
	err := r.breaker.Call(func() error {
		var callErr error
		result, callErr = r.inner.RespondTo(ctx, prompt)
		return callErr
	})
	return result, err
}

// State returns the current circuit state.
func (r *CircuitBreakerResponder) State() CircuitState { return r.breaker.State() }

// Inner returns the wrapped responder.
func (r *CircuitBreakerResponder) Inner() protocol.Responder { return r.inner }

// RateLimitResponder wraps a protocol.Responder with token bucket rate limiting.
type RateLimitResponder struct {
	inner   protocol.Responder
	limiter *RateLimiter
}

// NewRateLimitResponder wraps inner with rate limiting.
func NewRateLimitResponder(inner protocol.Responder, cfg RateLimitConfig) *RateLimitResponder {
	return &RateLimitResponder{
		inner:   inner,
		limiter: NewRateLimiter(cfg),
	}
}

// Respond waits for a rate limit token, then delegates.
func (r *RateLimitResponder) RespondTo(ctx context.Context, prompt string) (string, error) {
	if err := r.limiter.Wait(ctx); err != nil {
		return "", err
	}
	return r.inner.RespondTo(ctx, prompt)
}

// Waits returns the total number of times a call was delayed.
func (r *RateLimitResponder) Waits() int64 { return r.limiter.Waits() }

// Inner returns the wrapped responder.
func (r *RateLimitResponder) Inner() protocol.Responder { return r.inner }

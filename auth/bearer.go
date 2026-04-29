package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"os"

	
	"github.com/dpopsuev/tangle/resilience"
)

// Sentinel errors.
var (
	ErrMissingToken = errors.New("auth: missing or empty token")
	ErrInvalidToken = errors.New("auth: invalid token")
)

// Bearer validates tokens against an environment variable.
// Simple shared-secret auth for Docker Compose and single-node deployments.
type Bearer struct {
	envVar  string
	limiter *resilience.RateLimiter // optional rate limiter on auth failures
}

// BearerOption configures a Bearer authenticator.
type BearerOption func(*Bearer)

// WithRateLimit adds rate limiting to authentication attempts.
func WithRateLimit(cfg resilience.RateLimitConfig) BearerOption {
	return func(b *Bearer) {
		b.limiter = resilience.NewRateLimiter(cfg)
	}
}

// NewBearer creates a bearer authenticator that reads the expected token
// from the given environment variable.
func NewBearer(envVar string, opts ...BearerOption) *Bearer {
	b := &Bearer{envVar: envVar}
	for _, o := range opts {
		o(b)
	}
	return b
}

// Authenticate compares the provided token against the env var value.
// If rate limiting is configured, blocks until a token is available.
func (b *Bearer) Authenticate(ctx context.Context, token string) (Identity, error) {
	if b.limiter != nil {
		if err := b.limiter.Wait(ctx); err != nil {
			return Identity{}, err
		}
	}
	if token == "" {
		return Identity{}, ErrMissingToken
	}
	expected := os.Getenv(b.envVar)
	if expected == "" {
		return Identity{}, ErrMissingToken
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
		return Identity{}, ErrInvalidToken
	}
	return Identity{Subject: "bearer:" + b.envVar}, nil
}

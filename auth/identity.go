package auth

import "context"

// Identity represents an authenticated caller.
type Identity struct {
	Subject string            `json:"subject"`
	Claims  map[string]string `json:"claims,omitempty"`
}

// Authenticator resolves a token to an Identity.
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (Identity, error)
}

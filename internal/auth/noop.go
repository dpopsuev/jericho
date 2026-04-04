// Package auth provides Authenticator and Authorizer adapters for the
// Bugle Protocol. Consumers select the adapter matching their infrastructure.
package auth

import (
	"context"

	"github.com/dpopsuev/jericho/internal/protocol"
)

// Noop allows all requests. Use for development and testing.
type Noop struct{}

// Authenticate always succeeds with a generic symbol.
func (Noop) Authenticate(_ context.Context, token string) (protocol.Identity, error) {
	return protocol.Identity{Subject: "anonymous"}, nil
}

// Authorize always allows.
func (Noop) Authorize(_ protocol.Identity, _ protocol.Action) error {
	return nil
}

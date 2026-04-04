package protocol

import (
	"context"
	"fmt"
)

// AuthServer wraps a Server with mandatory authentication and authorization.
// Every request (except Start) must carry a valid AuthToken. Start is exempt
// because the caller needs to initiate a session before it has a token.
type AuthServer struct {
	inner Server
	authn Authenticator
	authz Authorizer
}

// NewAuthServer creates an authenticated Server wrapper.
func NewAuthServer(inner Server, authn Authenticator, authz Authorizer) *AuthServer {
	return &AuthServer{inner: inner, authn: authn, authz: authz}
}

// Start delegates directly — no auth required to initiate a session.
func (s *AuthServer) Start(ctx context.Context, req StartRequest) (StartResponse, error) {
	return s.inner.Start(ctx, req)
}

// Pull authenticates and authorizes before delegating.
func (s *AuthServer) Pull(ctx context.Context, req PullRequest) (PullResponse, error) {
	if err := s.check(ctx, req.Auth, ActionPull); err != nil {
		return PullResponse{}, err
	}
	return s.inner.Pull(ctx, req)
}

// Push authenticates and authorizes before delegating.
func (s *AuthServer) Push(ctx context.Context, req PushRequest) (PushResponse, error) {
	if err := s.check(ctx, req.Auth, ActionPush); err != nil {
		return PushResponse{}, err
	}
	return s.inner.Push(ctx, req)
}

// Cancel authenticates and authorizes before delegating.
func (s *AuthServer) Cancel(ctx context.Context, req CancelRequest) (CancelResponse, error) {
	if err := s.check(ctx, req.Auth, ActionCancel); err != nil {
		return CancelResponse{}, err
	}
	return s.inner.Cancel(ctx, req)
}

// Status authenticates and authorizes before delegating.
func (s *AuthServer) Status(ctx context.Context, req StatusRequest) (StatusResponse, error) {
	if err := s.check(ctx, req.Auth, ActionStatus); err != nil {
		return StatusResponse{}, err
	}
	return s.inner.Status(ctx, req)
}

func (s *AuthServer) check(ctx context.Context, token *AuthToken, action Action) error {
	if token == nil {
		return fmt.Errorf("auth: %w", ErrNoActiveSession)
	}
	identity, err := s.authn.Authenticate(ctx, token.Token)
	if err != nil {
		return fmt.Errorf("auth: authenticate: %w", err)
	}
	if err := s.authz.Authorize(identity, action); err != nil {
		return fmt.Errorf("auth: authorize %s: %w", action, err)
	}
	return nil
}

// Compile-time check.
var _ Server = (*AuthServer)(nil)

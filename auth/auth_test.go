package auth

import (
	"context"
	"errors"
	"testing"
)

func TestBearer_Authenticate(t *testing.T) {
	const envVar = "TEST_TROUPE_AUTH_TOKEN"
	t.Setenv(envVar, "secret-123")

	b := NewBearer(envVar)

	t.Run("valid token", func(t *testing.T) {
		id, err := b.Authenticate(context.Background(), "secret-123")
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if id.Subject != "bearer:"+envVar {
			t.Errorf("subject = %q", id.Subject)
		}
	})

	t.Run("wrong token", func(t *testing.T) {
		_, err := b.Authenticate(context.Background(), "wrong")
		if !errors.Is(err, ErrInvalidToken) {
			t.Errorf("error = %v, want ErrInvalidToken", err)
		}
	})

	t.Run("empty token", func(t *testing.T) {
		_, err := b.Authenticate(context.Background(), "")
		if !errors.Is(err, ErrMissingToken) {
			t.Errorf("error = %v, want ErrMissingToken", err)
		}
	})
}

func TestBearer_MissingEnvVar(t *testing.T) {
	b := NewBearer("NONEXISTENT_VAR_12345")
	_, err := b.Authenticate(context.Background(), "any")
	if !errors.Is(err, ErrMissingToken) {
		t.Errorf("error = %v, want ErrMissingToken", err)
	}
}

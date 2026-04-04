package collective

import (
	"context"
	"errors"
	"testing"

	"github.com/dpopsuev/troupe"
)

type succeedStrategy struct{ response string }

func (s *succeedStrategy) Orchestrate(_ context.Context, _ string, _ []troupe.Actor) (string, error) {
	return s.response, nil
}

type failStrategy struct{ err error }

func (s *failStrategy) Orchestrate(_ context.Context, _ string, _ []troupe.Actor) (string, error) {
	return "", s.err
}

func TestFallback_PrimarySucceeds(t *testing.T) {
	f := &Fallback{
		Primary:  &succeedStrategy{response: "primary"},
		Fallback: &failStrategy{err: errors.New("should not be called")},
	}

	result, err := f.Orchestrate(context.Background(), "test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "primary" {
		t.Errorf("result = %q, want %q", result, "primary")
	}
}

func TestFallback_PrimaryFails_FallbackSucceeds(t *testing.T) {
	f := &Fallback{
		Primary:  &failStrategy{err: errors.New("primary failed")},
		Fallback: &succeedStrategy{response: "fallback"},
	}

	result, err := f.Orchestrate(context.Background(), "test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "fallback" {
		t.Errorf("result = %q, want %q", result, "fallback")
	}
}

func TestFallback_BothFail(t *testing.T) {
	f := &Fallback{
		Primary:  &failStrategy{err: errors.New("primary failed")},
		Fallback: &failStrategy{err: errors.New("fallback failed")},
	}

	_, err := f.Orchestrate(context.Background(), "test", nil)
	if err == nil {
		t.Fatal("expected error when both fail")
	}
	if err.Error() != "fallback failed" {
		t.Errorf("error = %q, want fallback's error", err.Error())
	}
}

func TestFallback_Composable(t *testing.T) {
	// Fallback{Fallback{fail, fail}, succeed} — nested decoration
	inner := &Fallback{
		Primary:  &failStrategy{err: errors.New("a")},
		Fallback: &failStrategy{err: errors.New("b")},
	}
	outer := &Fallback{
		Primary:  inner,
		Fallback: &succeedStrategy{response: "rescued"},
	}

	result, err := outer.Orchestrate(context.Background(), "test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "rescued" {
		t.Errorf("result = %q, want %q", result, "rescued")
	}
}

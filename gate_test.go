package troupe_test

import (
	"context"
	"errors"
	"testing"

	"github.com/dpopsuev/troupe"
)

func TestAlwaysPass(t *testing.T) {
	ok, reason, err := troupe.AlwaysPass(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("AlwaysPass returned false")
	}
	if reason != "" {
		t.Fatalf("unexpected reason: %q", reason)
	}
}

func TestAlwaysDeny(t *testing.T) {
	ok, reason, err := troupe.AlwaysDeny(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("AlwaysDeny returned true")
	}
	if reason == "" {
		t.Fatal("AlwaysDeny should provide a reason")
	}
}

func TestComposeGates_AllPass(t *testing.T) {
	g := troupe.ComposeGates(troupe.AlwaysPass, troupe.AlwaysPass, troupe.AlwaysPass)
	ok, _, err := g(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("all-pass composition should pass")
	}
}

func TestComposeGates_ShortCircuit(t *testing.T) {
	called := false
	last := troupe.Gate(func(_ context.Context, _ any) (bool, string, error) {
		called = true
		return true, "", nil
	})

	g := troupe.ComposeGates(troupe.AlwaysPass, troupe.AlwaysDeny, last)
	ok, reason, err := g(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("should have been rejected")
	}
	if reason == "" {
		t.Fatal("rejection should include reason")
	}
	if called {
		t.Fatal("third gate should not have been called after rejection")
	}
}

func TestComposeGates_Empty(t *testing.T) {
	g := troupe.ComposeGates()
	ok, _, err := g(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("empty composition should pass")
	}
}

func TestComposeGates_ErrorStops(t *testing.T) {
	boom := errors.New("boom")
	errGate := troupe.Gate(func(_ context.Context, _ any) (bool, string, error) {
		return false, "", boom
	})

	g := troupe.ComposeGates(troupe.AlwaysPass, errGate, troupe.AlwaysPass)
	_, _, err := g(context.Background(), "test")
	if !errors.Is(err, boom) {
		t.Fatalf("expected boom error, got: %v", err)
	}
}

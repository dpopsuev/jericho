package testkit_test

import (
	"context"
	"testing"

	"github.com/dpopsuev/tangle/testkit"
)

func TestStubActorFunc_CyclesResponses(t *testing.T) {
	actor := testkit.StubActorFunc("alpha", "bravo", "charlie")
	ctx := context.Background()

	want := []string{"alpha", "bravo", "charlie", "alpha", "bravo"}
	for i, expected := range want {
		got, err := actor(ctx, "ignored")
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
		if got != expected {
			t.Errorf("call %d: got %q, want %q", i, got, expected)
		}
	}
}

func TestStubActorFunc_SingleResponse(t *testing.T) {
	actor := testkit.StubActorFunc("only")
	ctx := context.Background()

	for range 3 {
		got, err := actor(ctx, "anything")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "only" {
			t.Errorf("got %q, want %q", got, "only")
		}
	}
}

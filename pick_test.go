package troupe_test

import (
	"context"
	"testing"

	"github.com/dpopsuev/troupe"
)

func TestFirstMatch_ReturnsRequestedCount(t *testing.T) {
	candidates := []troupe.ActorConfig{{Role: "a"}, {Role: "b"}, {Role: "c"}}
	result := troupe.FirstMatch{}.Choose(context.Background(), candidates, troupe.Preferences{Count: 2})
	if len(result) != 2 {
		t.Fatalf("got %d, want 2", len(result))
	}
}

func TestFirstMatch_ClampsToAvailable(t *testing.T) {
	candidates := []troupe.ActorConfig{{Role: "a"}}
	result := troupe.FirstMatch{}.Choose(context.Background(), candidates, troupe.Preferences{Count: 5})
	if len(result) != 1 {
		t.Fatalf("got %d, want 1", len(result))
	}
}

func TestFirstMatch_DefaultCountOne(t *testing.T) {
	candidates := []troupe.ActorConfig{{Role: "a"}, {Role: "b"}}
	result := troupe.FirstMatch{}.Choose(context.Background(), candidates, troupe.Preferences{})
	if len(result) != 1 {
		t.Fatalf("got %d, want 1 (default)", len(result))
	}
}

var _ troupe.PickStrategy = troupe.FirstMatch{}

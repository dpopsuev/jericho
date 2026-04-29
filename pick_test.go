package troupe_test

import (
	"context"
	"testing"

	"github.com/dpopsuev/tangle"
)

func TestPickAll(t *testing.T) {
	items := []string{"a", "b", "c"}
	result := troupe.PickAll[string]()(context.Background(), items)
	if len(result) != 3 {
		t.Fatalf("PickAll returned %d items, want 3", len(result))
	}
}

func TestPickFirst(t *testing.T) {
	items := []string{"a", "b", "c", "d"}

	result := troupe.PickFirst[string](2)(context.Background(), items)
	if len(result) != 2 {
		t.Fatalf("PickFirst(2) returned %d items, want 2", len(result))
	}
	if result[0] != "a" || result[1] != "b" {
		t.Fatalf("PickFirst(2) returned %v, want [a b]", result)
	}
}

func TestPickFirst_MoreThanAvailable(t *testing.T) {
	items := []string{"a", "b"}
	result := troupe.PickFirst[string](5)(context.Background(), items)
	if len(result) != 2 {
		t.Fatalf("PickFirst(5) with 2 items returned %d, want 2", len(result))
	}
}

func TestPickFirst_Zero(t *testing.T) {
	items := []string{"a", "b", "c"}
	result := troupe.PickFirst[string](0)(context.Background(), items)
	if len(result) != 1 {
		t.Fatalf("PickFirst(0) returned %d items, want 1", len(result))
	}
}

func TestPickAll_Empty(t *testing.T) {
	result := troupe.PickAll[string]()(context.Background(), nil)
	if len(result) != 0 {
		t.Fatalf("PickAll on nil returned %d items, want 0", len(result))
	}
}

func TestPick_CustomFilter(t *testing.T) {
	onlyLong := troupe.Pick[string](func(_ context.Context, items []string) []string {
		var out []string
		for _, s := range items {
			if len(s) > 1 {
				out = append(out, s)
			}
		}
		return out
	})

	items := []string{"a", "bb", "c", "ddd"}
	result := onlyLong(context.Background(), items)
	if len(result) != 2 {
		t.Fatalf("custom filter returned %d items, want 2", len(result))
	}
}

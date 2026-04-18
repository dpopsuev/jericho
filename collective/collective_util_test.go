package collective_test

import (
	"context"
	"testing"

	"github.com/dpopsuev/troupe/collective"
	"github.com/dpopsuev/troupe/testkit"
)

func TestSpawnCollective_WithMockBroker(t *testing.T) {
	broker := testkit.NewMockBroker(3)
	actor, err := collective.SpawnCollective(context.Background(), broker, 3, &collective.RoundRobin{})
	if err != nil {
		t.Fatalf("SpawnCollective: %v", err)
	}
	if actor == nil {
		t.Fatal("returned nil actor")
	}

	result, err := actor.Perform(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("Perform: %v", err)
	}
	if result == "" {
		t.Error("Perform returned empty result")
	}
}

func TestSpawnCollective_Ready(t *testing.T) {
	broker := testkit.NewMockBroker(2)
	actor, err := collective.SpawnCollective(context.Background(), broker, 2, &collective.RoundRobin{})
	if err != nil {
		t.Fatalf("SpawnCollective: %v", err)
	}
	if !actor.Ready() {
		t.Error("Ready() = false, want true")
	}
}

func TestSpawnCollective_Kill(t *testing.T) {
	broker := testkit.NewMockBroker(2)
	actor, err := collective.SpawnCollective(context.Background(), broker, 2, &collective.RoundRobin{})
	if err != nil {
		t.Fatalf("SpawnCollective: %v", err)
	}
	if err := actor.Kill(context.Background()); err != nil {
		t.Fatalf("Kill: %v", err)
	}
}

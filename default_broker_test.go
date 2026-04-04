package troupe_test

import (
	"context"
	"testing"

	"github.com/dpopsuev/troupe"
)

func TestNewBroker_EmptyEndpoint_ReturnsLocal(t *testing.T) {
	broker := troupe.NewBroker("")
	if broker == nil {
		t.Fatal("NewBroker(\"\") returned nil")
	}
}

func TestNewBroker_HTTPSEndpoint_ReturnsRemote(t *testing.T) {
	broker := troupe.NewBroker("https://cluster:8080")
	if broker == nil {
		t.Fatal("NewBroker(\"https://...\") returned nil")
	}
}

func TestDefaultBroker_Pick_DefaultCount(t *testing.T) {
	broker := troupe.NewBroker("")
	configs, err := broker.Pick(context.Background(), troupe.Preferences{})
	if err != nil {
		t.Fatalf("Pick: %v", err)
	}
	if len(configs) != 1 {
		t.Errorf("Pick with empty prefs: got %d configs, want 1", len(configs))
	}
}

func TestDefaultBroker_Pick_ExplicitCount(t *testing.T) {
	broker := troupe.NewBroker("")
	configs, err := broker.Pick(context.Background(), troupe.Preferences{Count: 3, Role: "worker"})
	if err != nil {
		t.Fatalf("Pick: %v", err)
	}
	if len(configs) != 3 {
		t.Errorf("Pick count=3: got %d configs, want 3", len(configs))
	}
}

func TestDefaultBroker_Spawn_NoLauncher(t *testing.T) {
	broker := troupe.NewBroker("")
	_, err := broker.Spawn(context.Background(), troupe.ActorConfig{Model: "sonnet"})
	if err == nil {
		t.Fatal("expected error for spawn without launcher")
	}
}

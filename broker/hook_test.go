package broker_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/dpopsuev/tangle"
	"github.com/dpopsuev/tangle/broker"
	"github.com/dpopsuev/tangle/world"
)

type noopDriver struct{}

func (noopDriver) Start(_ context.Context, _ world.EntityID, _ troupe.AgentConfig) error { return nil }
func (noopDriver) Stop(_ context.Context, _ world.EntityID) error                        { return nil }

// --- test helpers ---

type testSpawnHook struct {
	preErr error
	spawns int
	mu     sync.Mutex
}

func (h *testSpawnHook) Name() string { return "test-spawn" }

func (h *testSpawnHook) PreSpawn(_ context.Context, _ troupe.AgentConfig) error {
	return h.preErr
}

func (h *testSpawnHook) PostSpawn(_ context.Context, _ troupe.AgentConfig, _ troupe.Agent, _ error) {
	h.mu.Lock()
	h.spawns++
	h.mu.Unlock()
}

type testPerformHook struct {
	preErr   error
	observed []string
	mu       sync.Mutex
}

func (h *testPerformHook) Name() string { return "test-perform" }

func (h *testPerformHook) PrePerform(_ context.Context, _ string) error {
	return h.preErr
}

func (h *testPerformHook) PostPerform(_ context.Context, _, response string, _ error) {
	h.mu.Lock()
	h.observed = append(h.observed, response)
	h.mu.Unlock()
}

// --- tests ---

func TestHook_PreSpawn_Reject(t *testing.T) {
	rejectHook := &testSpawnHook{preErr: errors.New("budget exceeded")}
	b := broker.New("", broker.WithDriver(noopDriver{}), broker.WithHook(rejectHook))
	_, err := b.Spawn(context.Background(), troupe.AgentConfig{Role: "test"})
	if err == nil {
		t.Fatal("expected PreSpawn rejection")
	}
	if !errors.Is(err, rejectHook.preErr) {
		t.Fatalf("expected wrapped budget error, got: %v", err)
	}
}

func TestHook_PostSpawn_Called(t *testing.T) {
	obs := &testSpawnHook{}
	b := broker.New("", broker.WithDriver(noopDriver{}), broker.WithHook(obs))
	_, err := b.Spawn(context.Background(), troupe.AgentConfig{Role: "test"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if obs.spawns != 1 {
		t.Errorf("PostSpawn called %d times, want 1", obs.spawns)
	}
}

func TestHook_NilHook_NoPanic(t *testing.T) {
	// WithHook(nil) must not panic or register anything.
	b := broker.New("", broker.WithDriver(noopDriver{}), broker.WithHook(nil))
	_, err := b.Spawn(context.Background(), troupe.AgentConfig{Role: "test"})
	if err != nil {
		t.Fatalf("Spawn with nil hook: %v", err)
	}
}

var _ broker.SpawnHook = (*testSpawnHook)(nil)
var _ broker.PerformHook = (*testPerformHook)(nil)

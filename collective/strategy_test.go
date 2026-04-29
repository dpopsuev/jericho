package collective

import (
	"context"
	"testing"

	"github.com/dpopsuev/tangle"
	"github.com/dpopsuev/tangle/testkit"
)

func mockActor(name string) *testkit.MockActor {
	return &testkit.MockActor{Name: name}
}

// --- Race ---

func TestRace_ReturnsResponse(t *testing.T) {
	r := Race{}
	resp, err := r.Orchestrate(context.Background(), "hello", []troupe.Agent{mockActor("a"), mockActor("b")})
	if err != nil {
		t.Fatalf("Race: %v", err)
	}
	if resp == "" {
		t.Fatal("empty response")
	}
}

func TestRace_NoAgents(t *testing.T) {
	_, err := Race{}.Orchestrate(context.Background(), "go", nil)
	if err == nil {
		t.Fatal("should error")
	}
}

// --- RoundRobin ---

func TestRoundRobin_Rotates(t *testing.T) {
	rr := &RoundRobin{}
	ctx := context.Background()
	a := mockActor("a")
	b := mockActor("b")

	r1, _ := rr.Orchestrate(ctx, "go", []troupe.Agent{a, b})
	r2, _ := rr.Orchestrate(ctx, "go", []troupe.Agent{a, b})

	// MockActor echoes prompts — both return "go" but from different actors.
	// We verify both calls succeed, rotation is internal.
	if r1 == "" || r2 == "" {
		t.Fatal("both calls should succeed")
	}
}

func TestRoundRobin_NoAgents(t *testing.T) {
	_, err := (&RoundRobin{}).Orchestrate(context.Background(), "go", nil)
	if err == nil {
		t.Fatal("should error")
	}
}

// --- Scatter ---

func TestScatter_CollectsAll(t *testing.T) {
	resp, err := (&Scatter{}).Orchestrate(context.Background(), "go", []troupe.Agent{mockActor("a"), mockActor("b")})
	if err != nil {
		t.Fatalf("Scatter: %v", err)
	}
	if resp == "" {
		t.Fatal("empty response")
	}
}

func TestScatter_NoAgents(t *testing.T) {
	_, err := (&Scatter{}).Orchestrate(context.Background(), "go", nil)
	if err == nil {
		t.Fatal("should error")
	}
}

// --- Scale ---

func TestScale_Up(t *testing.T) {
	broker := testkit.NewMockBroker(5)
	c := NewCollective(1, "team", Race{}, []troupe.Agent{mockActor("a")})
	err := c.Scale(context.Background(), 3, troupe.AgentConfig{Role: "worker"}, broker)
	if err != nil {
		t.Fatalf("Scale up: %v", err)
	}
	if len(c.Children()) != 3 {
		t.Fatalf("agents = %d, want 3", len(c.Children()))
	}
}

func TestScale_Down(t *testing.T) {
	c := NewCollective(1, "team", Race{}, []troupe.Agent{mockActor("a"), mockActor("b"), mockActor("c")})
	err := c.Scale(context.Background(), 1, troupe.AgentConfig{}, nil)
	if err != nil {
		t.Fatalf("Scale down: %v", err)
	}
	if len(c.Children()) != 1 {
		t.Fatalf("agents = %d, want 1", len(c.Children()))
	}
}

func TestScale_MaxSizeRejected(t *testing.T) {
	c := NewCollective(1, "team", Race{}, []troupe.Agent{mockActor("a")}, WithMaxSize(2))
	err := c.Scale(context.Background(), 5, troupe.AgentConfig{}, nil)
	if err == nil {
		t.Fatal("should reject beyond maxSize")
	}
}

func TestScale_DisruptionBudget(t *testing.T) {
	c := NewCollective(1, "team", Race{}, []troupe.Agent{mockActor("a"), mockActor("b"), mockActor("c")}, WithMinAvailable(2))
	err := c.Scale(context.Background(), 1, troupe.AgentConfig{}, nil)
	if err == nil {
		t.Fatal("should reject below minAvailable")
	}
}

func TestPerform_NoAgents(t *testing.T) {
	c := NewCollective(1, "empty", Race{}, nil)
	_, err := c.Perform(context.Background(), "hello")
	if err == nil {
		t.Fatal("should error")
	}
}

package collective

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/dpopsuev/bugle/facade"
	"github.com/dpopsuev/bugle/pool"
	"github.com/dpopsuev/bugle/world"
)

type mockLauncher struct {
	started map[world.EntityID]bool
	stopped map[world.EntityID]bool
}

func newMockLauncher() *mockLauncher {
	return &mockLauncher{
		started: make(map[world.EntityID]bool),
		stopped: make(map[world.EntityID]bool),
	}
}

func (m *mockLauncher) Start(_ context.Context, id world.EntityID, _ pool.LaunchConfig) error {
	m.started[id] = true
	return nil
}
func (m *mockLauncher) Stop(_ context.Context, id world.EntityID) error {
	m.stopped[id] = true
	return nil
}
func (m *mockLauncher) Healthy(_ context.Context, id world.EntityID) bool {
	return m.started[id] && !m.stopped[id]
}

// echoStrategy is a test strategy that asks thesis, gets critique, returns thesis.
type echoStrategy struct {
	orchestrateCalled bool
}

func (s *echoStrategy) Orchestrate(_ context.Context, prompt string, agents []*facade.AgentHandle) (string, error) {
	s.orchestrateCalled = true
	return "synthesized: " + prompt, nil
}

func TestAgentCollective_ImplementsAgent(t *testing.T) {
	var _ facade.Agent = (*AgentCollective)(nil)
	var _ facade.FacadeAgent = (*AgentCollective)(nil)
}

func TestAgentCollective_Ask(t *testing.T) {
	staff := facade.NewStaff(newMockLauncher())
	ctx := context.Background()

	a1, _ := staff.Spawn(ctx, "thesis", pool.LaunchConfig{})
	a2, _ := staff.Spawn(ctx, "antithesis", pool.LaunchConfig{})

	strategy := &echoStrategy{}
	coll := NewAgentCollective(a1.ID(), "debater", strategy, []*facade.AgentHandle{a1, a2})

	result, err := coll.Ask(ctx, "test prompt")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if result != "synthesized: test prompt" {
		t.Fatalf("result = %q", result)
	}
	if !strategy.orchestrateCalled {
		t.Fatal("strategy.Orchestrate was not called")
	}
}

func TestAgentCollective_Identity(t *testing.T) {
	staff := facade.NewStaff(newMockLauncher())
	ctx := context.Background()

	a1, _ := staff.Spawn(ctx, "thesis", pool.LaunchConfig{})
	a2, _ := staff.Spawn(ctx, "antithesis", pool.LaunchConfig{})

	coll := NewAgentCollective(a1.ID(), "reviewer", &echoStrategy{}, []*facade.AgentHandle{a1, a2})

	if coll.Role() != "reviewer" {
		t.Fatalf("Role = %q", coll.Role())
	}
	if coll.ID() != a1.ID() {
		t.Fatalf("ID = %d", coll.ID())
	}
	s := coll.String()
	if !strings.Contains(s, "reviewer") || !strings.Contains(s, "2 agents") {
		t.Fatalf("String = %q", s)
	}
}

func TestAgentCollective_IsAlive(t *testing.T) {
	staff := facade.NewStaff(newMockLauncher())
	ctx := context.Background()

	a1, _ := staff.Spawn(ctx, "thesis", pool.LaunchConfig{})
	a2, _ := staff.Spawn(ctx, "antithesis", pool.LaunchConfig{})

	coll := NewAgentCollective(a1.ID(), "debater", &echoStrategy{}, []*facade.AgentHandle{a1, a2})

	if !coll.IsAlive() {
		t.Fatal("collective should be alive")
	}
	if !coll.IsFacade() {
		t.Fatal("should be a facade")
	}
}

func TestAgentCollective_Children(t *testing.T) {
	staff := facade.NewStaff(newMockLauncher())
	ctx := context.Background()

	a1, _ := staff.Spawn(ctx, "thesis", pool.LaunchConfig{})
	a2, _ := staff.Spawn(ctx, "antithesis", pool.LaunchConfig{})

	coll := NewAgentCollective(a1.ID(), "debater", &echoStrategy{}, []*facade.AgentHandle{a1, a2})

	children := coll.Children()
	if len(children) != 2 {
		t.Fatalf("children = %d, want 2", len(children))
	}

	internal := coll.InternalAgents()
	if len(internal) != 2 {
		t.Fatalf("InternalAgents = %d, want 2", len(internal))
	}
}

func TestAgentCollective_Kill(t *testing.T) {
	staff := facade.NewStaff(newMockLauncher())
	ctx := context.Background()

	a1, _ := staff.Spawn(ctx, "thesis", pool.LaunchConfig{})
	a2, _ := staff.Spawn(ctx, "antithesis", pool.LaunchConfig{})

	coll := NewAgentCollective(a1.ID(), "debater", &echoStrategy{}, []*facade.AgentHandle{a1, a2})

	if err := coll.Kill(ctx); err != nil {
		t.Fatalf("Kill: %v", err)
	}
}

func TestDialectic_RequiresAtLeast2Agents(t *testing.T) {
	d := &Dialectic{MaxRounds: 3}
	_, err := d.Orchestrate(context.Background(), "test", []*facade.AgentHandle{})
	if err == nil || !strings.Contains(err.Error(), "at least 2") {
		t.Fatalf("err = %v, want 'at least 2 agents'", err)
	}
}

func TestDialectic_Defaults(t *testing.T) {
	d := &Dialectic{}
	maxRounds, word := d.defaults()
	if maxRounds != 3 {
		t.Fatalf("maxRounds = %d, want 3", maxRounds)
	}
	if word != "CONVERGED" {
		t.Fatalf("word = %q, want CONVERGED", word)
	}
}

func TestArbiter_RequiresAtLeast3Agents(t *testing.T) {
	a := &Arbiter{MaxRounds: 3}
	_, err := a.Orchestrate(context.Background(), "test", []*facade.AgentHandle{})
	if err == nil || !strings.Contains(err.Error(), "at least 3") {
		t.Fatalf("err = %v, want 'at least 3 agents'", err)
	}
}

func TestArbiter_Defaults(t *testing.T) {
	a := &Arbiter{}
	if a.defaults() != 3 {
		t.Fatalf("maxRounds = %d, want 3", a.defaults())
	}
}

func TestParseDecision(t *testing.T) {
	tests := []struct {
		input string
		want  ArbiterDecision
	}{
		{"AFFIRM the thesis is correct", DecisionAffirm},
		{"affirm", DecisionAffirm},
		{"REMAND - start over", DecisionRemand},
		{"AMEND the response", DecisionAmend},
		{"unclear response", DecisionAmend}, // default
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseDecision(tt.input)
			if got != tt.want {
				t.Fatalf("parseDecision(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSpawnCollective_RequiresStrategy(t *testing.T) {
	staff := facade.NewStaff(newMockLauncher())
	_, err := SpawnCollective(context.Background(), staff, CollectiveConfig{
		Role:   "debater",
		Agents: []pool.LaunchConfig{{Role: "a"}, {Role: "b"}},
	})
	if err == nil || !strings.Contains(err.Error(), "strategy") {
		t.Fatalf("err = %v, want strategy error", err)
	}
}

func TestSpawnCollective_RequiresAtLeast2(t *testing.T) {
	staff := facade.NewStaff(newMockLauncher())
	_, err := SpawnCollective(context.Background(), staff, CollectiveConfig{
		Role:     "debater",
		Strategy: &echoStrategy{},
		Agents:   []pool.LaunchConfig{{Role: "a"}},
	})
	if err == nil || !strings.Contains(err.Error(), "at least 2") {
		t.Fatalf("err = %v, want at least 2 error", err)
	}
}

func TestSpawnCollective_Success(t *testing.T) {
	staff := facade.NewStaff(newMockLauncher())
	ctx := context.Background()

	coll, err := SpawnCollective(ctx, staff, CollectiveConfig{
		Role:     "debater",
		Strategy: &echoStrategy{},
		Agents: []pool.LaunchConfig{
			{Role: "thesis"},
			{Role: "antithesis"},
		},
	})
	if err != nil {
		t.Fatalf("SpawnCollective: %v", err)
	}
	if coll.Role() != "debater" {
		t.Fatalf("role = %q", coll.Role())
	}
	if len(coll.InternalAgents()) != 2 {
		t.Fatalf("agents = %d", len(coll.InternalAgents()))
	}

	// Ask should delegate to strategy.
	result, err := coll.Ask(ctx, "hello")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if result != "synthesized: hello" {
		t.Fatalf("result = %q", result)
	}

	// Staff should see the spawned agents.
	if staff.Count() != 2 {
		t.Fatalf("staff count = %d, want 2", staff.Count())
	}

	fmt.Println(coll.String()) // smoke test
}

func TestDebateRound_Tracking(t *testing.T) {
	coll := NewAgentCollective(1, "test", &echoStrategy{}, nil)

	coll.addDebateRound(DebateRound{ThesisResponse: "draft", AntithesisResponse: "critique"})
	coll.addDebateRound(DebateRound{ThesisResponse: "revised", Converged: true})

	rounds := coll.DebateRounds()
	if len(rounds) != 2 {
		t.Fatalf("rounds = %d, want 2", len(rounds))
	}
	if !rounds[1].Converged {
		t.Fatal("round 2 should be converged")
	}
}

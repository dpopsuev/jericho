package collective

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/dpopsuev/jericho"
	"github.com/dpopsuev/jericho/internal/agent"
	"github.com/dpopsuev/jericho/internal/warden"
	"github.com/dpopsuev/jericho/world"
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

func (m *mockLauncher) Start(_ context.Context, id world.EntityID, _ warden.AgentConfig) error {
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

func (s *echoStrategy) Orchestrate(_ context.Context, prompt string, agents []jericho.Actor) (string, error) {
	s.orchestrateCalled = true
	return "synthesized: " + prompt, nil
}

func TestAgentCollective_ImplementsAgent(t *testing.T) {
}

func TestAgentCollective_Ask(t *testing.T) {
	staff := agent.NewStaff(newMockLauncher())
	ctx := context.Background()

	a1, _ := staff.Spawn(ctx, "thesis", warden.AgentConfig{})
	a2, _ := staff.Spawn(ctx, "antithesis", warden.AgentConfig{})

	strategy := &echoStrategy{}
	coll := NewCollective(a1.ID(), "debater", strategy, []jericho.Actor{a1, a2})

	result, err := coll.Perform(ctx, "test prompt")
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
	staff := agent.NewStaff(newMockLauncher())
	ctx := context.Background()

	a1, _ := staff.Spawn(ctx, "thesis", warden.AgentConfig{})
	a2, _ := staff.Spawn(ctx, "antithesis", warden.AgentConfig{})

	coll := NewCollective(a1.ID(), "reviewer", &echoStrategy{}, []jericho.Actor{a1, a2})

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
	staff := agent.NewStaff(newMockLauncher())
	ctx := context.Background()

	a1, _ := staff.Spawn(ctx, "thesis", warden.AgentConfig{})
	a2, _ := staff.Spawn(ctx, "antithesis", warden.AgentConfig{})

	coll := NewCollective(a1.ID(), "debater", &echoStrategy{}, []jericho.Actor{a1, a2})

	if !coll.Ready() {
		t.Fatal("collective should be alive")
	}
	if !coll.IsFacade() {
		t.Fatal("should be a facade")
	}
}

func TestAgentCollective_Children(t *testing.T) {
	staff := agent.NewStaff(newMockLauncher())
	ctx := context.Background()

	a1, _ := staff.Spawn(ctx, "thesis", warden.AgentConfig{})
	a2, _ := staff.Spawn(ctx, "antithesis", warden.AgentConfig{})

	coll := NewCollective(a1.ID(), "debater", &echoStrategy{}, []jericho.Actor{a1, a2})

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
	staff := agent.NewStaff(newMockLauncher())
	ctx := context.Background()

	a1, _ := staff.Spawn(ctx, "thesis", warden.AgentConfig{})
	a2, _ := staff.Spawn(ctx, "antithesis", warden.AgentConfig{})

	coll := NewCollective(a1.ID(), "debater", &echoStrategy{}, []jericho.Actor{a1, a2})

	if err := coll.Kill(ctx); err != nil {
		t.Fatalf("Kill: %v", err)
	}
}

func TestDialectic_RequiresAtLeast2Agents(t *testing.T) {
	d := &Dialectic{MaxRounds: 3}
	_, err := d.Orchestrate(context.Background(), "test", []jericho.Actor{})
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
	_, err := a.Orchestrate(context.Background(), "test", []jericho.Actor{})
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
	staff := agent.NewStaff(newMockLauncher())
	_, err := SpawnCollectiveFromStaff(context.Background(), staff, CollectiveConfig{
		Role:   "debater",
		Agents: []warden.AgentConfig{{Role: "a"}, {Role: "b"}},
	})
	if err == nil || !strings.Contains(err.Error(), "strategy") {
		t.Fatalf("err = %v, want strategy error", err)
	}
}

func TestSpawnCollective_RequiresAtLeast2(t *testing.T) {
	staff := agent.NewStaff(newMockLauncher())
	_, err := SpawnCollectiveFromStaff(context.Background(), staff, CollectiveConfig{
		Role:     "debater",
		Strategy: &echoStrategy{},
		Agents:   []warden.AgentConfig{{Role: "a"}},
	})
	if err == nil || !strings.Contains(err.Error(), "at least 2") {
		t.Fatalf("err = %v, want at least 2 error", err)
	}
}

func TestSpawnCollective_Success(t *testing.T) {
	staff := agent.NewStaff(newMockLauncher())
	ctx := context.Background()

	coll, err := SpawnCollectiveFromStaff(ctx, staff, CollectiveConfig{
		Role:     "debater",
		Strategy: &echoStrategy{},
		Agents: []warden.AgentConfig{
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
	result, err := coll.Perform(ctx, "hello")
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
	coll := NewCollective(1, "test", &echoStrategy{}, nil)

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

// ═══════════════════════════════════════════════════════════════════════
// Gatekeeper wiring tests — RED
// ═══════════════════════════════════════════════════════════════════════

type rejectGate struct{ reason string }

func (g *rejectGate) Pass(_ context.Context, _ string) (allowed bool, reason string, err error) {
	return false, g.reason, nil
}

type passGate struct{}

func (g *passGate) Pass(_ context.Context, _ string) (allowed bool, reason string, err error) {
	return true, "ok", nil
}

func TestCollective_IngressRejects(t *testing.T) {
	staff := agent.NewStaff(newMockLauncher())
	ctx := context.Background()

	a1, _ := staff.Spawn(ctx, "thesis", warden.AgentConfig{})
	a2, _ := staff.Spawn(ctx, "antithesis", warden.AgentConfig{})

	strategy := &echoStrategy{}
	coll := NewCollective(a1.ID(), "debater", strategy, []jericho.Actor{a1, a2},
		WithIngress(&rejectGate{reason: "destructive request"}),
	)

	_, err := coll.Perform(ctx, "delete everything")
	if err == nil {
		t.Fatal("ingress should reject")
	}
	if !strings.Contains(err.Error(), "ingress rejected") {
		t.Fatalf("err = %v", err)
	}
	if strategy.orchestrateCalled {
		t.Fatal("strategy should NOT be called when ingress rejects")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Gatekeeper wiring tests — GREEN
// ═══════════════════════════════════════════════════════════════════════

func TestCollective_NoGates_BackwardCompat(t *testing.T) {
	staff := agent.NewStaff(newMockLauncher())
	ctx := context.Background()

	a1, _ := staff.Spawn(ctx, "thesis", warden.AgentConfig{})
	a2, _ := staff.Spawn(ctx, "antithesis", warden.AgentConfig{})

	coll := NewCollective(a1.ID(), "debater", &echoStrategy{}, []jericho.Actor{a1, a2})

	result, err := coll.Perform(ctx, "test")
	if err != nil {
		t.Fatalf("no gates should pass: %v", err)
	}
	if result != "synthesized: test" {
		t.Fatalf("result = %q", result)
	}
}

func TestCollective_BothGatesPass(t *testing.T) {
	staff := agent.NewStaff(newMockLauncher())
	ctx := context.Background()

	a1, _ := staff.Spawn(ctx, "thesis", warden.AgentConfig{})
	a2, _ := staff.Spawn(ctx, "antithesis", warden.AgentConfig{})

	coll := NewCollective(a1.ID(), "debater", &echoStrategy{}, []jericho.Actor{a1, a2},
		WithIngress(&passGate{}),
		WithEgress(&passGate{}),
	)

	result, err := coll.Perform(ctx, "test")
	if err != nil {
		t.Fatalf("both gates pass: %v", err)
	}
	if result != "synthesized: test" {
		t.Fatalf("result = %q", result)
	}
}

func TestCollective_EgressRejects(t *testing.T) {
	staff := agent.NewStaff(newMockLauncher())
	ctx := context.Background()

	a1, _ := staff.Spawn(ctx, "thesis", warden.AgentConfig{})
	a2, _ := staff.Spawn(ctx, "antithesis", warden.AgentConfig{})

	coll := NewCollective(a1.ID(), "debater", &echoStrategy{}, []jericho.Actor{a1, a2},
		WithEgress(&rejectGate{reason: "low confidence"}),
	)

	_, err := coll.Perform(ctx, "test")
	if err == nil {
		t.Fatal("egress should reject")
	}
	if !strings.Contains(err.Error(), "egress rejected") {
		t.Fatalf("err = %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// BudgetGatekeeper tests — RED
// ═══════════════════════════════════════════════════════════════════════

func TestBudgetGate_ZeroIsUnlimited(t *testing.T) {
	gate := &BudgetGatekeeper{MaxTokens: 0}
	ok, _, err := gate.Pass(context.Background(), "anything")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("zero maxTokens should always pass")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// BudgetGatekeeper tests — GREEN
// ═══════════════════════════════════════════════════════════════════════

func TestBudgetGate_UnderBudget(t *testing.T) {
	gate := &BudgetGatekeeper{MaxTokens: 1000, Spent: func() int { return 500 }}
	ok, _, err := gate.Pass(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("under budget should pass")
	}
}

func TestBudgetGate_OverBudget(t *testing.T) {
	gate := &BudgetGatekeeper{MaxTokens: 1000, Spent: func() int { return 1500 }}
	ok, reason, err := gate.Pass(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("over budget should reject")
	}
	if !strings.Contains(reason, "budget exceeded") {
		t.Fatalf("reason = %q", reason)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// BudgetGatekeeper tests — BLUE
// ═══════════════════════════════════════════════════════════════════════

func TestBudgetGate_ExactBoundary(t *testing.T) {
	gate := &BudgetGatekeeper{MaxTokens: 1000, Spent: func() int { return 1000 }}
	ok, _, _ := gate.Pass(context.Background(), "test")
	if ok {
		t.Fatal("exact boundary (spent == max) should reject")
	}
}

func TestBudgetGate_NilSpentCallback(t *testing.T) {
	gate := &BudgetGatekeeper{MaxTokens: 1000, Spent: nil}
	ok, _, err := gate.Pass(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("nil Spent should default to 0 (under budget)")
	}
}

package collective

import (
	"context"
	"strings"
	"testing"

	tangle "github.com/dpopsuev/tangle"
)

// ═══════════════════════════════════════════════════════════════════════
// RED: Ambiguous / error cases
// ═══════════════════════════════════════════════════════════════════════

func TestAgentGatekeeper_GibberishDefaultsToPass(t *testing.T) {
	parts := newTestParts()
	ctx := context.Background()

	agent, _ := parts.spawn(ctx, "gate")
	agent.Listen(func(_ string) string { return "I don't understand the question" })

	gate := &AgentGatekeeper{Agent: agent}
	ok, _, err := gate.Pass(ctx, "test content")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if !ok {
		t.Fatal("ambiguous response should default to PASS (fail-open)")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// GREEN: Happy path
// ═══════════════════════════════════════════════════════════════════════

func TestAgentGatekeeper_PassResponse(t *testing.T) {
	parts := newTestParts()
	ctx := context.Background()

	agent, _ := parts.spawn(ctx, "gate")
	agent.Listen(func(_ string) string { return "PASS: looks good" })

	gate := &AgentGatekeeper{Agent: agent}
	ok, reason, err := gate.Pass(ctx, "review this code")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if !ok {
		t.Fatal("PASS response should allow")
	}
	if !strings.Contains(reason, "looks good") {
		t.Fatalf("reason = %q", reason)
	}
}

func TestAgentGatekeeper_RejectResponse(t *testing.T) {
	parts := newTestParts()
	ctx := context.Background()

	agent, _ := parts.spawn(ctx, "gate")
	agent.Listen(func(_ string) string { return "REJECT: destructive request" })

	gate := &AgentGatekeeper{Agent: agent}
	ok, reason, err := gate.Pass(ctx, "delete everything")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if ok {
		t.Fatal("REJECT response should block")
	}
	if !strings.Contains(reason, "destructive") {
		t.Fatalf("reason = %q", reason)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// BLUE: Edge cases
// ═══════════════════════════════════════════════════════════════════════

func TestAgentGatekeeper_CaseInsensitive(t *testing.T) {
	parts := newTestParts()
	ctx := context.Background()

	cases := []struct {
		response string
		wantPass bool
	}{
		{"reject: bad", false},
		{"Reject: bad", false},
		{"REJECT: bad", false},
		{"pass: ok", true},
		{"Pass: ok", true},
		{"PASS: ok", true},
	}

	for _, tc := range cases {
		agent, _ := parts.spawn(ctx, "gate")
		resp := tc.response
		agent.Listen(func(_ string) string { return resp })

		gate := &AgentGatekeeper{Agent: agent}
		ok, _, err := gate.Pass(ctx, "test")
		if err != nil {
			t.Fatalf("err = %v for %q", err, tc.response)
		}
		if ok != tc.wantPass {
			t.Fatalf("response %q: got pass=%v, want %v", tc.response, ok, tc.wantPass)
		}
	}
}

func TestAgentGatekeeper_EmptyResponse(t *testing.T) {
	parts := newTestParts()
	ctx := context.Background()

	agent, _ := parts.spawn(ctx, "gate")
	agent.Listen(func(_ string) string { return "" })

	gate := &AgentGatekeeper{Agent: agent}
	ok, _, _ := gate.Pass(ctx, "test")
	if !ok {
		t.Fatal("empty response should default to PASS")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Gate predicate integration
// ═══════════════════════════════════════════════════════════════════════

func TestGateKeeper_FromGate(t *testing.T) {
	gk := NewGateKeeper(tangle.AlwaysDeny)
	ok, reason, err := gk.Pass(context.Background(), "test")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if ok {
		t.Fatal("AlwaysDeny gate should reject")
	}
	if reason == "" {
		t.Fatal("rejection should include reason")
	}
}

func TestWithParentGates_IngressBlocks(t *testing.T) {
	parts := newTestParts()
	ctx := context.Background()

	agent, _ := parts.spawn(ctx, "worker")
	agent.Listen(func(_ string) string { return "response" })

	c := NewCollective(1, "test", &echoStrategy{}, []tangle.Agent{agent},
		WithParentGates(tangle.AlwaysDeny, nil),
	)

	_, err := c.Perform(ctx, "hello")
	if err == nil {
		t.Fatal("parent ingress gate should have blocked")
	}
}

func TestWithParentGates_ComposesWithOwnIngress(t *testing.T) {
	parts := newTestParts()
	ctx := context.Background()

	agent, _ := parts.spawn(ctx, "worker")
	agent.Listen(func(_ string) string { return "PASS: ok" })

	callCount := 0
	countingGate := tangle.Gate(func(_ context.Context, _ any) (bool, string, error) {
		callCount++
		return true, "", nil
	})

	gateAgent, _ := parts.spawn(ctx, "gate")
	gateAgent.Listen(func(_ string) string { return "PASS: ok" })

	c := NewCollective(1, "test", &echoStrategy{}, []tangle.Agent{agent},
		WithIngress(&AgentGatekeeper{Agent: gateAgent}),
		WithParentGates(countingGate, nil),
	)

	_, err := c.Perform(ctx, "hello")
	if err != nil {
		t.Fatalf("should pass: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("parent gate called %d times, want 1", callCount)
	}
}

func TestWithParentGates_ParentRejectsBeforeChild(t *testing.T) {
	parts := newTestParts()
	ctx := context.Background()

	agent, _ := parts.spawn(ctx, "worker")
	agent.Listen(func(_ string) string { return "response" })

	childCalled := false
	childGate := &passGate{}
	_ = childGate

	c := NewCollective(1, "test", &echoStrategy{}, []tangle.Agent{agent},
		WithIngress(NewGateKeeper(tangle.Gate(func(_ context.Context, _ any) (bool, string, error) {
			childCalled = true
			return true, "", nil
		}))),
		WithParentGates(tangle.AlwaysDeny, nil),
	)

	_, err := c.Perform(ctx, "hello")
	if err == nil {
		t.Fatal("parent gate should have rejected")
	}
	if childCalled {
		t.Fatal("child gate should not have been called after parent rejection")
	}
}

// Compile-time interface checks.
var (
	_ Gatekeeper = (*AgentGatekeeper)(nil)
	_ Gatekeeper = (*GateKeeper)(nil)
	_ Gatekeeper = (*composedGatekeeper)(nil)
)

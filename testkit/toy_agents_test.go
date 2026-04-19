package testkit_test

import (
	"context"
	"testing"
	"time"

	"github.com/dpopsuev/troupe"
	"github.com/dpopsuev/troupe/billing"
	"github.com/dpopsuev/troupe/testkit"
)

var (
	_ troupe.Actor = (*testkit.EchoAgent)(nil)
	_ troupe.Actor = (*testkit.SlowAgent)(nil)
	_ troupe.Actor = (*testkit.FailAgent)(nil)
	_ troupe.Actor = (*testkit.BudgetAgent)(nil)
)

func TestEchoAgent(t *testing.T) {
	a := &testkit.EchoAgent{}
	ctx := context.Background()

	got, err := a.Perform(ctx, "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
	if a.Calls() != 1 {
		t.Errorf("calls = %d, want 1", a.Calls())
	}
	if !a.Ready() {
		t.Error("should be ready")
	}

	_ = a.Kill(ctx)
	if a.Ready() {
		t.Error("should not be ready after kill")
	}
	_, err = a.Perform(ctx, "dead")
	if err == nil {
		t.Error("expected error after kill")
	}
}

func TestSlowAgent_Timeout(t *testing.T) {
	a := &testkit.SlowAgent{Delay: 5 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := a.Perform(ctx, "hello")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if a.Calls() != 0 {
		t.Errorf("calls = %d, want 0", a.Calls())
	}
}

func TestSlowAgent_Completes(t *testing.T) {
	a := &testkit.SlowAgent{Delay: 1 * time.Millisecond}
	ctx := context.Background()

	got, err := a.Perform(ctx, "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestFailAgent_FailsEveryN(t *testing.T) {
	a := &testkit.FailAgent{FailEvery: 3}
	ctx := context.Background()

	for i := 1; i <= 6; i++ {
		_, err := a.Perform(ctx, "test")
		shouldFail := i%3 == 0
		if shouldFail && err == nil {
			t.Errorf("call %d: expected error", i)
		}
		if !shouldFail && err != nil {
			t.Errorf("call %d: unexpected error: %v", i, err)
		}
	}
}

func TestFailAgent_AlwaysFails(t *testing.T) {
	a := &testkit.FailAgent{FailEvery: 1}
	ctx := context.Background()

	for range 3 {
		_, err := a.Perform(ctx, "test")
		if err == nil {
			t.Error("expected error")
		}
	}
}

func TestFailAgent_NeverFails(t *testing.T) {
	a := &testkit.FailAgent{FailEvery: 0}
	ctx := context.Background()

	for range 5 {
		_, err := a.Perform(ctx, "test")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}
}

func TestBudgetAgent_RecordsTokens(t *testing.T) {
	tracker := billing.NewTracker()
	a := &testkit.BudgetAgent{
		TokensPerCall: 100,
		Tracker:       tracker,
		AgentID:       "budget-1",
	}
	ctx := context.Background()

	for range 3 {
		_, err := a.Perform(ctx, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	summary := tracker.Summary()
	if summary.TotalPromptTokens != 300 {
		t.Errorf("prompt tokens = %d, want 300", summary.TotalPromptTokens)
	}
	if summary.TotalArtifactTokens != 300 {
		t.Errorf("artifact tokens = %d, want 300", summary.TotalArtifactTokens)
	}
	node := summary.PerNode["budget-1"]
	if node.Invocations != 3 {
		t.Errorf("invocations = %d, want 3", node.Invocations)
	}
}

func TestBudgetAgent_WithEnforcer(t *testing.T) {
	tracker := billing.NewTracker()
	enforcer := billing.NewBudgetEnforcer(tracker, nil)
	enforcer.SetLimit("budget-1", 0.0001)

	a := &testkit.BudgetAgent{
		TokensPerCall: 500,
		Tracker:       tracker,
		AgentID:       "budget-1",
	}
	ctx := context.Background()

	_, _ = a.Perform(ctx, "burn")

	err := enforcer.Check("budget-1")
	if err == nil {
		t.Error("expected budget exceeded")
	}
}

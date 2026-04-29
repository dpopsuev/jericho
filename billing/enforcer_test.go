package billing_test

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dpopsuev/tangle/billing"
)

func TestBudgetEnforcer_UnlimitedPasses(t *testing.T) {
	tracker := billing.NewTracker()
	enforcer := billing.NewBudgetEnforcer(tracker, nil)

	// No limit set — should always pass.
	if err := enforcer.Check("agent-1"); err != nil {
		t.Fatalf("unlimited check: %v", err)
	}
}

func TestBudgetEnforcer_UnderLimitPasses(t *testing.T) {
	tracker := billing.NewTracker()
	enforcer := billing.NewBudgetEnforcer(tracker, nil)
	enforcer.SetLimit("agent-1", 1.00) // $1 limit

	// No tokens recorded — should pass.
	if err := enforcer.Check("agent-1"); err != nil {
		t.Fatalf("under limit: %v", err)
	}
}

func TestBudgetEnforcer_ExceedsLimit(t *testing.T) {
	tracker := billing.NewTracker()
	enforcer := billing.NewBudgetEnforcer(tracker, nil)
	enforcer.SetLimit("agent-1", 0.0001) // tiny limit

	// Record enough tokens to exceed.
	tracker.Record(&billing.TokenRecord{
		Node:           "agent-1",
		PromptTokens:   100000,
		ArtifactTokens: 100000,
		Timestamp:      time.Now(),
	})

	err := enforcer.Check("agent-1")
	if !errors.Is(err, billing.ErrBudgetExceeded) {
		t.Fatalf("err = %v, want ErrBudgetExceeded", err)
	}
}

func TestBudgetEnforcer_HookCalled(t *testing.T) {
	tracker := billing.NewTracker()
	var hookCalled atomic.Bool
	enforcer := billing.NewBudgetEnforcer(tracker, func(id string, spent, limit float64) {
		hookCalled.Store(true)
	})
	enforcer.SetLimit("agent-1", 0.0001)

	tracker.Record(&billing.TokenRecord{
		Node:           "agent-1",
		PromptTokens:   100000,
		ArtifactTokens: 100000,
		Timestamp:      time.Now(),
	})

	enforcer.Check("agent-1")
	if !hookCalled.Load() {
		t.Fatal("onExceed hook should have been called")
	}
}

func TestBudgetEnforcer_SetLimitZeroRemoves(t *testing.T) {
	tracker := billing.NewTracker()
	enforcer := billing.NewBudgetEnforcer(tracker, nil)

	enforcer.SetLimit("agent-1", 1.00)
	if enforcer.Limit("agent-1") != 1.00 {
		t.Fatal("limit should be set")
	}

	enforcer.SetLimit("agent-1", 0) // remove
	if enforcer.Limit("agent-1") != 0 {
		t.Fatal("limit should be removed")
	}
}

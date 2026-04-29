package broker

import (
	"context"

	"github.com/dpopsuev/tangle"
	"github.com/dpopsuev/tangle/billing"
)

// BudgetHook implements SpawnHook to enforce budget limits before spawning.
type BudgetHook struct {
	enforcer *billing.BudgetEnforcer
}

// NewBudgetHook creates a hook that checks budget before each spawn.
func NewBudgetHook(e *billing.BudgetEnforcer) *BudgetHook {
	return &BudgetHook{enforcer: e}
}

// Name returns the hook identifier.
func (h *BudgetHook) Name() string { return "budget" }

// PreSpawn checks the budget enforcer for the actor's role.
func (h *BudgetHook) PreSpawn(_ context.Context, config troupe.AgentConfig) error {
	return h.enforcer.Check(config.Role)
}

// PostSpawn is a no-op observer.
func (h *BudgetHook) PostSpawn(_ context.Context, _ troupe.AgentConfig, _ troupe.Agent, _ error) {}

var _ SpawnHook = (*BudgetHook)(nil)

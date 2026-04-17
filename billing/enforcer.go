// enforcer.go — budget enforcement for per-agent cost limits.
//
// BudgetEnforcer wraps a Tracker and checks spending against configured
// limits before operations proceed. Returns ErrBudgetExceeded when
// an agent has spent more than its allocation.
package billing

import (
	"context"
	"errors"
	"fmt"
	"sync"

	troupe "github.com/dpopsuev/troupe"
)

// ErrBudgetExceeded is returned when an agent exceeds its cost limit.
var ErrBudgetExceeded = errors.New("budget exceeded")

// BudgetExceededHook is called when an agent exceeds its limit.
type BudgetExceededHook func(agentID string, spent, limit float64)

// BudgetEnforcer checks per-agent spending against configured limits.
type BudgetEnforcer struct {
	tracker  Tracker
	mu       sync.RWMutex
	limits   map[string]float64 // agentID → max cost USD
	onExceed BudgetExceededHook
}

// NewBudgetEnforcer creates an enforcer backed by the given tracker.
func NewBudgetEnforcer(tracker Tracker, onExceed BudgetExceededHook) *BudgetEnforcer {
	return &BudgetEnforcer{
		tracker:  tracker,
		limits:   make(map[string]float64),
		onExceed: onExceed,
	}
}

// SetLimit sets the cost ceiling for an agent. 0 = unlimited.
func (e *BudgetEnforcer) SetLimit(agentID string, maxCostUSD float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if maxCostUSD <= 0 {
		delete(e.limits, agentID)
	} else {
		e.limits[agentID] = maxCostUSD
	}
}

// Limit returns the configured limit for an agent (0 = unlimited).
func (e *BudgetEnforcer) Limit(agentID string) float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.limits[agentID]
}

// Check verifies that an agent hasn't exceeded its budget.
// Returns ErrBudgetExceeded if over limit, nil otherwise.
// Agents without a configured limit always pass.
func (e *BudgetEnforcer) Check(agentID string) error {
	e.mu.RLock()
	limit, hasLimit := e.limits[agentID]
	e.mu.RUnlock()

	if !hasLimit {
		return nil
	}

	summary := e.tracker.Summary()
	spent := 0.0
	if cs, ok := summary.PerNode[agentID]; ok {
		spent = float64(cs.PromptTokens+cs.ArtifactTokens) * 0.000003 // rough estimate
	}

	if spent >= limit {
		if e.onExceed != nil {
			e.onExceed(agentID, spent, limit)
		}
		return fmt.Errorf("%w: agent %s spent $%.4f of $%.4f limit", ErrBudgetExceeded, agentID, spent, limit)
	}
	return nil
}

// AsGate returns a Gate that checks the budget for the given agent.
// The Gate subject is ignored — the agentID is bound at creation time.
func (e *BudgetEnforcer) AsGate(agentID string) troupe.Gate {
	return func(_ context.Context, _ any) (bool, string, error) {
		if err := e.Check(agentID); err != nil {
			return false, err.Error(), nil
		}
		return true, "", nil
	}
}

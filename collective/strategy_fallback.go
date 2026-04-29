// strategy_fallback.go — Fallback: decorator that tries Primary, falls back on failure.
package collective

import (
	"context"
	"errors"

	"github.com/dpopsuev/tangle"
)

// Fallback wraps a Primary strategy with a Fallback. If Primary fails,
// Fallback is tried. Decorator pattern — composes any two strategies.
type Fallback struct {
	Primary  CollectiveStrategy
	Fallback CollectiveStrategy
}

// Select delegates to Primary's Selector.
func (f *Fallback) Select(ctx context.Context, agents []troupe.Agent) []troupe.Agent {
	if sel, ok := f.Primary.(Selector); ok {
		return sel.Select(ctx, agents)
	}
	return agents
}

// Execute tries Primary, falls back on failure.
func (f *Fallback) Execute(ctx context.Context, prompt string, agents []troupe.Agent) (string, error) {
	if exec, ok := f.Primary.(Executor); ok {
		result, err := exec.Execute(ctx, prompt, agents)
		if err == nil {
			return result, nil
		}
	}
	if exec, ok := f.Fallback.(Executor); ok {
		return exec.Execute(ctx, prompt, agents)
	}
	return "", ErrNoExecutor
}

// Orchestrate tries Primary.Orchestrate, falls back on failure.
func (f *Fallback) Orchestrate(ctx context.Context, prompt string, agents []troupe.Agent) (string, error) {
	result, err := f.Primary.Orchestrate(ctx, prompt, agents)
	if err == nil {
		return result, nil
	}
	return f.Fallback.Orchestrate(ctx, prompt, agents)
}

// ErrNoExecutor is returned when neither strategy implements Executor.
var ErrNoExecutor = errors.New("fallback: neither strategy implements Executor")

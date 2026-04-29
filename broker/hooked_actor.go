package broker

import (
	"context"
	"fmt"

	troupe "github.com/dpopsuev/tangle"
)

type hookedActor struct {
	inner troupe.Agent
	hooks []PerformHook
	gate  troupe.Gate
}

func newHookedActor(inner troupe.Agent, hooks []PerformHook, gate troupe.Gate) *hookedActor {
	return &hookedActor{inner: inner, hooks: hooks, gate: gate}
}

func (a *hookedActor) Perform(ctx context.Context, prompt string) (string, error) {
	if a.gate != nil {
		allowed, reason, err := a.gate(ctx, prompt)
		if err != nil {
			return "", fmt.Errorf("perform gate: %w", err)
		}
		if !allowed {
			return "", fmt.Errorf("perform gate rejected: %s", reason)
		}
	}
	for _, h := range a.hooks {
		if err := h.PrePerform(ctx, prompt); err != nil {
			return "", err
		}
	}
	resp, err := a.inner.Perform(ctx, prompt)
	for _, h := range a.hooks {
		h.PostPerform(ctx, prompt, resp, err)
	}
	return resp, err
}

func (a *hookedActor) Ready() bool                    { return a.inner.Ready() }
func (a *hookedActor) Kill(ctx context.Context) error { return a.inner.Kill(ctx) }

var _ troupe.Agent = (*hookedActor)(nil)

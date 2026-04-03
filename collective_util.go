package jericho

import (
	"context"
	"errors"
	"fmt"

	"github.com/dpopsuev/jericho/agent"
	"github.com/dpopsuev/jericho/collective"
)

// SpawnCollective is a convenience function that composes Pick + Spawn + Collective.
// The Broker spawns individual actors, then wraps them in a Collective with the
// given strategy. The returned Actor delegates Perform to the strategy.
func SpawnCollective(ctx context.Context, broker Broker, count int, strategy collective.CollectiveStrategy) (Actor, error) {
	if count <= 0 {
		count = 1
	}

	configs, err := broker.Pick(ctx, Preferences{Count: count})
	if err != nil {
		return nil, fmt.Errorf("spawn collective: pick: %w", err)
	}

	solos := make([]*agent.Solo, 0, len(configs))
	for _, cfg := range configs {
		actor, err := broker.Spawn(ctx, cfg)
		if err != nil {
			// Kill any already spawned.
			for _, s := range solos {
				_ = s.Kill(ctx)
			}
			return nil, fmt.Errorf("spawn collective: spawn: %w", err)
		}
		solo, ok := actor.(*agent.Solo)
		if !ok {
			return nil, fmt.Errorf("%w: got %T", ErrNotSolo, actor)
		}
		solos = append(solos, solo)
	}

	coll := collective.NewCollective(0, "collective", strategy, solos)
	return &collectiveActor{coll: coll}, nil
}

// collectiveActor adapts a Collective to the Actor interface.
type collectiveActor struct {
	coll *collective.Collective
}

func (ca *collectiveActor) Perform(ctx context.Context, prompt string) (string, error) {
	return ca.coll.Perform(ctx, prompt)
}

func (ca *collectiveActor) Ready() bool {
	return ca.coll.IsHealthy()
}

func (ca *collectiveActor) Kill(ctx context.Context) error {
	return ca.coll.Kill(ctx)
}

// ErrNotSolo is returned when an Actor cannot be unwrapped to *agent.Solo.
var ErrNotSolo = errors.New("spawn collective: actor is not *agent.Solo")

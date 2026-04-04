package collective

import (
	"context"
	"fmt"

	"github.com/dpopsuev/troupe"
)

// SpawnCollective is a convenience function that composes Pick + Spawn + Collective.
// The Broker spawns individual actors, then wraps them in a Collective with the
// given strategy. The returned Actor delegates Perform to the strategy.
func SpawnCollective(ctx context.Context, broker troupe.Broker, count int, strategy CollectiveStrategy) (troupe.Actor, error) {
	if count <= 0 {
		count = 1
	}

	configs, err := broker.Pick(ctx, troupe.Preferences{Count: count})
	if err != nil {
		return nil, fmt.Errorf("spawn collective: pick: %w", err)
	}

	actors := make([]troupe.Actor, 0, len(configs))
	for _, cfg := range configs {
		actor, err := broker.Spawn(ctx, cfg)
		if err != nil {
			for _, a := range actors {
				_ = a.Kill(ctx)
			}
			return nil, fmt.Errorf("spawn collective: spawn: %w", err)
		}
		actors = append(actors, actor)
	}

	coll := NewCollective(0, "collective", strategy, actors)
	return coll, nil
}

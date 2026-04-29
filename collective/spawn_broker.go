package collective

import (
	"context"
	"fmt"

	"github.com/dpopsuev/tangle"
)

// SpawnCollective composes Pick + Spawn + Collective.
// The Caster spawns individual actors, then wraps them in a Collective
// with the given strategy.
func SpawnCollective(ctx context.Context, caster troupe.Caster, count int, strategy CollectiveStrategy) (troupe.Agent, error) {
	if count <= 0 {
		count = 1
	}

	configs, err := caster.Pick(ctx, troupe.Preferences{Count: count})
	if err != nil {
		return nil, fmt.Errorf("spawn collective: pick: %w", err)
	}

	actors := make([]troupe.Agent, 0, len(configs))
	for _, cfg := range configs {
		actor, err := caster.Spawn(ctx, cfg)
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

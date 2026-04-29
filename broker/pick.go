package broker

import (
	"context"

	troupe "github.com/dpopsuev/tangle"
)

// PickStrategy selects from candidate ActorConfigs before spawning.
// Pluggable: consumers can implement custom selection logic.
type PickStrategy interface {
	Choose(ctx context.Context, candidates []troupe.AgentConfig, prefs troupe.Preferences) []troupe.AgentConfig
}

// FirstMatch returns the first N candidates. Default strategy, backward compatible.
type FirstMatch struct{}

// Choose returns up to prefs.Count candidates from the front of the list.
func (FirstMatch) Choose(_ context.Context, candidates []troupe.AgentConfig, prefs troupe.Preferences) []troupe.AgentConfig {
	count := prefs.Count
	if count <= 0 {
		count = 1
	}
	if count > len(candidates) {
		count = len(candidates)
	}
	return candidates[:count]
}

// WithPickStrategy sets the actor selection strategy. Default: FirstMatch.
func WithPickStrategy(s PickStrategy) Option {
	return func(c *config) { c.pickStrategy = s }
}

// PickStrategyFrom wraps a Pick[AgentConfig] as a PickStrategy.
func PickStrategyFrom(p troupe.Pick[troupe.AgentConfig]) PickStrategy {
	return &pickAdapter{pick: p}
}

type pickAdapter struct {
	pick troupe.Pick[troupe.AgentConfig]
}

func (a *pickAdapter) Choose(ctx context.Context, candidates []troupe.AgentConfig, _ troupe.Preferences) []troupe.AgentConfig {
	return a.pick(ctx, candidates)
}

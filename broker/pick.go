package broker

import (
	"context"

	tangle "github.com/dpopsuev/tangle"
)

// PickStrategy selects from candidate ActorConfigs before spawning.
// Pluggable: consumers can implement custom selection logic.
type PickStrategy interface {
	Choose(ctx context.Context, candidates []tangle.AgentConfig, prefs tangle.Preferences) []tangle.AgentConfig
}

// FirstMatch returns the first N candidates. Default strategy, backward compatible.
type FirstMatch struct{}

// Choose returns up to prefs.Count candidates from the front of the list.
func (FirstMatch) Choose(_ context.Context, candidates []tangle.AgentConfig, prefs tangle.Preferences) []tangle.AgentConfig {
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
func PickStrategyFrom(p tangle.Pick[tangle.AgentConfig]) PickStrategy {
	return &pickAdapter{pick: p}
}

type pickAdapter struct {
	pick tangle.Pick[tangle.AgentConfig]
}

func (a *pickAdapter) Choose(ctx context.Context, candidates []tangle.AgentConfig, _ tangle.Preferences) []tangle.AgentConfig {
	return a.pick(ctx, candidates)
}

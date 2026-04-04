package troupe

import "context"

// PickStrategy selects from candidate ActorConfigs before spawning.
// Pluggable: consumers can implement custom selection logic.
type PickStrategy interface {
	Choose(ctx context.Context, candidates []ActorConfig, prefs Preferences) []ActorConfig
}

// FirstMatch returns the first N candidates. Default strategy, backward compatible.
type FirstMatch struct{}

// Choose returns up to prefs.Count candidates from the front of the list.
func (FirstMatch) Choose(_ context.Context, candidates []ActorConfig, prefs Preferences) []ActorConfig {
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
func WithPickStrategy(s PickStrategy) BrokerOption {
	return func(c *brokerConfig) { c.pickStrategy = s }
}

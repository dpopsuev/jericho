package troupe

import "context"

// Hook observes and optionally intercepts broker lifecycle events.
type Hook interface {
	// Name returns a human-readable hook identifier for logging.
	Name() string
}

// SpawnHook intercepts Broker.Spawn calls.
type SpawnHook interface {
	Hook
	// PreSpawn is called before spawning. Return non-nil error to reject.
	PreSpawn(ctx context.Context, config ActorConfig) error
	// PostSpawn is called after spawning (success or failure).
	PostSpawn(ctx context.Context, config ActorConfig, actor Actor, err error)
}

// PerformHook intercepts Actor.Perform calls.
type PerformHook interface {
	Hook
	// PrePerform is called before performing. Return non-nil error to reject.
	PrePerform(ctx context.Context, prompt string) error
	// PostPerform is called after performing.
	PostPerform(ctx context.Context, prompt, response string, err error)
}

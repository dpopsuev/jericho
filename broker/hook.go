package broker

import (
	"context"

	troupe "github.com/dpopsuev/tangle"
	"github.com/dpopsuev/tangle/world"
)

// Hook observes and optionally intercepts broker lifecycle events.
type Hook interface {
	// Name returns a human-readable hook identifier for logging.
	Name() string
}

// SpawnHook intercepts Broker.Spawn calls.
type SpawnHook interface {
	Hook
	// PreSpawn is called before spawning. Return non-nil error to reject.
	PreSpawn(ctx context.Context, config troupe.AgentConfig) error
	// PostSpawn is called after spawning (success or failure).
	PostSpawn(ctx context.Context, config troupe.AgentConfig, actor troupe.Agent, err error)
}

// PerformHook intercepts Agent.Perform calls.
type PerformHook interface {
	Hook
	// PrePerform is called before performing. Return non-nil error to reject.
	PrePerform(ctx context.Context, prompt string) error
	// PostPerform is called after performing.
	PostPerform(ctx context.Context, prompt, response string, err error)
}

// KickHook intercepts Admission.Kick calls.
type KickHook interface {
	Hook
	// PreKick is called before kicking. Return non-nil error to block.
	PreKick(ctx context.Context, id world.EntityID) error
	// PostKick is called after kicking.
	PostKick(ctx context.Context, id world.EntityID, err error)
}

// BanHook intercepts Admission.Ban calls.
type BanHook interface {
	Hook
	// PreBan is called before banning. Return non-nil error to block.
	PreBan(ctx context.Context, id world.EntityID, reason string) error
	// PostBan is called after banning.
	PostBan(ctx context.Context, id world.EntityID, reason string, err error)
}

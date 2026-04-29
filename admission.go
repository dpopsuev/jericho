package troupe

import (
	"context"

	"github.com/dpopsuev/tangle/world"
)

// Admission is the single entry point for all agents into the World.
// Both internal spawns (Broker.Spawn) and external registrations
// (A2A, Lobby) go through this interface. This is the trust boundary —
// Gates enforce policy on every Admit call.
type Admission interface {
	// Admit registers an agent into the World. Runs admission gates,
	// creates an ECS entity, attaches components, registers in Transport,
	// and emits to ControlLog. Returns the entity ID.
	Admit(ctx context.Context, config AgentConfig) (world.EntityID, error)

	// Kick forcefully removes an agent from the World. Unregisters from
	// Transport, marks entity terminated, emits to ControlLog.
	Kick(ctx context.Context, id world.EntityID) error

	// Ban kicks an agent and adds its identity to a deny list, preventing
	// re-admission. Returns ErrNotFound if the agent doesn't exist.
	Ban(ctx context.Context, id world.EntityID, reason string) error

	// Unban removes an agent identity from the deny list.
	Unban(ctx context.Context, id world.EntityID) error
}

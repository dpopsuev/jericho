package troupe

import "context"

// Broker is the Actor Broker — it casts and hires actors for Directors.
// Facade over arsenal (Pick), pool+acp (Spawn).
type Broker interface {
	// Pick returns actor configurations matching the given preferences.
	// Backed by the Arsenal catalog.
	Pick(ctx context.Context, prefs Preferences) ([]ActorConfig, error)

	// Spawn creates a running actor from the given configuration.
	Spawn(ctx context.Context, config ActorConfig) (Actor, error)
}

// Preferences describes what kind of actor the Director needs.
type Preferences struct {
	// Role is the functional role (e.g., "investigator", "reviewer").
	Role string `json:"role,omitempty"`

	// Model requests a specific model (e.g., "sonnet", "opus").
	// Empty = let the Broker decide.
	Model string `json:"model,omitempty"`

	// Count is how many actors to pick. Default 1.
	Count int `json:"count,omitempty"`
}

// ActorConfig is the resolved configuration for spawning an actor.
// Returned by Broker.Pick, consumed by Broker.Spawn.
type ActorConfig struct {
	// Model is the resolved model identifier.
	Model string `json:"model"`

	// Provider is the resolved provider (e.g., "anthropic", "openai").
	Provider string `json:"provider,omitempty"`

	// Role is the assigned role.
	Role string `json:"role,omitempty"`
}

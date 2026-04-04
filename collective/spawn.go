// spawn.go — SpawnCollective creates an Collective via Staff.
package collective

import (
	"context"
	"errors"
	"fmt"

	"github.com/dpopsuev/jericho"

	"github.com/dpopsuev/jericho/internal/agent"
	"github.com/dpopsuev/jericho/internal/warden"
	"github.com/dpopsuev/jericho/world"
)

// Sentinel errors for collective spawning.
var (
	ErrTooFewAgents = errors.New("collective requires at least 2 agents")
	ErrNoStrategy   = errors.New("collective requires a strategy")
)

// ErrInitFailed is returned when an init agent fails before main agents start.
var ErrInitFailed = errors.New("collective init agent failed")

// CollectiveConfig configures a new Collective.
type CollectiveConfig struct {
	Role       string               // collective's external role name
	Strategy   CollectiveStrategy   // how agents collaborate
	Agents     []warden.AgentConfig // one config per internal agent
	Shade      string               // shade family for all agents (empty = random)
	Init       []warden.AgentConfig // init agents — run to completion before main agents start
	InitPrompt string               // prompt to send to each init agent (empty = skip Ask)
	Ingress    *warden.AgentConfig  // optional ingress gate agent (bouncer)
	Egress     *warden.AgentConfig  // optional egress gate agent (reviewer)
}

// SpawnCollective creates an Collective by spawning N agents via Staff.
// Returns a collective that implements agent.Agent — operators can't tell
// it's not a single agent.
func SpawnCollectiveFromStaff(ctx context.Context, staff *agent.Staff, cfg CollectiveConfig) (*Collective, error) {
	if len(cfg.Agents) < 2 {
		return nil, fmt.Errorf("%w, got %d", ErrTooFewAgents, len(cfg.Agents))
	}
	if cfg.Strategy == nil {
		return nil, ErrNoStrategy
	}

	// Phase: Pending — run init agents to completion first.
	for i := range cfg.Init {
		initAgent, err := staff.Spawn(ctx, cfg.Init[i].Role, cfg.Init[i])
		if err != nil {
			return nil, fmt.Errorf("%w: spawn init agent %q: %w", ErrInitFailed, cfg.Init[i].Role, err)
		}
		if cfg.InitPrompt != "" {
			if _, err := initAgent.Perform(ctx, cfg.InitPrompt); err != nil {
				initAgent.Kill(ctx) //nolint:errcheck // cleanup
				return nil, fmt.Errorf("%w: init agent %q: %w", ErrInitFailed, cfg.Init[i].Role, err)
			}
		}
		initAgent.Kill(ctx) //nolint:errcheck // init agents run to completion
	}

	// Phase: Spawning main agents.
	agents := make([]jericho.Actor, 0, len(cfg.Agents))
	for i := range cfg.Agents {
		a, err := staff.Spawn(ctx, cfg.Agents[i].Role, cfg.Agents[i])
		if err != nil {
			// Kill any already-spawned agents on failure.
			for _, spawned := range agents {
				spawned.Kill(ctx) //nolint:errcheck // best-effort cleanup on spawn failure
			}
			return nil, fmt.Errorf("spawn agent for collective %q: %w", cfg.Role, err)
		}
		agents = append(agents, a)
	}

	// Spawn gate agents if configured.
	var opts []CollectiveOption
	if cfg.Ingress != nil {
		gateAgent, err := staff.Spawn(ctx, "ingress", *cfg.Ingress)
		if err != nil {
			for _, a := range agents {
				a.Kill(ctx) //nolint:errcheck // best-effort cleanup on gate spawn failure
			}
			return nil, fmt.Errorf("spawn ingress gate for %q: %w", cfg.Role, err)
		}
		opts = append(opts, WithIngress(&AgentGatekeeper{Agent: gateAgent}))
	}
	if cfg.Egress != nil {
		gateAgent, err := staff.Spawn(ctx, "egress", *cfg.Egress)
		if err != nil {
			for _, a := range agents {
				a.Kill(ctx) //nolint:errcheck // best-effort cleanup on gate spawn failure
			}
			return nil, fmt.Errorf("spawn egress gate for %q: %w", cfg.Role, err)
		}
		opts = append(opts, WithEgress(&AgentGatekeeper{Agent: gateAgent}))
	}

	id := world.EntityID(0) // collective has no entity ID
	return NewCollective(id, cfg.Role, cfg.Strategy, agents, opts...), nil
}

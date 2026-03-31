// spawn.go — SpawnCollective creates an AgentCollective via Staff.
package collective

import (
	"context"
	"errors"
	"fmt"

	"github.com/dpopsuev/jericho/facade"
	"github.com/dpopsuev/jericho/pool"
)

// Sentinel errors for collective spawning.
var (
	ErrTooFewAgents = errors.New("collective requires at least 2 agents")
	ErrNoStrategy   = errors.New("collective requires a strategy")
)

// CollectiveConfig configures a new AgentCollective.
type CollectiveConfig struct {
	Role     string              // collective's external role name
	Strategy CollectiveStrategy  // how agents collaborate
	Agents   []pool.LaunchConfig // one config per internal agent
	Ingress  *pool.LaunchConfig  // optional ingress gate agent (bouncer)
	Egress   *pool.LaunchConfig  // optional egress gate agent (reviewer)
}

// SpawnCollective creates an AgentCollective by spawning N agents via Staff.
// Returns a collective that implements facade.Agent — operators can't tell
// it's not a single agent.
func SpawnCollective(ctx context.Context, staff *facade.Staff, cfg CollectiveConfig) (*AgentCollective, error) {
	if len(cfg.Agents) < 2 {
		return nil, fmt.Errorf("%w, got %d", ErrTooFewAgents, len(cfg.Agents))
	}
	if cfg.Strategy == nil {
		return nil, ErrNoStrategy
	}

	agents := make([]*facade.AgentHandle, 0, len(cfg.Agents))
	for _, acfg := range cfg.Agents {
		a, err := staff.Spawn(ctx, acfg.Role, acfg)
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
		opts = append(opts, WithIngress(&AgentGate{Agent: gateAgent}))
	}
	if cfg.Egress != nil {
		gateAgent, err := staff.Spawn(ctx, "egress", *cfg.Egress)
		if err != nil {
			for _, a := range agents {
				a.Kill(ctx) //nolint:errcheck // best-effort cleanup on gate spawn failure
			}
			return nil, fmt.Errorf("spawn egress gate for %q: %w", cfg.Role, err)
		}
		opts = append(opts, WithEgress(&AgentGate{Agent: gateAgent}))
	}

	id := agents[0].ID()
	return NewAgentCollective(id, cfg.Role, cfg.Strategy, agents, opts...), nil
}

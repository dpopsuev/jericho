// spawn.go — SpawnCollective creates an AgentCollective via Staff.
package collective

import (
	"context"
	"fmt"

	"github.com/dpopsuev/bugle/facade"
	"github.com/dpopsuev/bugle/pool"
)

// CollectiveConfig configures a new AgentCollective.
type CollectiveConfig struct {
	Role     string              // collective's external role name
	Strategy CollectiveStrategy  // how agents collaborate
	Agents   []pool.LaunchConfig // one config per internal agent
}

// SpawnCollective creates an AgentCollective by spawning N agents via Staff.
// Returns a collective that implements facade.Agent — operators can't tell
// it's not a single agent.
func SpawnCollective(ctx context.Context, staff *facade.Staff, cfg CollectiveConfig) (*AgentCollective, error) {
	if len(cfg.Agents) < 2 {
		return nil, fmt.Errorf("collective requires at least 2 agents, got %d", len(cfg.Agents))
	}
	if cfg.Strategy == nil {
		return nil, fmt.Errorf("collective requires a strategy")
	}

	var agents []*facade.AgentHandle
	for _, acfg := range cfg.Agents {
		a, err := staff.Spawn(ctx, acfg.Role, acfg)
		if err != nil {
			// Kill any already-spawned agents on failure.
			for _, spawned := range agents {
				spawned.Kill(ctx) //nolint:errcheck
			}
			return nil, fmt.Errorf("spawn agent for collective %q: %w", cfg.Role, err)
		}
		agents = append(agents, a)
	}

	// Use the first agent's ID as the collective's ID (for identification).
	id := agents[0].ID()
	return NewAgentCollective(id, cfg.Role, cfg.Strategy, agents), nil
}

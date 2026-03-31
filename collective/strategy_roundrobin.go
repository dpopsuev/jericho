// strategy_roundrobin.go — RoundRobin: stateless load distribution with health filtering.
package collective

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/dpopsuev/jericho/agent"
)

// ErrNoHealthyAgents is returned when all agents are not ready.
var ErrNoHealthyAgents = errors.New("roundrobin: no healthy agents available")

// RoundRobin picks one healthy agent per request via atomic round-robin index.
// Agents where IsReady() returns false are skipped.
type RoundRobin struct {
	idx atomic.Uint64
}

// Orchestrate picks the next healthy agent and forwards the prompt.
func (r *RoundRobin) Orchestrate(ctx context.Context, prompt string, agents []*agent.Solo) (string, error) {
	if len(agents) == 0 {
		return "", fmt.Errorf("%w, got 0 agents", ErrNoHealthyAgents)
	}

	start := r.idx.Add(1) - 1
	n := uint64(len(agents))

	// Scan all agents starting from current index, skip unhealthy.
	for i := range uint64(len(agents)) {
		candidate := agents[(start+i)%n]
		if candidate.IsReady() {
			return candidate.Ask(ctx, prompt)
		}
	}

	return "", ErrNoHealthyAgents
}

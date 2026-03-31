// strategy_race.go — Race: all agents compete, first response wins.
package collective

import (
	"context"
	"fmt"

	"github.com/dpopsuev/jericho/agent"
)

// Race fans out to all agents concurrently. The first successful response
// wins. Remaining agents are canceled via context.
type Race struct{}

// Orchestrate sends the prompt to all agents in parallel. Returns the first
// response received. If all agents fail, returns the last error.
func (Race) Orchestrate(ctx context.Context, prompt string, agents []*agent.Solo) (string, error) {
	if len(agents) == 0 {
		return "", ErrNoAgents
	}

	type result struct {
		resp string
		err  error
	}

	raceCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := make(chan result, len(agents))

	for _, a := range agents {
		go func(ag *agent.Solo) {
			resp, err := ag.Ask(raceCtx, prompt)
			ch <- result{resp, err}
		}(a)
	}

	var lastErr error
	for range agents {
		r := <-ch
		if r.err == nil {
			cancel() // cancel remaining
			return r.resp, nil
		}
		lastErr = r.err
	}

	return "", fmt.Errorf("race: all %d agents failed, last: %w", len(agents), lastErr)
}

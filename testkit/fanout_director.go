package testkit

import (
	"context"
	"sync"
	"time"

	"github.com/dpopsuev/tangle"
)

// FanOutDirector sends one prompt to multiple actors concurrently.
// Validates concurrent Agent.Perform + event streaming.
type FanOutDirector struct {
	Prompt string
	Count  int // how many actors to fan out to
}

func (d *FanOutDirector) Direct(ctx context.Context, caster troupe.Caster) (<-chan troupe.Event, error) {
	count := d.Count
	if count <= 0 {
		count = 1
	}

	configs, err := caster.Pick(ctx, troupe.Preferences{Count: count})
	if err != nil {
		return nil, err
	}

	actors := make([]troupe.Agent, len(configs))
	for i, cfg := range configs {
		a, err := caster.Spawn(ctx, cfg)
		if err != nil {
			return nil, err
		}
		actors[i] = a
	}

	ch := make(chan troupe.Event, len(actors)*2+1)

	go func() {
		defer close(ch)

		var wg sync.WaitGroup
		for i, actor := range actors {
			wg.Add(1)
			go func(a troupe.Agent, cfg troupe.AgentConfig) {
				defer wg.Done()

				ch <- troupe.Event{Kind: troupe.Started, Step: d.Prompt, Agent: cfg.Role}

				start := time.Now()
				_, err := a.Perform(ctx, d.Prompt)
				elapsed := time.Since(start)

				if err != nil {
					ch <- troupe.Event{Kind: troupe.Failed, Step: d.Prompt, Agent: cfg.Role, Error: err, Elapsed: elapsed}
					return
				}

				ch <- troupe.Event{Kind: troupe.Completed, Step: d.Prompt, Agent: cfg.Role, Elapsed: elapsed}
			}(actor, configs[i])
		}

		wg.Wait()
		ch <- troupe.Event{Kind: troupe.Done}
	}()

	return ch, nil
}

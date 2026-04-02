package testkit

import (
	"context"
	"sync"
	"time"

	"github.com/dpopsuev/jericho"
)

// FanOutDirector sends one prompt to multiple actors concurrently.
// Validates concurrent Actor.Perform + event streaming.
type FanOutDirector struct {
	Prompt string
	Count  int // how many actors to fan out to
}

func (d *FanOutDirector) Direct(ctx context.Context, broker jericho.Broker) (<-chan jericho.Event, error) {
	count := d.Count
	if count <= 0 {
		count = 1
	}

	configs, err := broker.Pick(ctx, jericho.Preferences{Count: count})
	if err != nil {
		return nil, err
	}

	actors := make([]jericho.Actor, len(configs))
	for i, cfg := range configs {
		a, err := broker.Spawn(ctx, cfg)
		if err != nil {
			return nil, err
		}
		actors[i] = a
	}

	ch := make(chan jericho.Event, len(actors)*2+1)

	go func() {
		defer close(ch)

		var wg sync.WaitGroup
		for i, actor := range actors {
			wg.Add(1)
			go func(a jericho.Actor, cfg jericho.ActorConfig) {
				defer wg.Done()

				ch <- jericho.Event{Kind: jericho.Started, Step: d.Prompt, Agent: cfg.Role}

				start := time.Now()
				_, err := a.Perform(ctx, d.Prompt)
				elapsed := time.Since(start)

				if err != nil {
					ch <- jericho.Event{Kind: jericho.Failed, Step: d.Prompt, Agent: cfg.Role, Error: err, Elapsed: elapsed}
					return
				}

				ch <- jericho.Event{Kind: jericho.Completed, Step: d.Prompt, Agent: cfg.Role, Elapsed: elapsed}
			}(actor, configs[i])
		}

		wg.Wait()
		ch <- jericho.Event{Kind: jericho.Done}
	}()

	return ch, nil
}

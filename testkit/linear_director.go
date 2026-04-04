package testkit

import (
	"context"
	"time"

	"github.com/dpopsuev/troupe"
)

// Step is a named unit of work for mock Directors.
type Step struct {
	Name   string
	Prompt string
}

// LinearDirector executes steps sequentially with one actor.
// The simplest possible Director — validates the contract.
type LinearDirector struct {
	Steps []Step
}

func (d *LinearDirector) Direct(ctx context.Context, broker troupe.Broker) (<-chan troupe.Event, error) {
	configs, err := broker.Pick(ctx, troupe.Preferences{Count: 1})
	if err != nil {
		return nil, err
	}
	if len(configs) == 0 {
		return nil, ErrNoActors
	}

	actor, err := broker.Spawn(ctx, configs[0])
	if err != nil {
		return nil, err
	}

	ch := make(chan troupe.Event, len(d.Steps)*2+1)

	go func() {
		defer close(ch)

		for _, step := range d.Steps {
			if ctx.Err() != nil {
				ch <- troupe.Event{Kind: troupe.Failed, Error: ctx.Err()}
				return
			}

			ch <- troupe.Event{Kind: troupe.Started, Step: step.Name, Agent: configs[0].Role}

			start := time.Now()
			_, err := actor.Perform(ctx, step.Prompt)
			elapsed := time.Since(start)

			if err != nil {
				ch <- troupe.Event{Kind: troupe.Failed, Step: step.Name, Agent: configs[0].Role, Error: err, Elapsed: elapsed}
				return
			}

			ch <- troupe.Event{Kind: troupe.Completed, Step: step.Name, Agent: configs[0].Role, Elapsed: elapsed}
		}

		ch <- troupe.Event{Kind: troupe.Done}
	}()

	return ch, nil
}

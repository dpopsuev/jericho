package execution

import (
	"context"
	"sync"
)

// ActorFunc is the work execution function. Takes input, returns output.
// This is what each pooled actor calls for every work item.
type ActorFunc func(ctx context.Context, input string) (string, error)

// Pool manages N warm actors pulling work from a Queue.
// Actors spawn once at Start(), stay alive pulling work, and drain
// gracefully when the context is cancelled or Drain() is called.
type Pool struct {
	queue Queue
	actor ActorFunc
	size  int
	wg    sync.WaitGroup
}

// NewPool creates an actor pool of the given size.
func NewPool(queue Queue, actor ActorFunc, size int) *Pool {
	return &Pool{queue: queue, actor: actor, size: size}
}

// Start launches N goroutines, each running the pull-execute-submit loop.
// Returns immediately. Call Drain() or cancel the context to stop.
func (p *Pool) Start(ctx context.Context) {
	for i := range p.size {
		p.wg.Add(1)
		go p.workerLoop(ctx, i)
	}
}

// Drain waits for all actors to finish their current work item and exit.
func (p *Pool) Drain() {
	p.wg.Wait()
}

func (p *Pool) workerLoop(ctx context.Context, _ int) {
	defer p.wg.Done()
	for {
		item, err := p.queue.Pull(ctx)
		if err != nil {
			return // context cancelled or queue closed
		}

		result, execErr := p.actor(ctx, item.Input())
		if execErr != nil {
			// Submit error as empty result — the queue routes it back.
			_ = p.queue.Submit(ctx, item.ID(), []byte("error: "+execErr.Error()))
			continue
		}

		_ = p.queue.Submit(ctx, item.ID(), []byte(result))
	}
}

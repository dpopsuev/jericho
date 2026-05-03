package providers

import (
	"context"
	"log/slog"
	troupe "github.com/dpopsuev/tangle"
	"sync"
	"time"
)

// Pool manages N warm actors pulling work from a Queue.
// Actors spawn once at Start(), stay alive pulling work, and drain
// gracefully when the context is cancelled or Drain() is called.
type Pool struct {
	queue Queue
	actor troupe.CompleteFunc
	size  int
	wg    sync.WaitGroup
}

// NewPool creates an actor pool of the given size.
func NewPool(queue Queue, actor troupe.CompleteFunc, size int) *Pool {
	return &Pool{queue: queue, actor: actor, size: size}
}

// Start launches N goroutines, each running the pull-execute-submit loop.
// Returns immediately. Call Drain() or cancel the context to stop.
func (p *Pool) Start(ctx context.Context) {
	slog.InfoContext(ctx, "pool starting", slog.Int("size", p.size))
	for i := range p.size {
		p.wg.Add(1)
		go p.workerLoop(ctx, i)
	}
}

// Drain waits for all actors to finish their current work item and exit.
func (p *Pool) Drain() {
	p.wg.Wait()
}

func (p *Pool) workerLoop(ctx context.Context, workerID int) {
	defer p.wg.Done()
	slog.DebugContext(ctx, "worker started", slog.Int(logKeyWorkerID, workerID))
	for {
		item, err := p.queue.Pull(ctx)
		if err != nil {
			slog.DebugContext(ctx, "worker stopped", slog.Int(logKeyWorkerID, workerID))
			return
		}

		start := time.Now()
		completion, execErr := p.actor(ctx, troupe.CompletionParams{Prompt: item.Input()})
		elapsed := time.Since(start)

		if execErr != nil {
			slog.ErrorContext(ctx, "worker exec failed",
				slog.Int(logKeyWorkerID, workerID),
				slog.Uint64(logKeyItemID, item.ID()),
				slog.Int64(logKeyElapsedMs, elapsed.Milliseconds()),
				slog.Any(logKeyError, execErr))
			_ = p.queue.Submit(ctx, item.ID(), []byte("error: "+execErr.Error()))
			continue
		}

		slog.DebugContext(ctx, "worker exec complete",
			slog.Int(logKeyWorkerID, workerID),
			slog.Uint64(logKeyItemID, item.ID()),
			slog.Int64(logKeyElapsedMs, elapsed.Milliseconds()))
		_ = p.queue.Submit(ctx, item.ID(), []byte(completion.Content))
	}
}

// Package execution provides work distribution strategies for actors.
// Two strategies: inline (caller drives) and queue (worker drives).
package execution

import (
	"context"
	"time"
)

// WorkItem is an opaque unit of work to be processed by an actor.
type WorkItem interface {
	// ID returns the unique identifier for this work item.
	ID() uint64
	// Input returns the work payload (prompt, task description, etc.).
	Input() string
	// Timeout returns the deadline for processing this item.
	// Zero means no timeout.
	Timeout() time.Duration
}

// WorkerHints expresses worker preferences for work matching.
// The queue uses these for locality-aware work stealing.
type WorkerHints struct {
	// PreferredTag is the primary affinity key (e.g., case ID, zone).
	PreferredTag string
	// Stickiness controls how aggressively the worker steals work.
	// 0 = steal immediately if no match, 3 = exclusive (only matching work).
	Stickiness int
	// ConsecutiveMisses tracks how many pulls returned no matching work.
	// The queue may relax stickiness after repeated misses.
	ConsecutiveMisses int
}

// Queue is a work distribution channel between producers and consumers.
// Producers enqueue work items. Consumers pull with optional preferences.
// Results flow back to the producer via Submit.
type Queue interface {
	// Enqueue adds a work item to the queue. Returns when the item is
	// accepted (not when it's processed). Blocks if the queue is full.
	Enqueue(ctx context.Context, item WorkItem) error

	// Pull returns the next available work item, blocking until one is
	// available or the context is cancelled.
	Pull(ctx context.Context) (WorkItem, error)

	// PullWithHints returns work matching the hints if available,
	// falling back to any available work based on stickiness level.
	PullWithHints(ctx context.Context, hints WorkerHints) (WorkItem, error)

	// Submit delivers the result for a work item back to the producer.
	// The id must match a previously pulled WorkItem.ID().
	Submit(ctx context.Context, id uint64, result []byte) error

	// ActiveCount returns the number of work items currently being processed
	// (pulled but not yet submitted).
	ActiveCount() int
}

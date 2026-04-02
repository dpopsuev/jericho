package jericho

import (
	"context"
	"time"
)

// Director is the consumer contract for orchestration strategies.
// Origami implements CircuitDirector. Djinn implements LocalDirector.
// Directors compose: an outer Director can wrap an inner Director.
type Director interface {
	// Direct executes the orchestration plan using actors from the Broker.
	// Returns a channel of Events that streams progress until completion.
	// The channel is closed when the Director is done.
	Direct(ctx context.Context, broker Broker) (<-chan Event, error)
}

// EventKind classifies what happened. String for OCP — consumers can
// define domain-specific kinds without modifying Jericho.
type EventKind string

const (
	// Started indicates work began on a step.
	Started EventKind = "started"
	// Completed indicates a step finished successfully.
	Completed EventKind = "completed"
	// Failed indicates a step failed.
	Failed EventKind = "failed"
	// Transition indicates movement between steps.
	Transition EventKind = "transition"
	// Done indicates all work is complete.
	Done EventKind = "done"
)

// Event is the universal output of a Director. Struct returned (Go idiom).
// Universal metadata in struct fields, domain-specific data in Detail.
type Event struct {
	// Kind classifies the event.
	Kind EventKind
	// Step identifies which step/node/stage this event relates to.
	Step string
	// Agent identifies which actor handled this step.
	Agent string
	// Error is set when Kind is Failed.
	Error error
	// Elapsed is the duration of the step (set on Completed/Failed).
	Elapsed time.Duration
	// Detail carries domain-specific data. Nil when not applicable.
	// Origami sets CircuitDetail, Djinn sets StageDetail.
	// Implements fmt.Stringer for logging.
	Detail EventDetail
}

// EventDetail is the extension point for domain-specific event data.
// Accept interfaces (Go idiom). Each Director defines its own concrete type.
// Recursive: a Detail can wrap an inner Detail (protocol layer pattern).
type EventDetail interface {
	String() string
}

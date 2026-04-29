package testkit

import (
	"sync"
	"time"

	"github.com/dpopsuev/tangle/signal"
)

var _ signal.EventLog = (*StubEventLog)(nil)

// StubEventLog is an in-memory EventLog for testing.
type StubEventLog struct {
	mu     sync.Mutex
	events []signal.Event
	hooks  []func(signal.Event)
}

// NewStubEventLog creates an empty event log.
func NewStubEventLog() *StubEventLog {
	return &StubEventLog{}
}

func (l *StubEventLog) Emit(e signal.Event) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	idx := len(l.events)
	l.events = append(l.events, e)
	for _, fn := range l.hooks {
		fn(e)
	}
	return idx
}

func (l *StubEventLog) Since(index int) []signal.Event {
	l.mu.Lock()
	defer l.mu.Unlock()

	if index < 0 {
		index = 0
	}
	if index >= len(l.events) {
		return nil
	}
	out := make([]signal.Event, len(l.events)-index)
	copy(out, l.events[index:])
	return out
}

func (l *StubEventLog) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.events)
}

func (l *StubEventLog) OnEmit(fn func(signal.Event)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hooks = append(l.hooks, fn)
}

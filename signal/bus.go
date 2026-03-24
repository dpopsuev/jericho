package signal

import (
	"sync"
	"time"
)

// Bus is the interface for an append-only signal log used for agent coordination.
type Bus interface {
	// Emit appends a signal to the bus and returns its index.
	Emit(s *Signal) int
	// Since returns a copy of signals from index onward.
	Since(index int) []Signal
	// Len returns the number of signals in the bus.
	Len() int
	// OnEmit registers a callback that fires on every Emit.
	OnEmit(fn func(Signal))
}

// MemBus is a thread-safe, append-only in-memory signal log for agent coordination.
type MemBus struct {
	mu      sync.Mutex
	signals []Signal
	onEmit  []func(Signal)
}

// NewMemBus returns a new MemBus.
func NewMemBus() *MemBus {
	return &MemBus{}
}

// OnEmit registers a callback that fires on every Emit.
// The callback runs under the bus lock -- keep it fast
// (buffered write, not I/O).
func (b *MemBus) OnEmit(fn func(Signal)) {
	b.mu.Lock()
	b.onEmit = append(b.onEmit, fn)
	b.mu.Unlock()
}

// Emit appends a signal to the bus and returns its 0-based index.
// If the signal has an empty Timestamp, one is auto-set to the
// current UTC time in RFC3339 format.
func (b *MemBus) Emit(s *Signal) int {
	b.mu.Lock()
	defer b.mu.Unlock()

	if s.Timestamp == "" {
		s.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	idx := len(b.signals)
	b.signals = append(b.signals, *s)
	for _, fn := range b.onEmit {
		fn(*s)
	}
	return idx
}

// Since returns a copy of signals from index idx onward.
// If idx is negative it is clamped to 0.
// If idx >= len(signals), returns nil.
func (b *MemBus) Since(idx int) []Signal {
	b.mu.Lock()
	defer b.mu.Unlock()

	if idx < 0 {
		idx = 0
	}
	if idx >= len(b.signals) {
		return nil
	}
	out := make([]Signal, len(b.signals)-idx)
	copy(out, b.signals[idx:])
	return out
}

// Len returns the number of signals in the bus.
func (b *MemBus) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.signals)
}

package broker

import (
	"sync"

	troupe "github.com/dpopsuev/tangle"
)

// InMemoryMeter is a thread-safe in-memory Meter implementation.
type InMemoryMeter struct {
	mu      sync.Mutex
	records []troupe.Usage
}

// NewInMemoryMeter creates an empty in-memory meter.
func NewInMemoryMeter() *InMemoryMeter {
	return &InMemoryMeter{}
}

// Record appends a usage entry (thread-safe).
func (m *InMemoryMeter) Record(u troupe.Usage) {
	m.mu.Lock()
	m.records = append(m.records, u)
	m.mu.Unlock()
}

// Query returns all usage entries for the given actor (thread-safe).
func (m *InMemoryMeter) Query(actor string) []troupe.Usage {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []troupe.Usage
	for _, r := range m.records {
		if r.Agent == actor {
			result = append(result, r)
		}
	}
	return result
}

var _ troupe.Meter = (*InMemoryMeter)(nil)

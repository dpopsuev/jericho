package troupe

import (
	"sync"
	"time"
)

// Meter records and queries resource usage across actors.
type Meter interface {
	// Record appends a usage entry.
	Record(u Usage)
	// Query returns all usage entries for the given actor.
	Query(actor string) []Usage
}

// Usage records resource consumption for a single operation.
type Usage struct {
	Actor    string
	Step     string
	Duration time.Duration
	Detail   UsageDetail
}

// UsageDetail is the extension point for provider-specific usage data.
// Tokens for cloud, GPU-seconds for on-prem. Same pattern as EventDetail.
type UsageDetail interface {
	String() string
}

// InMemoryMeter is a thread-safe in-memory Meter implementation.
type InMemoryMeter struct {
	mu      sync.Mutex
	records []Usage
}

// NewInMemoryMeter creates an empty in-memory meter.
func NewInMemoryMeter() *InMemoryMeter {
	return &InMemoryMeter{}
}

// Record appends a usage entry (thread-safe).
func (m *InMemoryMeter) Record(u Usage) {
	m.mu.Lock()
	m.records = append(m.records, u)
	m.mu.Unlock()
}

// Query returns all usage entries for the given actor (thread-safe).
func (m *InMemoryMeter) Query(actor string) []Usage {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []Usage
	for _, r := range m.records {
		if r.Actor == actor {
			result = append(result, r)
		}
	}
	return result
}

var _ Meter = (*InMemoryMeter)(nil)

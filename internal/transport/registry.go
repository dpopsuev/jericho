// registry.go — AgentLookup for agent discovery with metadata and health tracking.
//
// Extends RoleRegistry with metadata storage, heartbeat tracking, and
// stale entry eviction. InMemoryRegistry is the Day 1 implementation;
// pluggable backends (etcd, consul, DNS) can implement the interface.
package transport

import (
	"sync"
	"time"
)

// LookupEntry describes a registered agent with metadata and health.
type LookupEntry struct {
	ID       string
	Role     string
	Meta     map[string]string
	LastSeen time.Time
	Healthy  bool
}

// AgentLookup is the interface for agent discovery.
type AgentLookup interface {
	Register(id, role string, meta map[string]string) error
	Unregister(id string) error
	Discover(role string) []LookupEntry
	Heartbeat(id string) error
	All() []LookupEntry
}

// InMemoryRegistry implements AgentLookup with stale eviction.
type InMemoryRegistry struct {
	mu       sync.RWMutex
	entries  map[string]*LookupEntry
	staleTTL time.Duration // entries older than this are considered stale
}

// NewInMemoryRegistry creates a registry with the given stale TTL.
// Entries not heartbeated within TTL are marked unhealthy.
func NewInMemoryRegistry(staleTTL time.Duration) *InMemoryRegistry {
	if staleTTL <= 0 {
		staleTTL = 30 * time.Second
	}
	return &InMemoryRegistry{
		entries:  make(map[string]*LookupEntry),
		staleTTL: staleTTL,
	}
}

func (r *InMemoryRegistry) Register(id, role string, meta map[string]string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.entries[id] = &LookupEntry{
		ID:       id,
		Role:     role,
		Meta:     meta,
		LastSeen: time.Now(),
		Healthy:  true,
	}
	return nil
}

func (r *InMemoryRegistry) Unregister(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.entries, id)
	return nil
}

func (r *InMemoryRegistry) Discover(role string) []LookupEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]LookupEntry, 0, len(r.entries))
	now := time.Now()
	for _, e := range r.entries {
		if e.Role != role {
			continue
		}
		entry := *e
		if now.Sub(e.LastSeen) > r.staleTTL {
			entry.Healthy = false
		}
		result = append(result, entry)
	}
	return result
}

func (r *InMemoryRegistry) Heartbeat(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	e, ok := r.entries[id]
	if !ok {
		return nil
	}
	e.LastSeen = time.Now()
	e.Healthy = true
	return nil
}

func (r *InMemoryRegistry) All() []LookupEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now()
	result := make([]LookupEntry, 0, len(r.entries))
	for _, e := range r.entries {
		entry := *e
		if now.Sub(e.LastSeen) > r.staleTTL {
			entry.Healthy = false
		}
		result = append(result, entry)
	}
	return result
}

// EvictStale removes entries that haven't heartbeated within TTL.
// Returns the number of entries evicted.
func (r *InMemoryRegistry) EvictStale() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	evicted := 0
	for id, e := range r.entries {
		if now.Sub(e.LastSeen) > r.staleTTL {
			delete(r.entries, id)
			evicted++
		}
	}
	return evicted
}

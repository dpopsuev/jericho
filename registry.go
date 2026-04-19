package troupe

import "time"

// ServiceEntry describes a registered service in the registry.
type ServiceEntry struct {
	ID      string            `json:"id"`
	Role    string            `json:"role"`
	Meta    map[string]string `json:"meta,omitempty"`
	Healthy bool              `json:"healthy"`
}

// ServiceRegistry tracks agent/service registrations with health.
// InMemoryRegistry in internal/transport/ already implements this
// pattern. This is the public interface for daemon consumers.
type ServiceRegistry interface {
	Register(id, role string, meta map[string]string)
	Unregister(id string)
	Discover(role string) []ServiceEntry
	Heartbeat(id string)
	EvictStale(ttl time.Duration) int
}

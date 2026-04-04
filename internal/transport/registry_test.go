package transport

import (
	"testing"
	"time"
)

func TestInMemoryRegistry_RegisterDiscover(t *testing.T) {
	reg := NewInMemoryRegistry(1 * time.Hour)

	reg.Register("a1", "executor", map[string]string{"model": "claude"})
	reg.Register("a2", "executor", map[string]string{"model": "gemini"})
	reg.Register("a3", "inspector", nil)

	executors := reg.Discover("executor")
	if len(executors) != 2 {
		t.Fatalf("executors = %d, want 2", len(executors))
	}

	inspectors := reg.Discover("inspector")
	if len(inspectors) != 1 {
		t.Fatalf("inspectors = %d, want 1", len(inspectors))
	}

	// Verify metadata.
	for _, e := range executors {
		if e.Meta == nil {
			t.Fatal("executor meta should not be nil")
		}
	}
}

func TestInMemoryRegistry_Unregister(t *testing.T) {
	reg := NewInMemoryRegistry(1 * time.Hour)
	reg.Register("a1", "executor", nil)
	reg.Unregister("a1")

	if len(reg.Discover("executor")) != 0 {
		t.Fatal("should be empty after unregister")
	}
}

func TestInMemoryRegistry_Heartbeat(t *testing.T) {
	reg := NewInMemoryRegistry(50 * time.Millisecond)
	reg.Register("a1", "executor", nil)

	// Wait for staleness.
	time.Sleep(100 * time.Millisecond)

	entries := reg.Discover("executor")
	if len(entries) != 1 || entries[0].Healthy {
		t.Fatal("should be unhealthy after TTL")
	}

	// Heartbeat revives.
	reg.Heartbeat("a1")
	entries = reg.Discover("executor")
	if !entries[0].Healthy {
		t.Fatal("should be healthy after heartbeat")
	}
}

func TestInMemoryRegistry_EvictStale(t *testing.T) {
	reg := NewInMemoryRegistry(50 * time.Millisecond)
	reg.Register("a1", "executor", nil)
	reg.Register("a2", "executor", nil)

	time.Sleep(100 * time.Millisecond)

	evicted := reg.EvictStale()
	if evicted != 2 {
		t.Fatalf("evicted = %d, want 2", evicted)
	}
	if len(reg.All()) != 0 {
		t.Fatal("should be empty after eviction")
	}
}

func TestInMemoryRegistry_All(t *testing.T) {
	reg := NewInMemoryRegistry(1 * time.Hour)
	reg.Register("a1", "executor", nil)
	reg.Register("a2", "inspector", nil)

	all := reg.All()
	if len(all) != 2 {
		t.Fatalf("all = %d, want 2", len(all))
	}
}

func TestInMemoryRegistry_DiscoverEmpty(t *testing.T) {
	reg := NewInMemoryRegistry(1 * time.Hour)
	if len(reg.Discover("nonexistent")) != 0 {
		t.Fatal("should return empty for unknown role")
	}
}

// Compile-time interface check.
var _ AgentLookup = (*InMemoryRegistry)(nil)

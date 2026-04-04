package transport

import (
	"sync"
	"testing"
)

func TestRoleRegistry_RegisterAndRoleOf(t *testing.T) {
	r := NewRoleRegistry()
	r.Register("agent-1", "executor")

	if got := r.RoleOf("agent-1"); got != "executor" {
		t.Errorf("RoleOf(agent-1) = %q, want %q", got, "executor")
	}
}

func TestRoleRegistry_AgentsForRole(t *testing.T) {
	r := NewRoleRegistry()
	r.Register("agent-1", "executor")
	r.Register("agent-2", "executor")
	r.Register("agent-3", "reviewer")

	agents := r.AgentsForRole("executor")
	if len(agents) != 2 {
		t.Fatalf("AgentsForRole(executor) = %d agents, want 2", len(agents))
	}
	if agents[0] != "agent-1" || agents[1] != "agent-2" {
		t.Errorf("AgentsForRole(executor) = %v, want [agent-1, agent-2]", agents)
	}

	reviewers := r.AgentsForRole("reviewer")
	if len(reviewers) != 1 || reviewers[0] != "agent-3" {
		t.Errorf("AgentsForRole(reviewer) = %v, want [agent-3]", reviewers)
	}
}

func TestRoleRegistry_AgentsForRole_Unknown(t *testing.T) {
	r := NewRoleRegistry()
	if got := r.AgentsForRole("ghost"); got != nil {
		t.Errorf("AgentsForRole(ghost) = %v, want nil", got)
	}
}

func TestRoleRegistry_Unregister(t *testing.T) {
	r := NewRoleRegistry()
	r.Register("agent-1", "executor")
	r.Register("agent-2", "executor")
	r.Unregister("agent-1")

	if got := r.RoleOf("agent-1"); got != "" {
		t.Errorf("RoleOf(agent-1) after Unregister = %q, want empty", got)
	}

	agents := r.AgentsForRole("executor")
	if len(agents) != 1 || agents[0] != "agent-2" {
		t.Errorf("AgentsForRole(executor) after Unregister = %v, want [agent-2]", agents)
	}
}

func TestRoleRegistry_Unregister_LastAgent(t *testing.T) {
	r := NewRoleRegistry()
	r.Register("agent-1", "executor")
	r.Unregister("agent-1")

	if got := r.AgentsForRole("executor"); got != nil {
		t.Errorf("AgentsForRole(executor) after removing last agent = %v, want nil", got)
	}
}

func TestRoleRegistry_Unregister_Unknown(t *testing.T) {
	r := NewRoleRegistry()
	// Should not panic.
	r.Unregister("ghost")
}

func TestRoleRegistry_MultipleAgentsSameRole(t *testing.T) {
	r := NewRoleRegistry()
	r.Register("a", "worker")
	r.Register("b", "worker")
	r.Register("c", "worker")

	agents := r.AgentsForRole("worker")
	if len(agents) != 3 {
		t.Fatalf("AgentsForRole(worker) = %d, want 3", len(agents))
	}

	// Unregister middle one.
	r.Unregister("b")
	agents = r.AgentsForRole("worker")
	if len(agents) != 2 {
		t.Fatalf("after unregister b: AgentsForRole(worker) = %d, want 2", len(agents))
	}
	for _, id := range agents {
		if id == "b" {
			t.Error("agent b still present after Unregister")
		}
	}
}

func TestRoleRegistry_ConcurrentAccess(t *testing.T) {
	r := NewRoleRegistry()
	var wg sync.WaitGroup

	// Concurrent registrations.
	for i := range 100 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := "agent-" + string(rune('A'+i%26))
			r.Register(id, "worker")
			r.RoleOf(id)
			r.AgentsForRole("worker")
		}(i)
	}
	wg.Wait()

	// Concurrent unregistrations.
	for i := range 100 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := "agent-" + string(rune('A'+i%26))
			r.Unregister(id)
		}(i)
	}
	wg.Wait()
}

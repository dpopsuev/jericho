package transport

import "sync"

// RoleRegistry tracks agent ID to role mapping, enabling role-based
// message routing (SendToRole, AskRole, Broadcast).
type RoleRegistry struct {
	mu     sync.RWMutex
	byRole map[string][]string // role -> []agentID
	roleOf map[string]string   // agentID -> role
}

// NewRoleRegistry creates an empty RoleRegistry.
func NewRoleRegistry() *RoleRegistry {
	return &RoleRegistry{
		byRole: make(map[string][]string),
		roleOf: make(map[string]string),
	}
}

// Register associates an agent ID with a role.
func (r *RoleRegistry) Register(agentID, role string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.roleOf[agentID] = role
	r.byRole[role] = append(r.byRole[role], agentID)
}

// Unregister removes an agent from the registry.
func (r *RoleRegistry) Unregister(agentID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	role, ok := r.roleOf[agentID]
	if !ok {
		return
	}
	delete(r.roleOf, agentID)

	agents := r.byRole[role]
	for i, id := range agents {
		if id == agentID {
			r.byRole[role] = append(agents[:i], agents[i+1:]...)
			break
		}
	}
	if len(r.byRole[role]) == 0 {
		delete(r.byRole, role)
	}
}

// AgentsForRole returns all agent IDs registered with the given role.
// Returns nil if no agents have the role.
func (r *RoleRegistry) AgentsForRole(role string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agents := r.byRole[role]
	if len(agents) == 0 {
		return nil
	}
	out := make([]string, len(agents))
	copy(out, agents)
	return out
}

// RoleOf returns the role of the given agent ID, or "" if not registered.
func (r *RoleRegistry) RoleOf(agentID string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.roleOf[agentID]
}

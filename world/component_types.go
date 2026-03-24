package world

import "time"

// Component type constants for core components.
const (
	HealthType    ComponentType = "health"
	HierarchyType ComponentType = "hierarchy"
	BudgetType    ComponentType = "budget"
	ProgressType  ComponentType = "progress"
)

// AgentState represents the liveness state of an agent.
type AgentState string

const (
	Active  AgentState = "active"
	Idle    AgentState = "idle"
	Stale   AgentState = "stale"
	Errored AgentState = "errored"
	Done    AgentState = "done"
)

// Health tracks agent liveness and status.
type Health struct {
	State    AgentState `json:"state"`
	LastSeen time.Time  `json:"last_seen"`
	Error    string     `json:"error,omitempty"`
}

// ComponentType implements Component.
func (Health) ComponentType() ComponentType { return HealthType }

// Hierarchy represents a parent-child relationship.
type Hierarchy struct {
	Parent EntityID `json:"parent"`
}

// ComponentType implements Component.
func (Hierarchy) ComponentType() ComponentType { return HierarchyType }

// Budget tracks cost per entity.
type Budget struct {
	TokensUsed int     `json:"tokens_used"`
	Cost       float64 `json:"cost"`
	Ceiling    float64 `json:"ceiling"`
}

// ComponentType implements Component.
func (Budget) ComponentType() ComponentType { return BudgetType }

// Progress tracks task completion.
type Progress struct {
	Current int     `json:"current"`
	Total   int     `json:"total"`
	Percent float64 `json:"percent"`
}

// ComponentType implements Component.
func (Progress) ComponentType() ComponentType { return ProgressType }

// IdentityStrategy resolves agent roles into fully-formed entities
// with identity components.
type IdentityStrategy interface {
	Resolve(role, collective string) (EntityID, error)
}

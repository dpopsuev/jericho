package warden

// PriorityClass determines scheduling priority when quotas are constrained.
type PriorityClass string

const (
	PriorityCritical PriorityClass = "critical" // never preempted
	PriorityHigh     PriorityClass = "high"
	PriorityNormal   PriorityClass = "normal" // default
	PriorityLow      PriorityClass = "low"    // first to be preempted
)

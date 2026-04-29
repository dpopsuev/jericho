package troupe

import "time"

// Meter records and queries resource usage across actors.
type Meter interface {
	// Record appends a usage entry.
	Record(u Usage)
	// Query returns all usage entries for the given actor.
	Query(actor string) []Usage
}

// Usage records resource consumption for a single operation.
type Usage struct {
	Agent    string
	Step     string
	Duration time.Duration
	Detail   UsageDetail
}

// UsageDetail is the extension point for provider-specific usage data.
// Tokens for cloud, GPU-seconds for on-prem. Same pattern as EventDetail.
type UsageDetail interface {
	String() string
}

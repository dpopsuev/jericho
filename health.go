package troupe

import "context"

// HealthReporter exposes agent health for external monitoring.
// Mount on HTTP endpoints: /healthz (liveness), /readyz (readiness).
type HealthReporter interface {
	// Healthz returns true if the system is alive. Maps to K8s liveness probe.
	Healthz(ctx context.Context) bool

	// Readyz returns true if the system can accept work. Maps to K8s readiness probe.
	Readyz(ctx context.Context) bool

	// Status returns the full health snapshot for all agents.
	Status(ctx context.Context) []AgentStatus
}

package troupe

import "context"

// Lifecycle is the daemon harness contract. trouped implements it
// by coordinating broker startup, accepting connections, draining
// agents, then stopping.
//
// K8s maps:
//   - Start: container entrypoint
//   - Ready: readiness probe returns true
//   - Drain: preStop hook (stop accepting, finish in-flight)
//   - Stop: SIGTERM handler (kill remaining, close buses)
type Lifecycle interface {
	Start(ctx context.Context) error
	Drain(ctx context.Context) error
	Stop(ctx context.Context) error
	Ready() bool
}

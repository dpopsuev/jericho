package troupe

// Probe is an optional interface for actors that support health probing.
// Actors that don't implement it use default behavior:
// startup = always true, liveness = Ready().
//
// K8s semantics:
//   - Startup: has the agent finished initializing? Failed startup → kill+restart.
//   - Liveness: is the agent process alive? Failed liveness → kill+restart.
//   - Readiness: can the agent accept work? Failed readiness → stop sending work.
//
// Agent.Ready() is the readiness probe. Probe adds startup and liveness.
type Probe interface {
	// Startup reports whether the agent has finished initializing.
	// Called repeatedly until it returns true, then never called again.
	Startup() bool

	// Liveness reports whether the agent process is healthy.
	// A false return indicates the agent is stuck and should be restarted.
	Liveness() bool
}

// ProbeOf extracts the Probe from an actor, or returns a default
// that reports always-started and liveness=Ready.
func ProbeOf(actor Agent) Probe {
	if p, ok := actor.(Probe); ok {
		return p
	}
	return &defaultProbe{actor: actor}
}

type defaultProbe struct {
	actor Agent
}

func (p *defaultProbe) Startup() bool  { return true }
func (p *defaultProbe) Liveness() bool { return p.actor.Ready() }

// AgentStatus combines phase (liveness), conditions (readiness),
// and health (Andon) into a single snapshot. K8s Pod status analog.
type AgentStatus struct {
	Phase    string `json:"phase"`            // starting, running, terminated
	Ready    bool   `json:"ready"`            // can accept work
	Reason   string `json:"reason,omitempty"` // why not ready
	Startup  bool   `json:"startup"`          // initialization complete
	Liveness bool   `json:"liveness"`         // process healthy
	Health   string `json:"health,omitempty"` // andon level: nominal, degraded, failure, blocked, dead
}

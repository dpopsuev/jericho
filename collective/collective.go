// collective.go — AgentCollective: N agents behind one facade.Agent interface.
//
// The operator calls Ask() on a collective and gets one response. Internally,
// N agents collaborate via a pluggable CollectiveStrategy (dialectic, arbiter,
// consensus, pipeline). The strategy defines the modus operandi.
package collective

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/dpopsuev/jericho/facade"
	"github.com/dpopsuev/jericho/pool"
	"github.com/dpopsuev/jericho/world"
)

// Sentinel errors for collective operations.
var (
	ErrIngressRejected = errors.New("collective ingress rejected")
	ErrEgressRejected  = errors.New("collective egress rejected")
	ErrNoAgents        = errors.New("collective: no agents")
)

// CollectiveStrategy defines how agents collaborate inside a collective.
type CollectiveStrategy interface {
	Orchestrate(ctx context.Context, prompt string, agents []*facade.AgentHandle) (string, error)
}

// DebateRound records one debate round between agents.
type DebateRound struct {
	ThesisResponse     string
	AntithesisResponse string
	Converged          bool
}

// AgentCollective wraps N agents behind the facade.Agent interface.
// Operators see one agent. Internally, N agents debate/collaborate.
type AgentCollective struct {
	id       world.EntityID
	role     string
	agents   []*facade.AgentHandle
	strategy CollectiveStrategy
	ingress  Gate // optional bouncer (nil = pass-through)
	egress   Gate // optional reviewer (nil = pass-through)
	handler  func(content string) string
	mu       sync.RWMutex
	rounds   []DebateRound
}

// CollectiveOption configures an AgentCollective.
type CollectiveOption func(*AgentCollective)

// WithIngress sets the ingress gate (bouncer).
func WithIngress(g Gate) CollectiveOption {
	return func(c *AgentCollective) { c.ingress = g }
}

// WithEgress sets the egress gate (reviewer).
func WithEgress(g Gate) CollectiveOption {
	return func(c *AgentCollective) { c.egress = g }
}

// NewAgentCollective creates a collective from existing agent handles.
func NewAgentCollective(id world.EntityID, role string, strategy CollectiveStrategy, agents []*facade.AgentHandle, opts ...CollectiveOption) *AgentCollective {
	c := &AgentCollective{
		id:       id,
		role:     role,
		strategy: strategy,
		agents:   agents,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// --- Identity ---

func (c *AgentCollective) ID() world.EntityID { return c.id }
func (c *AgentCollective) Role() string       { return c.role }
func (c *AgentCollective) String() string {
	return fmt.Sprintf("%s(collective-%d, %d agents)", c.role, c.id, len(c.agents))
}

// --- Messaging ---

// Ask runs the collective strategy and returns the synthesized response.
// Applies ingress gate before entry and egress gate before exit.
func (c *AgentCollective) Ask(ctx context.Context, content string) (string, error) {
	// Ingress gate — bouncer decides if prompt enters the room.
	if c.ingress != nil {
		ok, reason, err := c.ingress.Pass(ctx, content)
		if err != nil {
			return "", fmt.Errorf("collective %s ingress: %w", c.role, err)
		}
		if !ok {
			return "", fmt.Errorf("%w: %s %s", ErrIngressRejected, c.role, reason)
		}
	}

	// The room — agents debate/collaborate.
	result, err := c.strategy.Orchestrate(ctx, content, c.agents)
	if err != nil {
		return "", fmt.Errorf("collective %s: %w", c.role, err)
	}

	// Egress gate — reviewer decides if response exits the room.
	if c.egress != nil {
		ok, reason, err := c.egress.Pass(ctx, result)
		if err != nil {
			return "", fmt.Errorf("collective %s egress: %w", c.role, err)
		}
		if !ok {
			return "", fmt.Errorf("%w: %s %s", ErrEgressRejected, c.role, reason)
		}
	}

	return result, nil
}

// Tell forwards to the first agent (no debate for fire-and-forget).
func (c *AgentCollective) Tell(content string) error {
	if len(c.agents) == 0 {
		return fmt.Errorf("%w: %s", ErrNoAgents, c.role)
	}
	return c.agents[0].Tell(content)
}

// Listen registers a handler. The collective's Ask result is passed through
// this handler before returning to the caller.
func (c *AgentCollective) Listen(handler func(content string) string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handler = handler
}

// Broadcast sends to all agents in the collective.
func (c *AgentCollective) Broadcast(ctx context.Context, content string) error {
	for _, a := range c.agents {
		if err := a.Tell(content); err != nil {
			return err
		}
	}
	return nil
}

// --- Lifecycle ---

// Kill stops all internal agents.
func (c *AgentCollective) Kill(ctx context.Context) error {
	for _, a := range c.agents {
		if err := a.Kill(ctx); err != nil {
			return err
		}
	}
	return nil
}

// KillWithReason stops all agents with the given exit code.
func (c *AgentCollective) KillWithReason(ctx context.Context, code pool.ExitCode) error {
	for _, a := range c.agents {
		if err := a.KillWithReason(ctx, code); err != nil {
			return err
		}
	}
	return nil
}

// Wait waits for all internal agents to finish. Returns the last exit status.
func (c *AgentCollective) Wait(ctx context.Context) (*pool.ExitStatus, error) {
	var lastStatus *pool.ExitStatus
	for _, a := range c.agents {
		status, err := a.Wait(ctx)
		if err != nil {
			return nil, err
		}
		lastStatus = status
	}
	return lastStatus, nil
}

// Spawn creates a child agent under the first agent in the collective.
func (c *AgentCollective) Spawn(ctx context.Context, role string, config pool.LaunchConfig) (*facade.AgentHandle, error) {
	if len(c.agents) == 0 {
		return nil, fmt.Errorf("%w: %s (spawn)", ErrNoAgents, c.role)
	}
	return c.agents[0].Spawn(ctx, role, config)
}

// --- State ---

// IsAlive returns true if all agents are alive.
func (c *AgentCollective) IsAlive() bool {
	for _, a := range c.agents {
		if !a.IsAlive() {
			return false
		}
	}
	return len(c.agents) > 0
}

// IsHealthy returns true if all agents are healthy.
func (c *AgentCollective) IsHealthy() bool {
	for _, a := range c.agents {
		if !a.IsHealthy() {
			return false
		}
	}
	return len(c.agents) > 0
}

// Children returns the internal agents (visible in full view).
func (c *AgentCollective) Children() []*facade.AgentHandle {
	return c.agents
}

// Parent returns nil — collectives are created by Staff, not by a parent agent.
func (c *AgentCollective) Parent() *facade.AgentHandle {
	return nil
}

// Progress returns the debate progress: current round / max rounds.
func (c *AgentCollective) Progress() (world.Progress, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.rounds) == 0 {
		return world.Progress{}, false
	}
	return world.Progress{Current: len(c.rounds), Total: len(c.rounds)}, true
}

// SetProgress is a no-op for collectives (progress is driven by rounds).
func (c *AgentCollective) SetProgress(_, _ int) {}

// --- FacadeAgent ---

// InternalAgents returns the agents hidden behind the facade.
func (c *AgentCollective) InternalAgents() []*facade.AgentHandle {
	return c.agents
}

// IsFacade returns true — this is a collective, not a single agent.
func (c *AgentCollective) IsFacade() bool { return true }

// DebateRounds returns the debate history.
func (c *AgentCollective) DebateRounds() []DebateRound {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]DebateRound, len(c.rounds))
	copy(out, c.rounds)
	return out
}

// addDebateRound appends a round to the debate history.
func (c *AgentCollective) addDebateRound(r DebateRound) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rounds = append(c.rounds, r)
}

// Compile-time checks.
var (
	_ facade.Agent       = (*AgentCollective)(nil)
	_ facade.FacadeAgent = (*AgentCollective)(nil)
)

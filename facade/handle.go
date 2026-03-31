// Package facade provides a human-readable API over Bugle's internal
// subsystems (pool, transport, world, signal). AgentHandle wraps a
// single agent; Staff is the top-level entry point.
package facade

import (
	"context"
	"fmt"
	"time"

	"github.com/dpopsuev/jericho/pool"
	"github.com/dpopsuev/jericho/signal"
	"github.com/dpopsuev/jericho/transport"
	"github.com/dpopsuev/jericho/world"
)

// AgentHandle wraps all subsystems into one human-readable object for a
// single agent. Created by Staff.Spawn or AgentHandle.Spawn — never
// instantiated directly.
type AgentHandle struct {
	id        world.EntityID
	role      string
	world     *world.World
	pool      *pool.AgentPool
	transport *transport.LocalTransport
}

// ---------------------------------------------------------------------------
// Identity
// ---------------------------------------------------------------------------

// ID returns the agent's entity ID.
func (a *AgentHandle) ID() world.EntityID { return a.id }

// Role returns the agent's staff role name.
func (a *AgentHandle) Role() string { return a.role }

// String returns a human-readable label: "role(agent-N)".
func (a *AgentHandle) String() string {
	return fmt.Sprintf("%s(%s)", a.role, a.agentID())
}

// ---------------------------------------------------------------------------
// State
// ---------------------------------------------------------------------------

// IsAlive returns true if the agent entity exists in the world.
func (a *AgentHandle) IsAlive() bool {
	return a.world.Alive(a.id)
}

// IsHealthy returns true if the agent's Health component has State == Active.
func (a *AgentHandle) IsHealthy() bool {
	h, ok := world.TryGet[world.Health](a.world, a.id)
	if !ok {
		return false
	}
	return h.State == world.Active
}

// IsZombie returns true if the agent is finished but not yet reaped.
func (a *AgentHandle) IsZombie() bool {
	return a.pool.IsZombie(a.id)
}

// Health returns the agent's Health component.
func (a *AgentHandle) Health() (world.Health, bool) {
	return world.TryGet[world.Health](a.world, a.id)
}

// Budget returns the agent's Budget component.
func (a *AgentHandle) Budget() (world.Budget, bool) {
	return world.TryGet[world.Budget](a.world, a.id)
}

// Progress returns the agent's Progress component.
func (a *AgentHandle) Progress() (world.Progress, bool) {
	return world.TryGet[world.Progress](a.world, a.id)
}

// Display returns the agent's Display component (name, color, icon).
func (a *AgentHandle) Display() (world.Display, bool) {
	return world.TryGet[world.Display](a.world, a.id)
}

// SetDisplay attaches or updates the Display component.
func (a *AgentHandle) SetDisplay(d world.Display) {
	world.Attach(a.world, a.id, d)
}

// SetProgress attaches or updates the Progress component.
func (a *AgentHandle) SetProgress(current, total int) {
	pct := 0.0
	if total > 0 {
		pct = float64(current) / float64(total) * 100
	}
	world.Attach(a.world, a.id, world.Progress{
		Current: current,
		Total:   total,
		Percent: pct,
	})
}

// Uptime returns how long the agent has been running (or total runtime if finished).
func (a *AgentHandle) Uptime() time.Duration {
	return a.pool.Uptime(a.id)
}

// ---------------------------------------------------------------------------
// Messaging
// ---------------------------------------------------------------------------

// Ask sends a message to this agent and blocks until a response is received.
// Returns the response content string.
func (a *AgentHandle) Ask(ctx context.Context, content string) (string, error) {
	msg := transport.Message{
		From:         "facade",
		To:           a.agentID(),
		Performative: signal.Request,
		Content:      content,
	}
	resp, err := a.transport.Ask(ctx, a.agentID(), msg)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

// Tell sends a fire-and-forget message to this agent.
func (a *AgentHandle) Tell(content string) error {
	msg := transport.Message{
		From:         "facade",
		To:           a.agentID(),
		Performative: signal.Inform,
		Content:      content,
	}
	_, err := a.transport.SendMessage(context.Background(), a.agentID(), msg)
	return err
}

// AskWithPerformative sends a message with a specific performative and blocks
// until a response is received. Returns the response content string.
func (a *AgentHandle) AskWithPerformative(ctx context.Context, perf signal.Performative, content string) (string, error) {
	msg := transport.Message{
		From:         "facade",
		To:           a.agentID(),
		Performative: perf,
		Content:      content,
	}
	resp, err := a.transport.Ask(ctx, a.agentID(), msg)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

// Broadcast sends a message to ALL agents with this agent's role.
func (a *AgentHandle) Broadcast(ctx context.Context, content string) error {
	msg := transport.Message{
		From:         a.agentID(),
		Performative: signal.Inform,
		Content:      content,
	}
	_, err := a.transport.Broadcast(ctx, a.role, msg)
	return err
}

// Listen registers a simplified handler for incoming messages to this agent.
// The handler receives the message content and returns a response content string.
// It replaces any previously registered handler for this agent.
func (a *AgentHandle) Listen(handler func(content string) string) {
	agentID := a.agentID()
	a.transport.Register(agentID, func(_ context.Context, msg transport.Message) (transport.Message, error) {
		resp := handler(msg.Content)
		return transport.Message{
			From:    agentID,
			Content: resp,
		}, nil
	})
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

// Spawn creates a child agent under this agent as parent.
func (a *AgentHandle) Spawn(ctx context.Context, role string, config pool.LaunchConfig) (*AgentHandle, error) {
	id, err := a.pool.Fork(ctx, role, config, a.id)
	if err != nil {
		return nil, err
	}
	return &AgentHandle{
		id:        id,
		role:      role,
		world:     a.world,
		pool:      a.pool,
		transport: a.transport,
	}, nil
}

// Kill stops this agent.
func (a *AgentHandle) Kill(ctx context.Context) error {
	return a.pool.Kill(ctx, a.id)
}

// KillWithReason stops this agent with a specific exit code.
func (a *AgentHandle) KillWithReason(ctx context.Context, code pool.ExitCode) error {
	return a.pool.KillWithCode(ctx, a.id, code)
}

// Wait blocks until this agent finishes and returns its exit status.
func (a *AgentHandle) Wait(ctx context.Context) (*pool.ExitStatus, error) {
	return a.pool.Wait(ctx, a.id)
}

// Children returns handles for all direct children of this agent.
func (a *AgentHandle) Children() []*AgentHandle {
	childIDs := a.pool.Children(a.id)
	handles := make([]*AgentHandle, 0, len(childIDs))
	for _, cid := range childIDs {
		role := a.transport.Roles().RoleOf(agentTransportID(cid))
		handles = append(handles, &AgentHandle{
			id:        cid,
			role:      role,
			world:     a.world,
			pool:      a.pool,
			transport: a.transport,
		})
	}
	return handles
}

// Parent returns a handle for this agent's parent, or nil if root (parentID == 0).
func (a *AgentHandle) Parent() *AgentHandle {
	parentID := a.pool.ParentOf(a.id)
	if parentID == 0 {
		return nil
	}
	role := a.transport.Roles().RoleOf(agentTransportID(parentID))
	return &AgentHandle{
		id:        parentID,
		role:      role,
		world:     a.world,
		pool:      a.pool,
		transport: a.transport,
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// agentID returns the transport-level identifier for this agent.
func (a *AgentHandle) agentID() string {
	return agentTransportID(a.id)
}

// agentTransportID converts an EntityID to the transport agent ID string.
func agentTransportID(id world.EntityID) string {
	return fmt.Sprintf("agent-%d", id)
}

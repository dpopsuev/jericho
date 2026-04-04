// agent.go — Agent interface for single agents and collectives.
//
// Agent is the common contract for anything that behaves like an agent.
// Solo (single agent) and Collective (N agents) both implement it.
// Operators program against Agent when they don't care whether they're
// talking to one agent or a facade over many.
package agent

import (
	"context"

	"github.com/dpopsuev/jericho/internal/warden"
	"github.com/dpopsuev/jericho/world"
)

// Agent is the interface for interacting with an agent — single or collective.
type Agent interface {
	// Identity
	ID() world.EntityID
	Role() string
	String() string

	// Messaging
	Perform(ctx context.Context, content string) (string, error)
	Tell(content string) error
	Listen(handler func(content string) string)
	Broadcast(ctx context.Context, content string) error

	// Lifecycle
	Kill(ctx context.Context) error
	KillWithReason(ctx context.Context, code warden.ExitCode) error
	Wait(ctx context.Context) (*warden.ExitStatus, error)
	Spawn(ctx context.Context, role string, config warden.AgentConfig) (*Solo, error)

	// State
	IsAlive() bool
	IsHealthy() bool
	Children() []*Solo
	Parent() *Solo
	Progress() (world.Progress, bool)
	SetProgress(current, total int)
}

// FacadeAgent is a marker interface for agents that wrap other agents.
// Used by Tree() vs TreeFull() to decide whether to show internals.
type FacadeAgent interface {
	Agent
	InternalAgents() []*Solo
	IsFacade() bool
}

// Compile-time checks.
var _ Agent = (*Solo)(nil)

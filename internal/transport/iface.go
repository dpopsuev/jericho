package transport

import "context"

// Registrar handles agent registration and role management.
type Registrar interface {
	Register(agentID AgentID, handler MsgHandler) error
	Unregister(agentID AgentID)
	Roles() *RoleRegistry
}

// Sender dispatches messages to agents.
type Sender interface {
	SendMessage(ctx context.Context, to AgentID, msg Message) (*Task, error)
	Ask(ctx context.Context, to AgentID, msg Message) (Message, error)
	Broadcast(ctx context.Context, role string, msg Message) ([]*Task, error)
	SendToRole(ctx context.Context, role string, msg Message) (*Task, error)
	AskRole(ctx context.Context, role string, msg Message) (Message, error)
}

// Subscriber observes task lifecycle events.
type Subscriber interface {
	Subscribe(ctx context.Context, taskID string) (<-chan Event, error)
}

// Transport is the full agent-to-agent messaging interface.
// Composed of Registrar + Sender + Subscriber + Close.
type Transport interface {
	Registrar
	Sender
	Subscriber
	Close() error
}

// Verify both transports implement the interface.
var (
	_ Transport = (*LocalTransport)(nil)
	_ Transport = (*HTTPTransport)(nil)
)

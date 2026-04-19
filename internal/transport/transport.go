// Package transport defines the A2A transport abstraction for agent-to-agent
// communication. LocalTransport provides in-process channel-based messaging;
// HTTPTransport connects to remote A2A agents via a2a-go SDK.
package transport

import "context"

// A2A message roles.
const (
	RoleUser  = "user"
	RoleAgent = "agent"
)

// AgentID is a typed identifier for agents in the transport layer.
type AgentID string

// Message is the internal routing envelope for agent-to-agent communication.
type Message struct {
	From     AgentID           `json:"from"`
	To       AgentID           `json:"to"`
	Role     string            `json:"role,omitempty"`
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata,omitempty"`
	TraceID  string            `json:"trace_id,omitempty"`
}

// Task represents an in-flight message processing job.
type Task struct {
	ID      string    `json:"id"`
	State   TaskState `json:"state"`
	Result  *Message  `json:"result,omitempty"`
	Error   string    `json:"error,omitempty"`
	History []Message `json:"history,omitempty"`
}

// TaskState is the lifecycle state of a Task.
type TaskState string

const (
	TaskSubmitted TaskState = "submitted"
	TaskWorking   TaskState = "working"
	TaskCompleted TaskState = "completed"
	TaskFailed    TaskState = "failed"
	TaskCanceled  TaskState = "canceled"
)

// Event is a state-change notification for a Task.
type Event struct {
	TaskID string    `json:"task_id"`
	State  TaskState `json:"state"`
	Data   *Message  `json:"data,omitempty"`
}

// MsgHandler processes a received message and returns a response.
type MsgHandler func(ctx context.Context, msg Message) (Message, error)

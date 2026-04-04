// Package transport defines the A2A transport abstraction for agent-to-agent
// communication. LocalTransport provides in-process channel-based messaging;
// HTTPTransport is deferred.
//
// HTTPTransport connects to remote A2A agents via a2a-go SDK.
// Deferred to BGL-TSK-16b — requires github.com/a2aproject/a2a-go dependency.
package transport

import (
	"context"

	"github.com/dpopsuev/jericho/signal"
)

// AgentID is a typed identifier for agents in the transport layer.
// Prevents accidental mixing with role names, session IDs, or other strings.
type AgentID string

// Transport is the send/subscribe interface for agent-to-agent communication.
type Transport interface {
	// SendMessage dispatches a message to the named agent and returns a Task
	// that tracks its progress. The call is asynchronous — the returned Task
	// starts in TaskSubmitted state and transitions as the handler executes.
	SendMessage(ctx context.Context, to string, msg Message) (*Task, error)

	// Subscribe returns a channel that receives Events whenever the given
	// task transitions state.
	Subscribe(ctx context.Context, taskID string) (<-chan Event, error)

	// Close releases all resources held by the transport.
	Close() error
}

// Message is the envelope for agent-to-agent communication.
type Message struct {
	From         AgentID             `json:"from"`
	To           AgentID             `json:"to"`
	Performative signal.Performative `json:"performative"`
	Content      string              `json:"content"`
	Metadata     map[string]string   `json:"metadata,omitempty"`
	TraceID      string              `json:"trace_id,omitempty"`
}

// Task represents an in-flight message processing job.
type Task struct {
	ID     string    `json:"id"`
	State  TaskState `json:"state"`
	Result *Message  `json:"result,omitempty"`
	Error  string    `json:"error,omitempty"`
}

// TaskState is the lifecycle state of a Task.
type TaskState string

const (
	// TaskSubmitted means the task has been created but not yet picked up.
	TaskSubmitted TaskState = "submitted"
	// TaskWorking means the handler is actively processing the message.
	TaskWorking TaskState = "working"
	// TaskCompleted means the handler finished successfully.
	TaskCompleted TaskState = "completed"
	// TaskFailed means the handler returned an error.
	TaskFailed TaskState = "failed"
	// TaskCanceled means the task was canceled before completion.
	TaskCanceled TaskState = "canceled"
)

// Event is a state-change notification for a Task.
type Event struct {
	TaskID string    `json:"task_id"`
	State  TaskState `json:"state"`
	Data   *Message  `json:"data,omitempty"`
}

// MsgHandler processes a received message and returns a response.
type MsgHandler func(ctx context.Context, msg Message) (Message, error)

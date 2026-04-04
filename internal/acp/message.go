// message.go — ACP-native message and streaming types.
//
// These types model the agent communication protocol at the wire level.
// Consumers (Djinn, Origami) convert to their own domain types as needed.
package acp

import "encoding/json"

// Roles.
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Message is a single conversational turn in an ACP session.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// StreamEvent types.
const (
	EventText     = "text"
	EventThinking = "thinking"
	EventToolUse  = "tool_use"
	EventDone     = "done"
	EventError    = "error"
)

// StreamEvent is a single event from an ACP streaming response.
type StreamEvent struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	Thinking string    `json:"thinking,omitempty"`
	ToolCall *ToolCall `json:"tool_call,omitempty"`
	Usage    *Usage    `json:"usage,omitempty"`
	Error    string    `json:"error,omitempty"`
}

// ToolCall is an agent's request to execute a tool.
type ToolCall struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input,omitempty"`
}

// Usage tracks token consumption.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

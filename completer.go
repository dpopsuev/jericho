package troupe

import (
	"context"
	"encoding/json"
)

type CompletionParams struct {
	Prompt    string
	Messages  []Message
	Tools     []Tool
	MaxTokens int
}

type Message struct {
	Role       string
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string
}

type Completion struct {
	Content   string
	ToolCalls []ToolCall
	Tokens    TokenUsage
}

type Tool struct {
	Name        string
	Description string
	InputSchema json.RawMessage
}

type ToolCall struct {
	ID    string
	Name  string
	Input json.RawMessage
}

type TokenUsage struct {
	Input  int
	Output int
}

type Completer interface {
	Complete(ctx context.Context, params CompletionParams) (*Completion, error)
}

type CompleteFunc func(ctx context.Context, params CompletionParams) (*Completion, error)

func (f CompleteFunc) Complete(ctx context.Context, params CompletionParams) (*Completion, error) {
	return f(ctx, params)
}

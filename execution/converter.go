// converter.go — MessageConverter: compile-time enforcement for message role handling.
//
// Every provider that converts anyllm.Messages to a native API format
// MUST implement all methods. Missing a role = compile error.
// This prevents TRP-BUG-2 (silently dropping tool/system messages).
package execution

import (
	"encoding/json"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"

	anyllm "github.com/mozilla-ai/any-llm-go/providers"
)

// MessageConverter enforces per-role message handling at compile time.
// Each method handles one message role. Adding a new role means adding
// a method — all implementations fail to compile until updated.
type MessageConverter interface {
	ConvertUser(msg anyllm.Message) anthropic.MessageParam
	ConvertAssistant(msg anyllm.Message) anthropic.MessageParam
	ConvertSystem(msg anyllm.Message) string
	ConvertToolResult(msg anyllm.Message) anthropic.MessageParam
}

// VertexConverter converts anyllm messages to Anthropic SDK types for Vertex AI.
type VertexConverter struct{}

var _ MessageConverter = (*VertexConverter)(nil)

// ConvertUser converts a user message to Anthropic format.
func (VertexConverter) ConvertUser(msg anyllm.Message) anthropic.MessageParam {
	content, _ := msg.Content.(string)
	return anthropic.NewUserMessage(anthropic.NewTextBlock(content))
}

// ConvertAssistant converts an assistant message, preserving tool_use blocks.
func (VertexConverter) ConvertAssistant(msg anyllm.Message) anthropic.MessageParam {
	if len(msg.ToolCalls) == 0 {
		content, _ := msg.Content.(string)
		return anthropic.NewAssistantMessage(anthropic.NewTextBlock(content))
	}

	blocks := make([]anthropic.ContentBlockParamUnion, 0, len(msg.ToolCalls)+1)
	if content, _ := msg.Content.(string); content != "" {
		blocks = append(blocks, anthropic.NewTextBlock(content))
	}
	for _, tc := range msg.ToolCalls {
		var input map[string]any
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &input)
		blocks = append(blocks, anthropic.ContentBlockParamUnion{
			OfToolUse: &anthropic.ToolUseBlockParam{
				Type:  "tool_use",
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: input,
			},
		})
	}
	return anthropic.NewAssistantMessage(blocks...)
}

// ConvertSystem extracts system message text.
func (VertexConverter) ConvertSystem(msg anyllm.Message) string {
	content, _ := msg.Content.(string)
	return content
}

// ConvertToolResult converts a tool result to a user message with tool_result block.
func (VertexConverter) ConvertToolResult(msg anyllm.Message) anthropic.MessageParam {
	content, _ := msg.Content.(string)
	return anthropic.NewUserMessage(
		anthropic.NewToolResultBlock(msg.ToolCallID, content, false),
	)
}

// convertAllMessages dispatches each message through the converter.
// Returns the converted messages and the combined system prompt.
func convertAllMessages(conv MessageConverter, msgs []anyllm.Message) ([]anthropic.MessageParam, string) {
	out := make([]anthropic.MessageParam, 0, len(msgs))
	var systemParts []string

	for _, m := range msgs {
		switch m.Role {
		case "system":
			systemParts = append(systemParts, conv.ConvertSystem(m))
		case vertexRoleUser:
			out = append(out, conv.ConvertUser(m))
		case vertexRoleAssistant:
			out = append(out, conv.ConvertAssistant(m))
		case "tool":
			out = append(out, conv.ConvertToolResult(m))
		}
	}

	return out, strings.Join(systemParts, "\n")
}

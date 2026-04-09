package execution

import (
	"testing"

	anyllm "github.com/mozilla-ai/any-llm-go/providers"
)

func TestConvertMessages_UserOnly(t *testing.T) {
	msgs := []anyllm.Message{{Role: "user", Content: "hello"}}
	out, system := convertAllMessages(VertexConverter{}, msgs)

	if system != "" {
		t.Fatalf("system = %q, want empty", system)
	}
	if len(out) != 1 {
		t.Fatalf("messages = %d, want 1", len(out))
	}
}

func TestConvertMessages_SystemExtracted(t *testing.T) {
	msgs := []anyllm.Message{
		{Role: "system", Content: "respond in French"},
		{Role: "user", Content: "hello"},
	}
	out, system := convertAllMessages(VertexConverter{}, msgs)

	if system != "respond in French" {
		t.Fatalf("system = %q, want 'respond in French'", system)
	}
	if len(out) != 1 {
		t.Fatalf("messages = %d, want 1 (system should be extracted)", len(out))
	}
}

func TestConvertMessages_ToolResult(t *testing.T) {
	msgs := []anyllm.Message{
		{Role: "tool", ToolCallID: "call-1", Content: "4"},
	}
	out, _ := convertAllMessages(VertexConverter{}, msgs)

	if len(out) != 1 {
		t.Fatalf("messages = %d, want 1", len(out))
	}
	// Tool results become user messages with tool_result content blocks
	// (Anthropic API requires tool_result as user-role messages)
}

func TestConvertMessages_AssistantWithToolCalls(t *testing.T) {
	msgs := []anyllm.Message{
		{
			Role:    "assistant",
			Content: "I'll calculate",
			ToolCalls: []anyllm.ToolCall{
				{ID: "c1", Type: "function", Function: anyllm.FunctionCall{Name: "calc", Arguments: `{"expr":"2+2"}`}},
			},
		},
	}
	out, _ := convertAllMessages(VertexConverter{}, msgs)

	if len(out) != 1 {
		t.Fatalf("messages = %d, want 1", len(out))
	}
	// The assistant message should contain both text and tool_use blocks
	// (not just text like the old broken convertMessages)
}

func TestConvertMessages_FullRoundTrip(t *testing.T) {
	msgs := []anyllm.Message{
		{Role: "system", Content: "be helpful"},
		{Role: "user", Content: "what is 2+2?"},
		{
			Role:    "assistant",
			Content: "I'll calculate",
			ToolCalls: []anyllm.ToolCall{
				{ID: "c1", Type: "function", Function: anyllm.FunctionCall{Name: "calc", Arguments: `{"expr":"2+2"}`}},
			},
		},
		{Role: "tool", ToolCallID: "c1", Content: "4"},
	}
	out, system := convertAllMessages(VertexConverter{}, msgs)

	if system != "be helpful" {
		t.Fatalf("system = %q", system)
	}
	// 3 messages: user + assistant + user(tool_result)
	if len(out) != 3 {
		t.Fatalf("messages = %d, want 3", len(out))
	}
}

func TestConvertMessages_EmptyContent(t *testing.T) {
	msgs := []anyllm.Message{
		{Role: "user", Content: ""},
	}
	out, _ := convertAllMessages(VertexConverter{}, msgs)

	if len(out) != 1 {
		t.Fatalf("messages = %d, want 1", len(out))
	}
	// No panic on empty content
}

func TestConvertMessages_UnknownRole(t *testing.T) {
	msgs := []anyllm.Message{
		{Role: "custom", Content: "ignored"},
		{Role: "user", Content: "kept"},
	}
	out, _ := convertAllMessages(VertexConverter{}, msgs)

	if len(out) != 1 {
		t.Fatalf("messages = %d, want 1 (unknown role should be skipped)", len(out))
	}
}

func TestConvertMessages_MultipleSystemMessages(t *testing.T) {
	msgs := []anyllm.Message{
		{Role: "system", Content: "rule 1"},
		{Role: "system", Content: "rule 2"},
		{Role: "user", Content: "hello"},
	}
	_, system := convertAllMessages(VertexConverter{}, msgs)

	if system != "rule 1\nrule 2" {
		t.Fatalf("system = %q, want 'rule 1\\nrule 2'", system)
	}
}

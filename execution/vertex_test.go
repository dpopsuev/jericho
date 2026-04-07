package execution

import (
	"context"
	"os"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"

	anyllm "github.com/mozilla-ai/any-llm-go/providers"
)

func TestConvertMessages(t *testing.T) {
	t.Parallel()
	msgs := convertMessages([]anyllm.Message{
		{Role: vertexRoleUser, Content: "hello"},
		{Role: vertexRoleAssistant, Content: "hi"},
		{Role: vertexRoleUser, Content: "how are you"},
	})

	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
}

func TestConvertMessages_SkipsUnknownRoles(t *testing.T) {
	t.Parallel()
	msgs := convertMessages([]anyllm.Message{
		{Role: vertexRoleUser, Content: "hello"},
		{Role: "system", Content: "ignored"},
		{Role: vertexRoleAssistant, Content: "hi"},
	})

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages (system skipped), got %d", len(msgs))
	}
}

func TestConvertResponse(t *testing.T) {
	t.Parallel()
	resp := convertResponse(&anthropic.Message{
		ID:    "msg-123",
		Model: vertexDefaultModel,
		Content: []anthropic.ContentBlockUnion{
			{Type: vertexBlockTypeText, Text: "hello world"},
		},
		StopReason: "end_turn",
		Usage: anthropic.Usage{
			InputTokens:  10,
			OutputTokens: 5,
		},
	})

	if resp.ID != "msg-123" {
		t.Errorf("ID = %q, want msg-123", resp.ID)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("Choices = %d, want 1", len(resp.Choices))
	}
	content, _ := resp.Choices[0].Message.Content.(string)
	if content != "hello world" {
		t.Errorf("Content = %q, want hello world", content)
	}
	if resp.Choices[0].FinishReason != "end_turn" {
		t.Errorf("FinishReason = %q, want end_turn", resp.Choices[0].FinishReason)
	}
	if resp.Usage.PromptTokens != 10 {
		t.Errorf("PromptTokens = %d, want 10", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 5 {
		t.Errorf("CompletionTokens = %d, want 5", resp.Usage.CompletionTokens)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("TotalTokens = %d, want 15", resp.Usage.TotalTokens)
	}
}

func TestConvertResponse_MultipleBlocks(t *testing.T) {
	t.Parallel()
	resp := convertResponse(&anthropic.Message{
		Content: []anthropic.ContentBlockUnion{
			{Type: vertexBlockTypeText, Text: "hello "},
			{Type: "tool_use", Text: ""},
			{Type: vertexBlockTypeText, Text: "world"},
		},
	})

	content, _ := resp.Choices[0].Message.Content.(string)
	if content != "hello world" {
		t.Errorf("Content = %q, want 'hello world'", content)
	}
}

func TestNewProviderFromEnv_Vertex(t *testing.T) {
	t.Setenv(envUseVertex, "1")
	t.Setenv(envVertexRegion, "us-east5")
	t.Setenv(envVertexProject, "test-project")
	// Clear other keys so Vertex wins
	t.Setenv(envAnthropicKey, "")
	t.Setenv(envOpenAIKey, "")
	t.Setenv(envGeminiKey, "")

	p, err := NewProviderFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != vertexProviderName {
		t.Errorf("Name() = %q, want %q", p.Name(), vertexProviderName)
	}
}

func TestNewProviderFromEnv_FallsThrough(t *testing.T) {
	t.Setenv(envUseVertex, "")
	t.Setenv(envAnthropicKey, "")
	t.Setenv(envOpenAIKey, "")
	t.Setenv(envGeminiKey, "")

	_, err := NewProviderFromEnv()
	if err == nil {
		t.Fatal("expected error when no provider configured")
	}
}

func TestNewProviderByName_Vertex(t *testing.T) {
	t.Setenv(envUseVertex, "1")
	t.Setenv(envVertexRegion, "us-east5")
	t.Setenv(envVertexProject, "test-project")

	p, err := NewProviderByName("claude")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != vertexProviderName {
		t.Errorf("Name() = %q, want %q", p.Name(), vertexProviderName)
	}
}

// TestVertexProvider_E2E_RealCall makes a real API call to Vertex AI.
// Requires: gcloud auth application-default login + env vars set.
// Skips if env vars not configured.
func TestVertexProvider_E2E_RealCall(t *testing.T) {
	if os.Getenv(envUseVertex) != "1" {
		t.Skip("CLAUDE_CODE_USE_VERTEX not set")
	}
	region := os.Getenv(envVertexRegion)
	project := os.Getenv(envVertexProject)
	if region == "" || project == "" {
		t.Skip("CLOUD_ML_REGION or ANTHROPIC_VERTEX_PROJECT_ID not set")
	}

	ctx := context.Background()
	p, err := NewVertexProvider(ctx, region, project)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := p.Completion(ctx, anyllm.CompletionParams{
		Model: vertexDefaultModel,
		Messages: []anyllm.Message{
			{Role: vertexRoleUser, Content: "Reply with exactly one word: hello"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("no choices returned")
	}
	content, _ := resp.Choices[0].Message.Content.(string)
	if content == "" {
		t.Fatal("empty response content")
	}
	t.Logf("Vertex response: %q (tokens: %d in, %d out)", content, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
}

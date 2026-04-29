package providers

import (
	"context"
	"os"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"

	anyllm "github.com/mozilla-ai/any-llm-go/providers"
)

func TestVertexProvider_MissingModel(t *testing.T) {
	t.Parallel()
	p := &VertexProvider{}
	_, err := p.Completion(context.Background(), anyllm.CompletionParams{
		Messages: []anyllm.Message{{Role: vertexRoleUser, Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error when model is empty")
	}
}

func TestConvertMessages_Legacy(t *testing.T) {
	t.Parallel()
	msgs, _ := convertMessages([]anyllm.Message{
		{Role: vertexRoleUser, Content: "hello"},
		{Role: vertexRoleAssistant, Content: "hi"},
		{Role: vertexRoleUser, Content: "how are you"},
	})

	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
}

func TestConvertMessages_SkipsUnknownRoles_Legacy(t *testing.T) {
	t.Parallel()
	msgs, system := convertMessages([]anyllm.Message{
		{Role: vertexRoleUser, Content: "hello"},
		{Role: "system", Content: "extracted"},
		{Role: vertexRoleAssistant, Content: "hi"},
	})

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages (system extracted), got %d", len(msgs))
	}
	if system != "extracted" {
		t.Fatalf("system = %q, want 'extracted'", system)
	}
}

func TestConvertResponse(t *testing.T) {
	t.Parallel()
	resp := convertResponse(&anthropic.Message{
		ID:    "msg-123",
		Model: "claude-sonnet-4",
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
	t.Setenv(envProvider, "vertex-ai")
	t.Setenv(envVertexRegion, "us-east5")
	t.Setenv(envVertexProject, "test-project")

	p, err := NewProviderFromEnv("")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != vertexProviderName {
		t.Errorf("Name() = %q, want %q", p.Name(), vertexProviderName)
	}
}

func TestNewProviderFromEnv_NotSet(t *testing.T) {
	t.Setenv(envProvider, "")

	_, err := NewProviderFromEnv("")
	if err == nil {
		t.Fatal("expected error when TROUPE_PROVIDER not set")
	}
}

func TestNewProviderFromEnv_CustomEnvName(t *testing.T) {
	t.Setenv("DJINN_PROVIDER", "vertex-ai")
	t.Setenv(envVertexRegion, "us-east5")
	t.Setenv(envVertexProject, "test-project")

	p, err := NewProviderFromEnv("DJINN_PROVIDER")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != vertexProviderName {
		t.Errorf("Name() = %q, want %q", p.Name(), vertexProviderName)
	}
}

func TestNewProviderByName_MissingCredentials(t *testing.T) {
	t.Setenv(envAnthropicKey, "")
	_, err := NewProviderByName("anthropic-api")
	if err == nil {
		t.Fatal("expected error when ANTHROPIC_API_KEY not set")
	}

	t.Setenv(envOpenAIKey, "")
	_, err = NewProviderByName("openai-api")
	if err == nil {
		t.Fatal("expected error when OPENAI_API_KEY not set")
	}

	t.Setenv(envGeminiKey, "")
	_, err = NewProviderByName("gemini-api")
	if err == nil {
		t.Fatal("expected error when GEMINI_API_KEY not set")
	}

	t.Setenv(envOpenRouterKey, "")
	_, err = NewProviderByName("openrouter")
	if err == nil {
		t.Fatal("expected error when OPENROUTER_API_KEY not set")
	}
}

func TestNewProviderByName_Vertex(t *testing.T) {
	t.Setenv(envVertexRegion, "us-east5")
	t.Setenv(envVertexProject, "test-project")

	p, err := NewProviderByName("vertex-ai")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != vertexProviderName {
		t.Errorf("Name() = %q, want %q", p.Name(), vertexProviderName)
	}
}

func TestNewProviderByName_UnknownProvider(t *testing.T) {
	_, err := NewProviderByName("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

// TestOpenRouter_E2E_RealCall makes a real API call through OpenRouter.
// Requires OPENROUTER_API_KEY set.
func TestOpenRouter_E2E_RealCall(t *testing.T) {
	if os.Getenv("OPENROUTER_API_KEY") == "" {
		t.Skip("OPENROUTER_API_KEY not set")
	}

	t.Setenv(envAnthropicKey, "")
	t.Setenv(envOpenAIKey, "")
	t.Setenv(envGeminiKey, "")
	t.Setenv(envOpenRouterKey, os.Getenv("OPENROUTER_API_KEY"))
	t.Setenv(envProvider, "openrouter")

	p, err := NewProviderFromEnv("")
	if err != nil {
		t.Fatal(err)
	}

	resp, err := p.Completion(context.Background(), anyllm.CompletionParams{
		Model:    "anthropic/claude-sonnet-4",
		Messages: []anyllm.Message{{Role: "user", Content: "Reply with one word: hello"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	content, _ := resp.Choices[0].Message.Content.(string)
	if content == "" {
		t.Fatal("empty response")
	}
	t.Logf("OpenRouter response: %q", content)
}

// TestVertexProvider_E2E_UsageTracking makes a real Vertex call and
// verifies usage data flows through the UsageRecorder.
func TestVertexProvider_E2E_UsageTracking(t *testing.T) {
	if os.Getenv("TROUPE_TEST_LIVE_LLM") == "" {
		t.Skip("TROUPE_TEST_LIVE_LLM not set")
	}
	region := os.Getenv(envVertexRegion)
	project := os.Getenv(envVertexProject)
	if region == "" || project == "" {
		t.Skip("GOOGLE_CLOUD_LOCATION or GOOGLE_CLOUD_PROJECT not set")
	}

	ctx := context.Background()
	p, err := NewVertexProvider(ctx, region, project)
	if err != nil {
		t.Fatal(err)
	}

	var totalIn, totalOut int
	recorder := func(model string, usage *anyllm.Usage) {
		totalIn += usage.PromptTokens
		totalOut += usage.CompletionTokens
		t.Logf("Usage: model=%s in=%d out=%d", model, usage.PromptTokens, usage.CompletionTokens)
	}

	actor := NewCompleter(p, "claude-sonnet-4", recorder)

	result, err := actor(ctx, "Reply with exactly one word: hello")
	if err != nil {
		t.Fatal(err)
	}
	if result == "" {
		t.Fatal("empty response")
	}

	if totalIn == 0 {
		t.Error("PromptTokens not recorded")
	}
	if totalOut == 0 {
		t.Error("CompletionTokens not recorded")
	}
	t.Logf("Total: %d in + %d out = %d tokens, response: %q", totalIn, totalOut, totalIn+totalOut, result)
}

// TestVertexProvider_E2E_RealCall makes a real API call to Vertex AI.
// Requires: gcloud auth application-default login + env vars set.
// Skips if env vars not configured.
func TestVertexProvider_E2E_RealCall(t *testing.T) {
	if os.Getenv("TROUPE_TEST_LIVE_LLM") == "" {
		t.Skip("TROUPE_TEST_LIVE_LLM not set")
	}
	region := os.Getenv(envVertexRegion)
	project := os.Getenv(envVertexProject)
	if region == "" || project == "" {
		t.Skip("GOOGLE_CLOUD_LOCATION or GOOGLE_CLOUD_PROJECT not set")
	}

	ctx := context.Background()
	p, err := NewVertexProvider(ctx, region, project)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := p.Completion(ctx, anyllm.CompletionParams{
		Model: "claude-sonnet-4",
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

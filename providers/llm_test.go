package providers_test

import (
	"context"
	"encoding/json"
	"testing"

	tangle "github.com/dpopsuev/tangle"
	"github.com/dpopsuev/tangle/providers"
	anyllm "github.com/mozilla-ai/any-llm-go/providers"
)

type stubProvider struct {
	response  string
	toolCalls []anyllm.ToolCall
	usage     *anyllm.Usage
}

func (s *stubProvider) Name() string { return "stub" }

func (s *stubProvider) Completion(_ context.Context, params anyllm.CompletionParams) (*anyllm.ChatCompletion, error) {
	msg := anyllm.Message{Role: "assistant", Content: s.response}
	if len(s.toolCalls) > 0 {
		msg.ToolCalls = s.toolCalls
	}
	return &anyllm.ChatCompletion{
		Model: params.Model,
		Choices: []anyllm.Choice{
			{Message: msg},
		},
		Usage: s.usage,
	}, nil
}

func (s *stubProvider) CompletionStream(_ context.Context, _ anyllm.CompletionParams) (<-chan anyllm.ChatCompletionChunk, <-chan error) {
	return nil, nil
}

func TestLLMActorFunc_ReturnsResponse(t *testing.T) {
	provider := &stubProvider{response: "hello from LLM"}
	actor := providers.NewCompleter(provider, "test-model", nil)

	result, err := actor(context.Background(), tangle.CompletionParams{Prompt: "test prompt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Content != "hello from LLM" {
		t.Errorf("got %q, want %q", result.Content, "hello from LLM")
	}
}

func TestLLMActorFunc_ReusesConnection(t *testing.T) {
	provider := &stubProvider{response: "warm"}
	actor := providers.NewCompleter(provider, "test-model", nil)

	for i := range 3 {
		result, err := actor(context.Background(), tangle.CompletionParams{Prompt: "prompt"})
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if result.Content != "warm" {
			t.Errorf("call %d: got %q", i, result.Content)
		}
	}
}

func TestLLMActorFunc_RecordsUsage(t *testing.T) {
	provider := &stubProvider{
		response: "response",
		usage: &anyllm.Usage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
	}

	var recorded []anyllm.Usage
	recorder := func(model string, usage *anyllm.Usage) {
		if model != "test-model" {
			t.Errorf("model = %q, want test-model", model)
		}
		recorded = append(recorded, *usage)
	}

	actor := providers.NewCompleter(provider, "test-model", recorder)

	actor(context.Background(), tangle.CompletionParams{Prompt: "prompt 1"})
	actor(context.Background(), tangle.CompletionParams{Prompt: "prompt 2"})

	if len(recorded) != 2 {
		t.Fatalf("recorded %d usage entries, want 2", len(recorded))
	}
	if recorded[0].PromptTokens != 100 {
		t.Errorf("PromptTokens = %d, want 100", recorded[0].PromptTokens)
	}
	if recorded[0].CompletionTokens != 50 {
		t.Errorf("CompletionTokens = %d, want 50", recorded[0].CompletionTokens)
	}
}

func TestLLMActorFunc_NilRecorder(t *testing.T) {
	provider := &stubProvider{
		response: "ok",
		usage:    &anyllm.Usage{PromptTokens: 10},
	}

	actor := providers.NewCompleter(provider, "test-model", nil)
	result, err := actor(context.Background(), tangle.CompletionParams{Prompt: "prompt"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "ok" {
		t.Errorf("got %q, want ok", result.Content)
	}
}

func TestLLMActorFunc_ToolCalls(t *testing.T) {
	provider := &stubProvider{
		response: "I'll look in the fridge",
		toolCalls: []anyllm.ToolCall{
			{
				ID:   "call_1",
				Type: "function",
				Function: anyllm.FunctionCall{
					Name:      "look_fridge",
					Arguments: `{}`,
				},
			},
		},
	}

	actor := providers.NewCompleter(provider, "test-model", nil)
	result, err := actor(context.Background(), tangle.CompletionParams{Prompt: "find food"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "I'll look in the fridge" {
		t.Errorf("content: got %q", result.Content)
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(result.ToolCalls))
	}
	if result.ToolCalls[0].Name != "look_fridge" {
		t.Errorf("tool name: got %q, want look_fridge", result.ToolCalls[0].Name)
	}
	if result.ToolCalls[0].ID != "call_1" {
		t.Errorf("tool ID: got %q, want call_1", result.ToolCalls[0].ID)
	}
}

func TestLLMActorFunc_TokenUsage(t *testing.T) {
	provider := &stubProvider{
		response: "ok",
		usage:    &anyllm.Usage{PromptTokens: 42, CompletionTokens: 7},
	}

	actor := providers.NewCompleter(provider, "test-model", nil)
	result, _ := actor(context.Background(), tangle.CompletionParams{Prompt: "test"})

	if result.Tokens.Input != 42 {
		t.Errorf("input tokens: got %d, want 42", result.Tokens.Input)
	}
	if result.Tokens.Output != 7 {
		t.Errorf("output tokens: got %d, want 7", result.Tokens.Output)
	}
}

var _ = json.Marshal

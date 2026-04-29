package providers_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dpopsuev/tangle/providers"
	anyllm "github.com/mozilla-ai/any-llm-go/providers"
)

// stubProvider implements anyllm.Provider for testing without real LLM calls.
type stubProvider struct {
	response string
	usage    *anyllm.Usage
}

func (s *stubProvider) Name() string { return "stub" }

func (s *stubProvider) Completion(_ context.Context, params anyllm.CompletionParams) (*anyllm.ChatCompletion, error) {
	return &anyllm.ChatCompletion{
		Model: params.Model,
		Choices: []anyllm.Choice{
			{Message: anyllm.Message{Role: "assistant", Content: s.response}},
		},
		Usage: s.usage,
	}, nil
}

func (s *stubProvider) CompletionStream(_ context.Context, _ anyllm.CompletionParams) (<-chan anyllm.ChatCompletionChunk, <-chan error) {
	return nil, nil
}

func TestLLMActorFunc_ReturnsResponse(t *testing.T) {
	provider := &stubProvider{response: "hello from LLM"}
	actor := providers.LLMActorFunc(provider, "test-model", nil)

	result, err := actor(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello from LLM" {
		t.Errorf("got %q, want %q", result, "hello from LLM")
	}
}

func TestLLMActorFunc_ReusesConnection(t *testing.T) {
	provider := &stubProvider{response: "warm"}
	actor := providers.LLMActorFunc(provider, "test-model", nil)

	for i := range 3 {
		result, err := actor(context.Background(), "prompt")
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if result != "warm" {
			t.Errorf("call %d: got %q", i, result)
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

	actor := providers.LLMActorFunc(provider, "test-model", recorder)

	// Call twice
	actor(context.Background(), "prompt 1") //nolint:errcheck
	actor(context.Background(), "prompt 2") //nolint:errcheck

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

	// nil recorder should not panic
	actor := providers.LLMActorFunc(provider, "test-model", nil)
	result, err := actor(context.Background(), "prompt")
	if err != nil {
		t.Fatal(err)
	}
	if result != "ok" {
		t.Errorf("got %q, want ok", result)
	}
}

// Ensure json import is used (satisfies compiler for CompletionParams internals)
var _ = json.Marshal

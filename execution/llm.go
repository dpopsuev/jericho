package execution

import (
	"context"
	"fmt"

	anyllm "github.com/mozilla-ai/any-llm-go/providers"
)

// UsageRecorder is called after each successful LLM completion with
// normalized usage data. The provider has already unified the format —
// Vertex, OpenAI, Gemini, OpenRouter all return the same Usage type.
// The consumer wires this to billing.Tracker, Meter, or whatever.
type UsageRecorder func(model string, usage *anyllm.Usage)

// LLMActorFunc creates an ActorFunc that calls an any-llm-go Provider.
// The provider connection persists across calls — warm, not cold.
// If a UsageRecorder is provided, it receives normalized usage after each call.
func LLMActorFunc(provider anyllm.Provider, model string, recorder UsageRecorder) ActorFunc {
	return func(ctx context.Context, input string) (string, error) {
		resp, err := provider.Completion(ctx, anyllm.CompletionParams{
			Model: model,
			Messages: []anyllm.Message{
				{Role: "user", Content: input},
			},
		})
		if err != nil {
			return "", fmt.Errorf("llm completion: %w", err)
		}
		if len(resp.Choices) == 0 {
			return "", ErrNoChoices
		}

		if recorder != nil && resp.Usage != nil {
			recorder(resp.Model, resp.Usage)
		}

		content, ok := resp.Choices[0].Message.Content.(string)
		if !ok {
			return "", ErrContentNotText
		}
		return content, nil
	}
}

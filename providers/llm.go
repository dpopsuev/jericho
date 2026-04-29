package providers

import (
	"context"
	troupe "github.com/dpopsuev/tangle"
	"fmt"

	anyllm "github.com/mozilla-ai/any-llm-go/providers"
)

// UsageRecorder is called after each successful LLM completion with
// normalized usage data. The provider has already unified the format —
// Vertex, OpenAI, Gemini, OpenRouter all return the same Usage type.
// The consumer wires this to billing.Tracker, Meter, or whatever.
type UsageRecorder func(model string, usage *anyllm.Usage)

// NewCompleter creates an troupe.CompleteFunc that calls an any-llm-go Provider.
// The provider connection persists across calls — warm, not cold.
// If a UsageRecorder is provided, it receives normalized usage after each call.
func NewCompleter(provider anyllm.Provider, model string, recorder UsageRecorder) troupe.CompleteFunc {
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

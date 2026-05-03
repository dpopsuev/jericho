package providers

import (
	"context"
	"encoding/json"
	"fmt"

	troupe "github.com/dpopsuev/tangle"
	anyllm "github.com/mozilla-ai/any-llm-go/providers"
)

type UsageRecorder func(model string, usage *anyllm.Usage)

func NewCompleter(provider anyllm.Provider, model string, recorder UsageRecorder) troupe.CompleteFunc {
	return func(ctx context.Context, params troupe.CompletionParams) (*troupe.Completion, error) {
		req := anyllm.CompletionParams{
			Model: model,
			Messages: []anyllm.Message{
				{Role: "user", Content: params.Prompt},
			},
		}

		if params.MaxTokens > 0 {
			req.MaxTokens = &params.MaxTokens
		}

		for _, t := range params.Tools {
			props := make(map[string]any)
			if t.InputSchema != nil {
				json.Unmarshal(t.InputSchema, &props)
			}
			req.Tools = append(req.Tools, anyllm.Tool{
				Type: "function",
				Function: anyllm.Function{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  props,
				},
			})
		}

		resp, err := provider.Completion(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("llm completion: %w", err)
		}
		if len(resp.Choices) == 0 {
			return nil, ErrNoChoices
		}

		if recorder != nil && resp.Usage != nil {
			recorder(resp.Model, resp.Usage)
		}

		completion := &troupe.Completion{}

		if content, ok := resp.Choices[0].Message.Content.(string); ok {
			completion.Content = content
		}

		for _, tc := range resp.Choices[0].Message.ToolCalls {
			input := json.RawMessage(tc.Function.Arguments)
			completion.ToolCalls = append(completion.ToolCalls, troupe.ToolCall{
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: input,
			})
		}

		if resp.Usage != nil {
			completion.Tokens = troupe.TokenUsage{
				Input:  resp.Usage.PromptTokens,
				Output: resp.Usage.CompletionTokens,
			}
		}

		return completion, nil
	}
}

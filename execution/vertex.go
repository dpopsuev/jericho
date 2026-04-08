package execution

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/vertex"

	anyllm "github.com/mozilla-ai/any-llm-go/providers"
)

// Vertex provider constants.
const (
	vertexProviderName = "anthropic-vertex"
	vertexMaxTokens    = int64(4096)
	vertexAPIKeyDummy  = "vertex" // SDK requires non-empty key even for Vertex
)

// Anthropic content block type.
const vertexBlockTypeText = "text"

// Anthropic roles.
const (
	vertexRoleUser      = "user"
	vertexRoleAssistant = "assistant"
)

// VertexProvider implements anyllm.Provider using anthropic-sdk-go with
// Vertex AI authentication. Bypasses any-llm-go's Anthropic provider
// which doesn't support custom client options.
type VertexProvider struct {
	client *anthropic.Client
}

var _ anyllm.Provider = (*VertexProvider)(nil)

// NewVertexProvider creates a provider that routes Claude API calls
// through Google Vertex AI using Application Default Credentials.
func NewVertexProvider(ctx context.Context, region, projectID string) (*VertexProvider, error) {
	client := anthropic.NewClient(
		vertex.WithGoogleAuth(ctx, region, projectID),
		option.WithAPIKey(vertexAPIKeyDummy),
	)
	return &VertexProvider{client: &client}, nil
}

// Name returns the provider identifier.
func (v *VertexProvider) Name() string { return vertexProviderName }

// Completion sends a chat completion request via Vertex AI.
func (v *VertexProvider) Completion(ctx context.Context, params anyllm.CompletionParams) (*anyllm.ChatCompletion, error) {
	msgs := convertMessages(params.Messages)

	maxTokens := vertexMaxTokens
	if params.MaxTokens != nil && *params.MaxTokens > 0 {
		maxTokens = int64(*params.MaxTokens)
	}

	if params.Model == "" {
		return nil, fmt.Errorf("%w (resolved by Arsenal, not provider)", ErrModelRequired)
	}

	resp, err := v.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(params.Model),
		Messages:  msgs,
		MaxTokens: maxTokens,
	})
	if err != nil {
		return nil, classifyVertexError(err)
	}

	return convertResponse(resp), nil
}

// CompletionStream is not implemented — Shell Harness uses Completion().
// Streaming is TUI concern, not agent concern.
// classifyVertexError maps HTTP errors from the Anthropic SDK to sentinel errors.
func classifyVertexError(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "404") || strings.Contains(msg, "NOT_FOUND"):
		return fmt.Errorf("%w: %w", ErrModelNotFound, err)
	case strings.Contains(msg, "401") || strings.Contains(msg, "403") || strings.Contains(msg, "PERMISSION_DENIED"):
		return fmt.Errorf("%w: %w", ErrAuthFailed, err)
	case strings.Contains(msg, "429") || strings.Contains(msg, "RESOURCE_EXHAUSTED"):
		return fmt.Errorf("%w: %w", ErrQuotaExceeded, err)
	default:
		return fmt.Errorf("vertex completion: %w", err)
	}
}

func (v *VertexProvider) CompletionStream(_ context.Context, _ anyllm.CompletionParams) (<-chan anyllm.ChatCompletionChunk, <-chan error) {
	errs := make(chan error, 1)
	errs <- fmt.Errorf("%w: use Completion()", ErrStreamingNotSupported)
	close(errs)
	chunks := make(chan anyllm.ChatCompletionChunk)
	close(chunks)
	return chunks, errs
}

func convertMessages(msgs []anyllm.Message) []anthropic.MessageParam {
	out := make([]anthropic.MessageParam, 0, len(msgs))
	for _, m := range msgs {
		content, _ := m.Content.(string)
		switch m.Role {
		case vertexRoleUser:
			out = append(out, anthropic.NewUserMessage(
				anthropic.NewTextBlock(content),
			))
		case vertexRoleAssistant:
			out = append(out, anthropic.NewAssistantMessage(
				anthropic.NewTextBlock(content),
			))
		}
	}
	return out
}

func convertResponse(resp *anthropic.Message) *anyllm.ChatCompletion {
	var content string
	for _, block := range resp.Content {
		if block.Type == vertexBlockTypeText {
			content += block.Text
		}
	}

	inputTokens := int(resp.Usage.InputTokens)
	outputTokens := int(resp.Usage.OutputTokens)

	return &anyllm.ChatCompletion{
		ID:    resp.ID,
		Model: string(resp.Model),
		Choices: []anyllm.Choice{
			{
				Message: anyllm.Message{
					Role:    vertexRoleAssistant,
					Content: content,
				},
				FinishReason: string(resp.StopReason),
			},
		},
		Usage: &anyllm.Usage{
			PromptTokens:     inputTokens,
			CompletionTokens: outputTokens,
			TotalTokens:      inputTokens + outputTokens,
		},
	}
}

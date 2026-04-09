package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/vertex"

	anyllm "github.com/mozilla-ai/any-llm-go/providers"
)

// Vertex provider constants.
const (
	vertexProviderName = "anthropic-vertex"
	vertexMaxTokensFallback = int64(4096) // overridden by ConfiguredProvider
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
	start := time.Now()
	conv := VertexConverter{}
	msgs, system := convertAllMessages(conv, params.Messages)

	maxTokens := vertexMaxTokensFallback
	if params.MaxTokens != nil && *params.MaxTokens > 0 {
		maxTokens = int64(*params.MaxTokens)
	}

	if params.Model == "" {
		return nil, fmt.Errorf("%w (resolved by Arsenal, not provider)", ErrModelRequired)
	}

	slog.DebugContext(ctx, "vertex completion request",
		slog.String(logKeyModel, params.Model),
		slog.Int(logKeyMessageCount, len(params.Messages)),
		slog.Int(logKeyToolCount, len(params.Tools)),
		slog.Any(logKeyToolChoice, params.ToolChoice))

	req := anthropic.MessageNewParams{
		Model:     anthropic.Model(params.Model),
		Messages:  msgs,
		MaxTokens: maxTokens,
	}

	// Pass system prompt if extracted from messages.
	if system != "" {
		req.System = []anthropic.TextBlockParam{{Text: system}}
	}

	// Pass through Tools.
	if len(params.Tools) > 0 {
		tools := make([]anthropic.ToolUnionParam, 0, len(params.Tools))
		for _, tool := range params.Tools {
			schema := anthropic.ToolInputSchemaParam{Type: "object"}
			if tool.Function.Parameters != nil {
				if props, ok := tool.Function.Parameters["properties"]; ok {
					schema.Properties = props
				}
				if req, ok := tool.Function.Parameters["required"]; ok {
					if strs, ok := toStringSlice(req); ok {
						schema.Required = strs
					}
				}
			}
			tools = append(tools, anthropic.ToolUnionParam{
				OfTool: &anthropic.ToolParam{
					Name:        tool.Function.Name,
					Description: anthropic.String(tool.Function.Description),
					InputSchema: schema,
				},
			})
		}
		req.Tools = tools
	}

	// Pass through ToolChoice.
	if params.ToolChoice != nil {
		req.ToolChoice = convertVertexToolChoice(params.ToolChoice)
	}

	resp, err := v.client.Messages.New(ctx, req)
	elapsed := time.Since(start)
	if err != nil {
		classified := classifyVertexError(err)
		slog.ErrorContext(ctx, "vertex completion failed",
			slog.String(logKeyModel, params.Model),
			slog.Int64(logKeyElapsedMs, elapsed.Milliseconds()),
			slog.Any(logKeyError, classified))
		return nil, classified
	}

	// Log response details for debugging.
	var textBlocks, toolUseBlocks int
	for _, block := range resp.Content {
		switch block.Type {
		case vertexBlockTypeText:
			textBlocks++
		case "tool_use":
			toolUseBlocks++
		}
	}
	slog.DebugContext(ctx, "vertex completion response",
		slog.String(logKeyModel, string(resp.Model)),
		slog.Int64(logKeyElapsedMs, elapsed.Milliseconds()),
		slog.Int(logKeyBlockCount, len(resp.Content)),
		slog.Int(logKeyTextBlocks, textBlocks),
		slog.Int(logKeyToolUseBlocks, toolUseBlocks),
		slog.Int64(logKeyInputTokens, resp.Usage.InputTokens),
		slog.Int64(logKeyOutputTokens, resp.Usage.OutputTokens),
		slog.String("stop_reason", string(resp.StopReason)))

	result := convertResponse(resp)
	slog.DebugContext(ctx, "vertex completion parsed",
		slog.Int(logKeyToolCalls, len(result.Choices[0].Message.ToolCalls)),
		slog.String("finish_reason", result.Choices[0].FinishReason))

	return result, nil
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

// convertMessages is kept for backward compatibility with tests.
// New code should use convertAllMessages(converter, msgs).
func convertMessages(msgs []anyllm.Message) []anthropic.MessageParam {
	conv := VertexConverter{}
	out, _ := convertAllMessages(conv, msgs)
	return out
}

func convertResponse(resp *anthropic.Message) *anyllm.ChatCompletion {
	var content string
	var toolCalls []anyllm.ToolCall

	for _, block := range resp.Content {
		switch block.Type {
		case vertexBlockTypeText:
			content += block.Text
		case "tool_use":
			inputJSON, _ := json.Marshal(block.Input)
			toolCalls = append(toolCalls, anyllm.ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: anyllm.FunctionCall{
					Name:      block.Name,
					Arguments: string(inputJSON),
				},
			})
		}
	}

	finishReason := string(resp.StopReason)

	inputTokens := int(resp.Usage.InputTokens)
	outputTokens := int(resp.Usage.OutputTokens)

	return &anyllm.ChatCompletion{
		ID:    resp.ID,
		Model: string(resp.Model),
		Choices: []anyllm.Choice{
			{
				Message: anyllm.Message{
					Role:      vertexRoleAssistant,
					Content:   content,
					ToolCalls: toolCalls,
				},
				FinishReason: finishReason,
			},
		},
		Usage: &anyllm.Usage{
			PromptTokens:     inputTokens,
			CompletionTokens: outputTokens,
			TotalTokens:      inputTokens + outputTokens,
		},
	}
}

// convertVertexToolChoice maps anyllm ToolChoice to Anthropic format.
func convertVertexToolChoice(choice any) anthropic.ToolChoiceUnionParam {
	switch v := choice.(type) {
	case string:
		switch v {
		case "auto":
			return anthropic.ToolChoiceUnionParam{
				OfAuto: &anthropic.ToolChoiceAutoParam{},
			}
		case "none":
			return anthropic.ToolChoiceUnionParam{
				OfNone: &anthropic.ToolChoiceNoneParam{},
			}
		case "required", "any":
			return anthropic.ToolChoiceUnionParam{
				OfAny: &anthropic.ToolChoiceAnyParam{},
			}
		default:
			// Treat as tool name.
			return anthropic.ToolChoiceUnionParam{
				OfTool: &anthropic.ToolChoiceToolParam{Name: v},
			}
		}
	case anyllm.ToolChoice:
		if v.Function != nil {
			return anthropic.ToolChoiceUnionParam{
				OfTool: &anthropic.ToolChoiceToolParam{Name: v.Function.Name},
			}
		}
	}
	return anthropic.ToolChoiceUnionParam{
		OfAuto: &anthropic.ToolChoiceAutoParam{},
	}
}

// toStringSlice converts an any to []string if possible.
func toStringSlice(v any) ([]string, bool) {
	arr, ok := v.([]any)
	if !ok {
		return nil, false
	}
	strs := make([]string, 0, len(arr))
	for _, item := range arr {
		s, ok := item.(string)
		if !ok {
			return nil, false
		}
		strs = append(strs, s)
	}
	return strs, true
}

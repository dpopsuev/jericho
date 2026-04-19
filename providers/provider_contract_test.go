package providers

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	anyllm "github.com/mozilla-ai/any-llm-go/providers"
)

// ContractLevel defines which behaviors a provider must support.
type ContractLevel int

const (
	ContractText      ContractLevel = iota // L0: text completion
	ContractSystem                         // L1: system message affects behavior
	ContractToolCall                       // L2: tool_use response
	ContractToolRound                      // L3: tool_result → response (catches TRP-BUG-2)
	ContractMultiTurn                      // L4: conversation history preserved
)

// RunProviderContractSuite runs behavioral contract tests up to maxLevel.
// Each provider declares its max level. Failure at any level = hard fail.
func RunProviderContractSuite(t *testing.T, p anyllm.Provider, model string, maxLevel ContractLevel) {
	t.Helper()

	if maxLevel >= ContractText {
		t.Run("L0_TextCompletion", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			resp, err := p.Completion(ctx, anyllm.CompletionParams{
				Model:    model,
				Messages: []anyllm.Message{{Role: "user", Content: "Reply with exactly: PONG"}},
			})
			if err != nil {
				t.Fatalf("Completion: %v", err)
			}
			if len(resp.Choices) == 0 {
				t.Fatal("no choices")
			}
			text := resp.Choices[0].Message.ContentString()
			if text == "" {
				t.Fatal("empty text response")
			}
			if resp.Usage == nil {
				t.Fatal("no usage reported")
			}
			t.Logf("L0: %q (in=%d, out=%d)", text, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
		})
	}

	if maxLevel >= ContractSystem {
		t.Run("L1_SystemMessage", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			resp, err := p.Completion(ctx, anyllm.CompletionParams{
				Model: model,
				Messages: []anyllm.Message{
					{Role: "system", Content: "You must reply in French only. Never use English."},
					{Role: "user", Content: "Say hello"},
				},
			})
			if err != nil {
				t.Fatalf("Completion: %v", err)
			}
			text := strings.ToLower(resp.Choices[0].Message.ContentString())
			// Check for common French greetings
			if !strings.Contains(text, "bonjour") && !strings.Contains(text, "salut") && !strings.Contains(text, "coucou") {
				t.Fatalf("L1: expected French response, got: %q", text)
			}
			t.Logf("L1: %q", resp.Choices[0].Message.ContentString())
		})
	}

	if maxLevel >= ContractToolCall {
		t.Run("L2_ToolCall", func(t *testing.T) {
			runToolCallContract(t, p, model)
		})
	}

	if maxLevel >= ContractToolRound {
		t.Run("L3_ToolRoundTrip", func(t *testing.T) {
			runToolRoundTripContract(t, p, model)
		})
	}

	if maxLevel >= ContractMultiTurn {
		t.Run("L4_MultiTurn", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Turn 1
			r1, err := p.Completion(ctx, anyllm.CompletionParams{
				Model: model,
				Messages: []anyllm.Message{
					{Role: "user", Content: "Remember this word: FLAMINGO"},
				},
			})
			if err != nil {
				t.Fatalf("Turn 1: %v", err)
			}

			// Turn 2 — ask about what was said
			r2, err := p.Completion(ctx, anyllm.CompletionParams{
				Model: model,
				Messages: []anyllm.Message{
					{Role: "user", Content: "Remember this word: FLAMINGO"},
					{Role: "assistant", Content: r1.Choices[0].Message.ContentString()},
					{Role: "user", Content: "What word did I ask you to remember? Reply with just the word."},
				},
			})
			if err != nil {
				t.Fatalf("Turn 2: %v", err)
			}
			text := strings.ToUpper(r2.Choices[0].Message.ContentString())
			if !strings.Contains(text, "FLAMINGO") {
				t.Fatalf("L4: expected FLAMINGO in response, got: %q", r2.Choices[0].Message.ContentString())
			}
			t.Logf("L4: %q", r2.Choices[0].Message.ContentString())
		})
	}
}

// runToolCallContract is the existing L2 test (tools + forced ToolChoice).
func runToolCallContract(t *testing.T, p anyllm.Provider, model string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := p.Completion(ctx, anyllm.CompletionParams{
		Model: model,
		Messages: []anyllm.Message{
			{Role: "user", Content: "Fix the unused import: fmt is imported but not used in main.go"},
		},
		Tools: []anyllm.Tool{{
			Type: "function",
			Function: anyllm.Function{
				Name:        "apply_fix",
				Description: "Apply a code fix to a file",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"file":    map[string]any{"type": "string", "description": "file path"},
						"content": map[string]any{"type": "string", "description": "complete file content"},
					},
					"required": []any{"file", "content"},
				},
			},
		}},
		ToolChoice: anyllm.ToolChoice{
			Type:     "function",
			Function: &anyllm.ToolChoiceFunction{Name: "apply_fix"},
		},
	})
	if err != nil {
		t.Fatalf("Completion with tools: %v", err)
	}
	if len(resp.Choices) == 0 {
		t.Fatal("no choices")
	}
	if len(resp.Choices[0].Message.ToolCalls) == 0 {
		t.Fatalf("expected tool_use, got text: %v", resp.Choices[0].Message.Content)
	}
	tc := resp.Choices[0].Message.ToolCalls[0]
	if tc.Function.Name != "apply_fix" {
		t.Errorf("tool name = %q, want apply_fix", tc.Function.Name)
	}
	t.Logf("L2: tool=%s", tc.Function.Name)
}

// runToolRoundTripContract is L3: send tool_result, verify LLM uses it.
// This is THE test that catches TRP-BUG-2.
func runToolRoundTripContract(t *testing.T, p anyllm.Provider, model string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	calcTool := anyllm.Tool{
		Type: "function",
		Function: anyllm.Function{
			Name:        "calculate",
			Description: "Evaluate a math expression and return the result",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"expression": map[string]any{"type": "string", "description": "math expression"},
				},
				"required": []any{"expression"},
			},
		},
	}

	// Turn 1: force tool call
	r1, err := p.Completion(ctx, anyllm.CompletionParams{
		Model: model,
		Messages: []anyllm.Message{
			{Role: "user", Content: "What is 2 + 2? Use the calculate tool."},
		},
		Tools:      []anyllm.Tool{calcTool},
		ToolChoice: anyllm.ToolChoice{Type: "function", Function: &anyllm.ToolChoiceFunction{Name: "calculate"}},
	})
	if err != nil {
		t.Fatalf("Turn 1: %v", err)
	}
	if len(r1.Choices[0].Message.ToolCalls) == 0 {
		t.Fatalf("Turn 1: expected tool_use, got text: %v", r1.Choices[0].Message.Content)
	}

	tc := r1.Choices[0].Message.ToolCalls[0]
	t.Logf("L3 Turn 1: tool=%s args=%s", tc.Function.Name, tc.Function.Arguments)

	// Turn 2: send tool result, get final text
	r2, err := p.Completion(ctx, anyllm.CompletionParams{
		Model: model,
		Messages: []anyllm.Message{
			{Role: "user", Content: "What is 2 + 2? Use the calculate tool."},
			{
				Role:      "assistant",
				Content:   r1.Choices[0].Message.ContentString(),
				ToolCalls: r1.Choices[0].Message.ToolCalls,
			},
			{Role: "tool", ToolCallID: tc.ID, Content: "4"},
		},
		Tools: []anyllm.Tool{calcTool},
	})
	if err != nil {
		t.Fatalf("Turn 2: %v", err)
	}

	text := r2.Choices[0].Message.ContentString()
	if !strings.Contains(text, "4") {
		t.Fatalf("L3 Turn 2: expected '4' in response, got: %q", text)
	}
	if r2.Choices[0].FinishReason == anyllm.FinishReasonToolCalls {
		t.Fatal("L3 Turn 2: expected stop, got tool_calls (infinite loop)")
	}
	t.Logf("L3 Turn 2: %q", text)
}

// RunProviderContract is the legacy L2 contract test. Kept for backward compat.
// New code should use RunProviderContractSuite.
func RunProviderContract(t *testing.T, p anyllm.Provider, model string) {
	t.Helper()
	runToolCallContract(t, p, model)
}

// --- Provider-specific contract tests ---

func TestProviderContract_Vertex(t *testing.T) {
	if os.Getenv("TROUPE_TEST_LIVE_LLM") == "" {
		t.Skip("TROUPE_TEST_LIVE_LLM not set — skipping billable API test")
	}
	region := os.Getenv("GOOGLE_CLOUD_LOCATION")
	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if region == "" || project == "" {
		t.Skip("Vertex credentials not configured")
	}

	p, err := NewVertexProvider(context.Background(), region, project)
	if err != nil {
		t.Fatalf("NewVertexProvider: %v", err)
	}

	RunProviderContractSuite(t, p, "claude-sonnet-4-6", ContractToolRound)
}

func TestProviderContract_Anthropic(t *testing.T) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}
	if os.Getenv("TROUPE_TEST_LIVE_LLM") == "" {
		t.Skip("TROUPE_TEST_LIVE_LLM not set — skipping billable API test")
	}

	p, err := NewProviderByName("claude")
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}

	RunProviderContractSuite(t, p, "claude-sonnet-4-6", ContractMultiTurn)
}

// --- Offline contract tests using StubProvider ---

func TestProviderContractSuite_Stub_L0(t *testing.T) {
	p := &simpleStubProvider{response: "PONG", usage: &anyllm.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15}}
	RunProviderContractSuite(t, p, "stub", ContractText)
}

// simpleStubProvider returns a fixed text response. Internal to contract tests.
type simpleStubProvider struct {
	response string
	usage    *anyllm.Usage
}

func (s *simpleStubProvider) Name() string { return "stub" }

func (s *simpleStubProvider) Completion(_ context.Context, _ anyllm.CompletionParams) (*anyllm.ChatCompletion, error) {
	return &anyllm.ChatCompletion{
		Choices: []anyllm.Choice{{
			Message:      anyllm.Message{Role: "assistant", Content: s.response},
			FinishReason: "stop",
		}},
		Usage: s.usage,
	}, nil
}

func (s *simpleStubProvider) CompletionStream(_ context.Context, _ anyllm.CompletionParams) (<-chan anyllm.ChatCompletionChunk, <-chan error) {
	ch := make(chan anyllm.ChatCompletionChunk)
	close(ch)
	errs := make(chan error)
	close(errs)
	return ch, errs
}

// stubToolProvider supports L2 (tool calls) with canned responses.
type stubToolProvider struct {
	turn int
}

func (s *stubToolProvider) Name() string { return "stub-tool" }

func (s *stubToolProvider) Completion(_ context.Context, params anyllm.CompletionParams) (*anyllm.ChatCompletion, error) {
	s.turn++
	switch s.turn {
	case 1:
		// L0/L1: text
		return &anyllm.ChatCompletion{
			Choices: []anyllm.Choice{{
				Message:      anyllm.Message{Role: "assistant", Content: "bonjour"},
				FinishReason: "stop",
			}},
			Usage: &anyllm.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		}, nil
	case 2:
		// L1: French for system test
		return &anyllm.ChatCompletion{
			Choices: []anyllm.Choice{{
				Message:      anyllm.Message{Role: "assistant", Content: "bonjour"},
				FinishReason: "stop",
			}},
			Usage: &anyllm.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		}, nil
	case 3:
		// L2: tool call
		return &anyllm.ChatCompletion{
			Choices: []anyllm.Choice{{
				Message: anyllm.Message{
					Role: "assistant",
					ToolCalls: []anyllm.ToolCall{{
						ID: "call-1", Type: "function",
						Function: anyllm.FunctionCall{Name: "apply_fix", Arguments: `{"file":"main.go","content":"fixed"}`},
					}},
				},
				FinishReason: "tool_calls",
			}},
			Usage: &anyllm.Usage{PromptTokens: 20, CompletionTokens: 10},
		}, nil
	case 4:
		// L3: tool round-trip turn 1 (calculator)
		return &anyllm.ChatCompletion{
			Choices: []anyllm.Choice{{
				Message: anyllm.Message{
					Role: "assistant",
					ToolCalls: []anyllm.ToolCall{{
						ID: "calc-1", Type: "function",
						Function: anyllm.FunctionCall{Name: "calculate", Arguments: `{"expression":"2+2"}`},
					}},
				},
				FinishReason: "tool_calls",
			}},
			Usage: &anyllm.Usage{PromptTokens: 20, CompletionTokens: 10},
		}, nil
	case 5:
		// L3: tool round-trip turn 2 (uses result)
		// Verify tool result was in the messages
		hasTool := false
		for _, m := range params.Messages {
			if m.Role == "tool" {
				hasTool = true
			}
		}
		text := "The answer is 4"
		if !hasTool {
			text = "ERROR: no tool result received"
		}
		return &anyllm.ChatCompletion{
			Choices: []anyllm.Choice{{
				Message:      anyllm.Message{Role: "assistant", Content: text},
				FinishReason: "stop",
			}},
			Usage: &anyllm.Usage{PromptTokens: 30, CompletionTokens: 10},
		}, nil
	default:
		// L4: multi-turn
		for _, m := range params.Messages {
			if s, ok := m.Content.(string); ok && strings.Contains(strings.ToUpper(s), "FLAMINGO") {
				return &anyllm.ChatCompletion{
					Choices: []anyllm.Choice{{
						Message:      anyllm.Message{Role: "assistant", Content: "FLAMINGO"},
						FinishReason: "stop",
					}},
					Usage: &anyllm.Usage{PromptTokens: 30, CompletionTokens: 5},
				}, nil
			}
		}
		return &anyllm.ChatCompletion{
			Choices: []anyllm.Choice{{
				Message:      anyllm.Message{Role: "assistant", Content: "ok"},
				FinishReason: "stop",
			}},
			Usage: &anyllm.Usage{PromptTokens: 10, CompletionTokens: 5},
		}, nil
	}
}

func (s *stubToolProvider) CompletionStream(_ context.Context, _ anyllm.CompletionParams) (<-chan anyllm.ChatCompletionChunk, <-chan error) {
	ch := make(chan anyllm.ChatCompletionChunk)
	close(ch)
	errs := make(chan error)
	close(errs)
	return ch, errs
}

func TestProviderContractSuite_Stub_L4(t *testing.T) {
	p := &stubToolProvider{}
	RunProviderContractSuite(t, p, "stub-tool", ContractMultiTurn)
}

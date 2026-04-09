package testkit_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	anyllm "github.com/mozilla-ai/any-llm-go/providers"

	"github.com/dpopsuev/troupe/testkit"
)

func TestStubProvider_Name(t *testing.T) {
	p := testkit.NewStubProvider()
	if p.Name() != "stub" {
		t.Fatalf("Name() = %q, want stub", p.Name())
	}
}

func TestStubProvider_TextCompletion(t *testing.T) {
	p := testkit.NewStubProvider(testkit.TextResponse("hello", 10, 5))

	resp, err := p.Completion(context.Background(), anyllm.CompletionParams{Model: "test"})
	if err != nil {
		t.Fatalf("Completion: %v", err)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("choices = %d", len(resp.Choices))
	}
	if resp.Choices[0].Message.ContentString() != "hello" {
		t.Fatalf("content = %q", resp.Choices[0].Message.ContentString())
	}
	if resp.Usage.PromptTokens != 10 {
		t.Fatalf("prompt_tokens = %d", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 5 {
		t.Fatalf("completion_tokens = %d", resp.Usage.CompletionTokens)
	}
}

func TestStubProvider_ToolCallCompletion(t *testing.T) {
	p := testkit.NewStubProvider(testkit.ToolCallResponse(
		"I'll calculate",
		[]anyllm.ToolCall{testkit.ToolCall("c1", "calculator", `{"expr":"2+2"}`)},
		20, 10,
	))

	resp, err := p.Completion(context.Background(), anyllm.CompletionParams{Model: "test"})
	if err != nil {
		t.Fatalf("Completion: %v", err)
	}
	if resp.Choices[0].FinishReason != anyllm.FinishReasonToolCalls {
		t.Fatalf("finish_reason = %q", resp.Choices[0].FinishReason)
	}
	if len(resp.Choices[0].Message.ToolCalls) != 1 {
		t.Fatalf("tool_calls = %d", len(resp.Choices[0].Message.ToolCalls))
	}
	tc := resp.Choices[0].Message.ToolCalls[0]
	if tc.Function.Name != "calculator" {
		t.Fatalf("tool name = %q", tc.Function.Name)
	}
}

func TestStubProvider_CyclesResponses(t *testing.T) {
	p := testkit.NewStubProvider(
		testkit.TextResponse("first", 10, 5),
		testkit.TextResponse("second", 10, 5),
	)
	ctx := context.Background()
	params := anyllm.CompletionParams{Model: "test"}

	r1, _ := p.Completion(ctx, params)
	r2, _ := p.Completion(ctx, params)
	r3, _ := p.Completion(ctx, params) // exhausted → fallback

	if r1.Choices[0].Message.ContentString() != "first" {
		t.Fatalf("r1 = %q", r1.Choices[0].Message.ContentString())
	}
	if r2.Choices[0].Message.ContentString() != "second" {
		t.Fatalf("r2 = %q", r2.Choices[0].Message.ContentString())
	}
	if r3.Choices[0].Message.ContentString() != "(no more responses)" {
		t.Fatalf("r3 = %q, want fallback", r3.Choices[0].Message.ContentString())
	}
}

func TestStubProvider_RecordsCallLog(t *testing.T) {
	p := testkit.NewStubProvider(testkit.TextResponse("ok", 10, 5))
	ctx := context.Background()

	p.Completion(ctx, anyllm.CompletionParams{Model: "model-a"})   //nolint:errcheck // test
	p.Completion(ctx, anyllm.CompletionParams{Model: "model-b"})   //nolint:errcheck // test

	calls := p.Calls()
	if len(calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(calls))
	}
	if calls[0].Model != "model-a" {
		t.Fatalf("call[0].model = %q", calls[0].Model)
	}
	if calls[1].Model != "model-b" {
		t.Fatalf("call[1].model = %q", calls[1].Model)
	}
}

func TestStubProvider_ErrorMode(t *testing.T) {
	p := testkit.NewStubProvider()
	p.Error = errors.New("rate limit exceeded")

	_, err := p.Completion(context.Background(), anyllm.CompletionParams{Model: "test"})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "rate limit exceeded" {
		t.Fatalf("error = %q", err.Error())
	}

	// CallLog still records the call
	if len(p.Calls()) != 1 {
		t.Fatalf("calls = %d, want 1 (error calls should be logged)", len(p.Calls()))
	}
}

func TestStubProvider_ConcurrentSafe(t *testing.T) {
	var responses []*anyllm.ChatCompletion
	for range 10 {
		responses = append(responses, testkit.TextResponse("ok", 10, 5))
	}
	p := testkit.NewStubProvider(responses...)
	ctx := context.Background()

	var wg sync.WaitGroup
	errs := make(chan error, 10)

	for range 10 {
		wg.Go(func() {
			_, err := p.Completion(ctx, anyllm.CompletionParams{Model: "test"})
			if err != nil {
				errs <- err
			}
		})
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("concurrent error: %v", err)
	}
	if len(p.Calls()) != 10 {
		t.Fatalf("calls = %d, want 10", len(p.Calls()))
	}
}

func TestStubProvider_CompletionStream(t *testing.T) {
	p := testkit.NewStubProvider(testkit.TextResponse("streamed", 10, 5))

	chunks, errs := p.CompletionStream(context.Background(), anyllm.CompletionParams{Model: "test"})

	var gotChunk bool
	for c := range chunks {
		gotChunk = true
		if len(c.Choices) == 0 {
			t.Fatal("chunk has no choices")
		}
		if c.Choices[0].Delta.Content != "streamed" {
			t.Fatalf("chunk content = %q", c.Choices[0].Delta.Content)
		}
	}
	if !gotChunk {
		t.Fatal("no chunks received")
	}

	for err := range errs {
		t.Fatalf("stream error: %v", err)
	}
}

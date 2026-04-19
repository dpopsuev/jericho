package e2e_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	anyllm "github.com/mozilla-ai/any-llm-go/providers"

	"github.com/dpopsuev/troupe"
	"github.com/dpopsuev/troupe/arsenal"
	"github.com/dpopsuev/troupe/broker"
	"github.com/dpopsuev/troupe/internal/transport"
	"github.com/dpopsuev/troupe/providers"
	"github.com/dpopsuev/troupe/referee"
	"github.com/dpopsuev/troupe/signal"
	"github.com/dpopsuev/troupe/testkit"
	"github.com/dpopsuev/troupe/world"
)

const (
	eventAdmitSuccess   = "admit_success"
	eventLLMResponse    = "llm_response"
	eventTransportSend  = "transport_send"
	eventDismissSuccess = "dismiss_success"
	eventLLMError       = "llm_error"
	eventAdmitError     = "admit_error"
)

func TestE2E_RealLLM_TwoAgents_SameAdmission(t *testing.T) {
	if os.Getenv("TROUPE_TEST_LIVE_LLM") == "" {
		t.Skip("TROUPE_TEST_LIVE_LLM not set — skipping billable API test")
	}

	provider, err := providers.NewProviderFromEnv("")
	if err != nil {
		t.Fatalf("NewProviderFromEnv: %v", err)
	}

	a, err := arsenal.NewArsenal("")
	if err != nil {
		t.Fatalf("NewArsenal: %v", err)
	}
	source := os.Getenv("TROUPE_PROVIDER")
	picked, err := a.Select("", &arsenal.Preferences{
		Sources: arsenal.Filter{Allow: []string{source}},
	})
	if err != nil {
		t.Fatalf("Arsenal.Select (source=%s): %v", source, err)
	}
	model := picked.Model
	t.Logf("source: %s, model: %s (provider: %s)", source, model, picked.Provider)

	sc := referee.Scorecard{
		Name:      "real_llm_e2e",
		Threshold: 40,
		Rules: []referee.ScorecardRule{
			{On: eventAdmitSuccess, Weight: 10},
			{On: eventLLMResponse, Weight: 10},
			{On: eventTransportSend, Weight: 10},
			{On: eventDismissSuccess, Weight: 5},
			{On: eventLLMError, Weight: -50},
			{On: eventAdmitError, Weight: -50},
		},
	}

	statusLog := testkit.NewStubEventLog()
	ref := referee.New(sc)
	ref.Subscribe(statusLog)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	w := world.NewWorld()
	tr := transport.NewLocalTransport()
	bs := testkit.NewTestBusSet()

	lobby := broker.NewLobby(broker.LobbyConfig{
		World:      w,
		Transport:  tr,
		ControlLog: bs.Control,
	})

	// 1. Admit Agent A.
	idA, err := lobby.Admit(ctx, troupe.ActorConfig{Role: "summarizer"})
	if err != nil {
		statusLog.Emit(signal.Event{Kind: eventAdmitError, Source: "test"})
		t.Fatalf("Admit summarizer: %v", err)
	}
	statusLog.Emit(signal.Event{Kind: eventAdmitSuccess, Source: "agent-a"})

	// 2. Admit Agent B.
	idB, err := lobby.Admit(ctx, troupe.ActorConfig{Role: "reviewer"})
	if err != nil {
		statusLog.Emit(signal.Event{Kind: eventAdmitError, Source: "test"})
		t.Fatalf("Admit reviewer: %v", err)
	}
	statusLog.Emit(signal.Event{Kind: eventAdmitSuccess, Source: "agent-b"})

	// 3. Agent A calls the LLM.
	maxTokens := 128
	respA, err := provider.Completion(ctx, anyllm.CompletionParams{
		Model: model,
		Messages: []anyllm.Message{
			{Role: "user", Content: "Respond with exactly three words: HELLO FROM ALPHA"},
		},
		MaxTokens: &maxTokens,
	})
	if err != nil {
		statusLog.Emit(signal.Event{Kind: eventLLMError, Source: "agent-a"})
		t.Fatalf("Agent A completion: %v", err)
	}
	contentA := respA.Choices[0].Message.Content
	t.Logf("Agent A: %s", contentA)
	statusLog.Emit(signal.Event{Kind: eventLLMResponse, Source: "agent-a"})

	// 4. Agent A sends result to Agent B via Transport.
	agentBID := transport.AgentID(fmt.Sprintf("agent-%d", idB))
	agentAID := transport.AgentID(fmt.Sprintf("agent-%d", idA))
	_, err = tr.SendMessage(ctx, agentBID, transport.Message{
		From:    agentAID,
		Content: contentA.(string),
	})
	if err != nil {
		t.Fatalf("SendMessage A->B: %v", err)
	}
	statusLog.Emit(signal.Event{Kind: eventTransportSend, Source: "agent-a"})

	// 5. Agent B calls the LLM.
	respB, err := provider.Completion(ctx, anyllm.CompletionParams{
		Model: model,
		Messages: []anyllm.Message{
			{Role: "user", Content: "Respond with exactly three words: HELLO FROM BRAVO"},
		},
		MaxTokens: &maxTokens,
	})
	if err != nil {
		statusLog.Emit(signal.Event{Kind: eventLLMError, Source: "agent-b"})
		t.Fatalf("Agent B completion: %v", err)
	}
	contentB := respB.Choices[0].Message.Content
	t.Logf("Agent B: %s", contentB)
	statusLog.Emit(signal.Event{Kind: eventLLMResponse, Source: "agent-b"})

	// 6. Dismiss both.
	lobby.Dismiss(ctx, idA) //nolint:errcheck // test cleanup
	statusLog.Emit(signal.Event{Kind: eventDismissSuccess, Source: "agent-a"})
	lobby.Dismiss(ctx, idB) //nolint:errcheck // test cleanup
	statusLog.Emit(signal.Event{Kind: eventDismissSuccess, Source: "agent-b"})

	// 7. Referee verdict.
	result := ref.Result()
	t.Logf("Referee: %s score=%d/%d pass=%t events=%d",
		result.Name, result.Score, result.Threshold, result.Pass, len(result.Events))
	for kind, bucket := range result.Buckets {
		t.Logf("  %s: count=%d weight=%d", kind, bucket.Count, bucket.TotalWeight)
	}
	if !result.Pass {
		t.Fatalf("Referee FAIL: score=%d threshold=%d", result.Score, result.Threshold)
	}
}

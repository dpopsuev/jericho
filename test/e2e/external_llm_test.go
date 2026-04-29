package e2e_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	anyllm "github.com/mozilla-ai/any-llm-go/providers"

	"github.com/dpopsuev/tangle/arsenal"
	"github.com/dpopsuev/tangle/broker"
	"github.com/dpopsuev/tangle/client"
	"github.com/dpopsuev/tangle/internal/transport"
	"github.com/dpopsuev/tangle/providers"
	"github.com/dpopsuev/tangle/world"
)

func TestE2E_ExternalAgent_RealLLM_FullRoundTrip(t *testing.T) {
	if os.Getenv("TROUPE_TEST_LIVE_LLM") == "" {
		t.Skip("TROUPE_TEST_LIVE_LLM not set — skipping billable API test")
	}

	llmProvider, err := providers.NewProviderFromEnv("")
	if err != nil {
		t.Fatalf("NewProviderFromEnv: %v", err)
	}

	a, err := arsenal.NewArsenal("")
	if err != nil {
		t.Fatalf("NewArsenal: %v", err)
	}
	source := os.Getenv("TROUPE_PROVIDER")
	model := "claude-sonnet-4-6"
	if source != "" {
		filtered, ferr := a.Select("", &arsenal.Preferences{
			Sources: arsenal.Filter{Allow: []string{source}},
		})
		if ferr == nil {
			model = filtered.Model
		}
	}
	t.Logf("Using model: %s", model)

	// 1. Start external agent's A2A server.
	// This agent calls the real LLM to generate its response.
	agentTransport := transport.NewA2ATransport(a2a.AgentCard{
		Name: "llm-agent",
		URL:  "http://localhost",
	})
	defer agentTransport.Close()

	_ = agentTransport.Register("default", func(ctx context.Context, msg transport.Message) (transport.Message, error) {
		maxTokens := 64
		resp, cerr := llmProvider.Completion(ctx, anyllm.CompletionParams{
			Model: model,
			Messages: []anyllm.Message{
				{Role: "user", Content: msg.Content},
			},
			MaxTokens: &maxTokens,
		})
		if cerr != nil {
			return transport.Message{}, cerr
		}
		content, _ := resp.Choices[0].Message.Content.(string)
		return transport.Message{
			Content: content,
			Role:    transport.RoleAgent,
		}, nil
	})

	agentServer := httptest.NewServer(agentTransport.Mux())
	defer agentServer.Close()

	// 2. Start Troupe server with Lobby + admission endpoint.
	w := world.NewWorld()
	tr := transport.NewLocalTransport()
	lobby := broker.NewLobby(broker.LobbyConfig{
		World:        w,
		Transport:    tr,
		ProxyFactory: broker.A2AProxyFactory(),
	})

	mux := http.NewServeMux()
	mux.HandleFunc("POST /admission", broker.AdmissionHandler(lobby))
	troupeServer := httptest.NewServer(mux)
	defer troupeServer.Close()

	// 3. External agent registers via SDK.
	sdk := client.New(troupeServer.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	entityID, err := sdk.Register(ctx, "llm-reviewer", agentServer.URL)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	t.Logf("Registered: entity_id=%d", entityID)

	// 4. Troupe sends message to external agent.
	agentID := transport.AgentID("agent-1")
	task, err := tr.SendMessage(ctx, agentID, transport.Message{
		From:    "test",
		Content: "Respond with exactly three words: EXTERNAL AGENT WORKS",
		Role:    transport.RoleUser,
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	ch, err := tr.Subscribe(ctx, task.ID)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	var response string
	for ev := range ch {
		if ev.State == transport.TaskCompleted && ev.Data != nil {
			response = ev.Data.Content
		}
	}

	if response == "" {
		t.Fatal("expected non-empty response from external LLM agent")
	}
	t.Logf("External agent (real LLM) responded: %q", response)

	// 5. Verify agent is in World.
	if !w.Alive(world.EntityID(entityID)) {
		t.Error("agent should be alive in World")
	}

	// 6. Kick the agent.
	lobby.Kick(ctx, world.EntityID(entityID)) //nolint:errcheck // test cleanup
	if lobby.Count() != 0 {
		t.Errorf("lobby count = %d after kick, want 0", lobby.Count())
	}

	t.Logf("Full round-trip: register → message → LLM call → response → kick")
}

func TestE2E_ExternalAgent_RealLLM_TwoAgentsCollaborate(t *testing.T) {
	if os.Getenv("TROUPE_TEST_LIVE_LLM") == "" {
		t.Skip("TROUPE_TEST_LIVE_LLM not set — skipping billable API test")
	}

	llmProvider, err := providers.NewProviderFromEnv("")
	if err != nil {
		t.Fatalf("NewProviderFromEnv: %v", err)
	}

	a, err := arsenal.NewArsenal("")
	if err != nil {
		t.Fatalf("NewArsenal: %v", err)
	}
	source := os.Getenv("TROUPE_PROVIDER")
	model := "claude-sonnet-4-6"
	if source != "" {
		filtered, ferr := a.Select("", &arsenal.Preferences{
			Sources: arsenal.Filter{Allow: []string{source}},
		})
		if ferr == nil {
			model = filtered.Model
		}
	}

	makeAgent := func(name string) *httptest.Server {
		agentTr := transport.NewA2ATransport(a2a.AgentCard{Name: name, URL: "http://localhost"})
		t.Cleanup(func() { agentTr.Close() })
		_ = agentTr.Register("default", func(ctx context.Context, msg transport.Message) (transport.Message, error) {
			maxTokens := 64
			resp, cerr := llmProvider.Completion(ctx, anyllm.CompletionParams{
				Model: model,
				Messages: []anyllm.Message{
					{Role: "user", Content: msg.Content},
				},
				MaxTokens: &maxTokens,
			})
			if cerr != nil {
				return transport.Message{}, cerr
			}
			content, _ := resp.Choices[0].Message.Content.(string)
			return transport.Message{Content: content, Role: transport.RoleAgent}, nil
		})
		return httptest.NewServer(agentTr.Mux())
	}

	agentA := makeAgent("writer")
	defer agentA.Close()
	agentB := makeAgent("reviewer")
	defer agentB.Close()

	// Troupe server.
	w := world.NewWorld()
	tr := transport.NewLocalTransport()
	lobby := broker.NewLobby(broker.LobbyConfig{
		World:        w,
		Transport:    tr,
		ProxyFactory: broker.A2AProxyFactory(),
	})
	mux := http.NewServeMux()
	mux.HandleFunc("POST /admission", broker.AdmissionHandler(lobby))
	troupeServer := httptest.NewServer(mux)
	defer troupeServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Register both agents.
	sdkA := client.New(troupeServer.URL)
	idA, _ := sdkA.Register(ctx, "writer", agentA.URL)

	sdkB := client.New(troupeServer.URL)
	idB, _ := sdkB.Register(ctx, "reviewer", agentB.URL)

	t.Logf("Registered: writer=%d reviewer=%d", idA, idB)

	// Agent A writes.
	taskA, _ := tr.SendMessage(ctx, "agent-1", transport.Message{
		From: "orchestrator", Content: "Write one sentence about Go programming.", Role: transport.RoleUser,
	})
	chA, _ := tr.Subscribe(ctx, taskA.ID)
	var writerResponse string
	for ev := range chA {
		if ev.State == transport.TaskCompleted && ev.Data != nil {
			writerResponse = ev.Data.Content
		}
	}
	t.Logf("Writer: %s", writerResponse)

	// Agent B reviews what A wrote.
	taskB, _ := tr.SendMessage(ctx, "agent-2", transport.Message{
		From: "orchestrator", Content: "Review this text and say APPROVED or REJECTED: " + writerResponse, Role: transport.RoleUser,
	})
	chB, _ := tr.Subscribe(ctx, taskB.ID)
	var reviewerResponse string
	for ev := range chB {
		if ev.State == transport.TaskCompleted && ev.Data != nil {
			reviewerResponse = ev.Data.Content
		}
	}
	t.Logf("Reviewer: %s", reviewerResponse)

	if writerResponse == "" || reviewerResponse == "" {
		t.Fatal("both agents should have responded")
	}

	// Cleanup.
	lobby.Kick(ctx, world.EntityID(idA)) //nolint:errcheck // test cleanup
	lobby.Kick(ctx, world.EntityID(idB)) //nolint:errcheck // test cleanup

	t.Logf("Two external agents collaborated via real LLM through Troupe")
}

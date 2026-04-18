package broker_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/dpopsuev/troupe"
	"github.com/dpopsuev/troupe/broker"
	"github.com/dpopsuev/troupe/internal/transport"
	"github.com/dpopsuev/troupe/referee"
	"github.com/dpopsuev/troupe/signal"
	"github.com/dpopsuev/troupe/testkit"
	"github.com/dpopsuev/troupe/world"
)

func TestE2E_TwoAgents_SameAdmission_ThreeBuses(t *testing.T) {
	w := world.NewWorld()
	tr := transport.NewLocalTransport()
	bs := testkit.NewTestBusSet()

	gateCallCount := 0
	countingGate := troupe.Gate(func(_ context.Context, _ any) (bool, string, error) {
		gateCallCount++
		return true, "", nil
	})

	lobby := broker.NewLobby(broker.LobbyConfig{
		World:      w,
		Transport:  tr,
		ControlLog: bs.Control,
		Gates:      []troupe.Gate{countingGate},
		ProxyFactory: func(callbackURL string) transport.MsgHandler {
			return func(_ context.Context, msg transport.Message) (transport.Message, error) {
				return transport.Message{
					From:    "external-agent",
					Content: "proxied response from " + callbackURL,
				}, nil
			}
		},
	})

	// 1. Admit internal agent.
	internalID, err := lobby.Admit(context.Background(), troupe.ActorConfig{Role: "analyzer"})
	if err != nil {
		t.Fatalf("Admit internal: %v", err)
	}

	// 2. Admit external agent.
	externalID, err := lobby.Admit(context.Background(), troupe.ActorConfig{
		Role:        "reviewer",
		CallbackURL: "https://remote.example.com",
	})
	if err != nil {
		t.Fatalf("Admit external: %v", err)
	}

	// 3. Same gate evaluated for both.
	if gateCallCount != 2 {
		t.Fatalf("gate called %d times, want 2", gateCallCount)
	}

	// 4. Both visible in World.
	if w.Count() != 2 {
		t.Fatalf("World has %d entities, want 2", w.Count())
	}

	alive1, ok := world.TryGet[world.Alive](w, internalID)
	if !ok || alive1.State != world.AliveRunning {
		t.Fatal("internal agent should be AliveRunning")
	}
	alive2, ok := world.TryGet[world.Alive](w, externalID)
	if !ok || alive2.State != world.AliveRunning {
		t.Fatal("external agent should be AliveRunning")
	}

	// 5. Cross-agent message: internal → external via Transport.
	externalAgentID := transport.AgentID(fmt.Sprintf("agent-%d", externalID))
	resp, err := tr.Ask(context.Background(), externalAgentID, transport.Message{
		From:    transport.AgentID(fmt.Sprintf("agent-%d", internalID)),
		Content: "please review this code",
	})
	if err != nil {
		t.Fatalf("Ask external: %v", err)
	}
	if resp.Content == "" {
		t.Fatal("external agent response should not be empty")
	}

	// 6. Emit work events to WorkLog (simulating hookedActor).
	bs.Work.Emit(signal.Event{Kind: signal.EventWorkerStart, Source: "actor"})
	bs.Work.Emit(signal.Event{Kind: signal.EventWorkerDone, Source: "actor"})

	// 7. Emit status events to StatusLog (simulating warden lifecycle).
	bs.Status.Emit(signal.Event{Kind: signal.EventWorkerStarted, Source: "warden"})
	bs.Status.Emit(signal.Event{Kind: signal.EventWorkerStarted, Source: "warden"})

	// 8. Verify ControlLog: dispatch_routed x2 (both admissions).
	testkit.AssertControlEvent(t, bs, signal.EventDispatchRouted)
	if bs.Control.Len() != 2 {
		t.Fatalf("ControlLog has %d events, want 2", bs.Control.Len())
	}

	// 9. Verify WorkLog: start + done.
	testkit.AssertWorkEvent(t, bs, signal.EventWorkerStart)
	testkit.AssertWorkEvent(t, bs, signal.EventWorkerDone)
	if bs.Work.Len() != 2 {
		t.Fatalf("WorkLog has %d events, want 2", bs.Work.Len())
	}

	// 10. Verify StatusLog: worker_started x2.
	testkit.AssertStatusEvent(t, bs, signal.EventWorkerStarted)
	if bs.Status.Len() != 2 {
		t.Fatalf("StatusLog has %d events, want 2", bs.Status.Len())
	}

	// 11. Referee scores StatusLog only.
	sc := referee.Scorecard{
		Name:      "e2e",
		Threshold: 0,
		Rules: []referee.ScorecardRule{
			{On: signal.EventWorkerStarted, Weight: 10},
		},
		UnknownEventWeight: 1,
	}
	ref := referee.New(sc)
	ref.Subscribe(bs.Status)

	// Emit one more status event to trigger scoring.
	bs.Status.Emit(signal.Event{Kind: signal.EventWorkerStarted, Source: "warden"})

	result := ref.Result()
	if result.Score != 10 {
		t.Fatalf("Referee score = %d, want 10 (one worker_started on StatusLog)", result.Score)
	}

	// 12. Verify Referee did NOT score WorkLog events.
	bs.Work.Emit(signal.Event{Kind: signal.EventWorkerError, Source: "actor"})
	result2 := ref.Result()
	if result2.Score != 10 {
		t.Fatalf("Referee score = %d after WorkLog event, want still 10 (should not see WorkLog)", result2.Score)
	}

	// 13. Dismiss both.
	if err := lobby.Dismiss(context.Background(), internalID); err != nil {
		t.Fatalf("Dismiss internal: %v", err)
	}
	if err := lobby.Dismiss(context.Background(), externalID); err != nil {
		t.Fatalf("Dismiss external: %v", err)
	}
	if lobby.Count() != 0 {
		t.Fatalf("Lobby has %d entries after dismiss, want 0", lobby.Count())
	}
}


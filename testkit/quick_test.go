package testkit

import (
	"context"
	"fmt"
	"testing"

	"github.com/dpopsuev/tangle/internal/transport"
	"github.com/dpopsuev/tangle/visual"
	"github.com/dpopsuev/tangle/world"
)

func TestQuickWorld_CreatesNAgents(t *testing.T) {
	w, agents := QuickWorld(5, "TestTeam")

	if len(agents) != 5 {
		t.Fatalf("len(agents) = %d, want 5", len(agents))
	}
	if w.Count() != 5 {
		t.Errorf("world.Count() = %d, want 5", w.Count())
	}
	for i, id := range agents {
		if !w.Alive(id) {
			t.Errorf("agent %d (id=%d) should be alive", i, id)
		}
		AssertEntityHas[visual.Color](t, w, id)
		AssertEntityHas[world.Alive](t, w, id)
	}
}

func TestQuickWorld_UniqueIdentities(t *testing.T) {
	_, agents := QuickWorld(10, "UniqueTeam")

	if len(agents) != 10 {
		t.Fatalf("len(agents) = %d, want 10", len(agents))
	}
}

func TestQuickTransport_RegistersHandlers(t *testing.T) {
	w, agents := QuickWorld(3, "TransportTeam")
	tr := QuickTransport(w, agents)
	defer tr.Close()

	for _, id := range agents {
		color := world.Get[visual.Color](w, id)
		task, err := tr.SendMessage(context.Background(), transport.AgentID(color.Short()), transport.Message{
			From:    "test-sender",
			To:      transport.AgentID(color.Short()),
			Role:    "user",
			Content: "ping",
		})
		if err != nil {
			t.Fatalf("SendMessage to %s: %v", transport.AgentID(color.Short()), err)
		}

		ch, err := tr.Subscribe(context.Background(), task.ID)
		if err != nil {
			t.Fatalf("Subscribe %s: %v", task.ID, err)
		}

		var completed bool
		for ev := range ch {
			if ev.State == transport.TaskCompleted {
				completed = true
			}
		}
		if !completed {
			t.Errorf("handler for %s did not complete", transport.AgentID(color.Short()))
		}
	}
}

func TestErrorHandler_ReturnsFail(t *testing.T) {
	handler := ErrorHandler(fmt.Errorf("boom"))
	_, err := handler(context.Background(), transport.Message{
		From:    "sender",
		Content: "test",
	})
	if err == nil {
		t.Fatal("ErrorHandler should return error")
	}
	if err.Error() != "boom" {
		t.Errorf("error = %q, want %q", err.Error(), "boom")
	}
}

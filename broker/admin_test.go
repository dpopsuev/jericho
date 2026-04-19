package broker_test

import (
	"context"
	"testing"

	"github.com/dpopsuev/troupe"
	"github.com/dpopsuev/troupe/broker"
	"github.com/dpopsuev/troupe/internal/transport"
	"github.com/dpopsuev/troupe/internal/warden"
	"github.com/dpopsuev/troupe/signal"
	"github.com/dpopsuev/troupe/world"
)

type noopSupervisor struct{}

func (noopSupervisor) Start(_ context.Context, _ world.EntityID, _ warden.AgentConfig) error {
	return nil
}
func (noopSupervisor) Stop(_ context.Context, _ world.EntityID) error   { return nil }
func (noopSupervisor) Healthy(_ context.Context, _ world.EntityID) bool { return true }

func newTestAdmin(t *testing.T) (*broker.DefaultAdmin, *broker.Lobby, *world.World) {
	t.Helper()
	w := world.NewWorld()
	tr := transport.NewLocalTransport()
	buses := signal.NewBusSet()
	p := warden.NewWarden(w, tr, buses.Status, noopSupervisor{})
	lobby := broker.NewLobby(broker.LobbyConfig{
		World:      w,
		Transport:  tr,
		ControlLog: buses.Control,
	})
	admin := broker.NewAdmin(w, p, lobby, buses.Control)
	return admin, lobby, w
}

func TestAdmin_Agents_ListsAdmitted(t *testing.T) {
	admin, lobby, _ := newTestAdmin(t)
	ctx := context.Background()

	lobby.Admit(ctx, troupe.ActorConfig{Role: "worker"})   //nolint:errcheck // test setup
	lobby.Admit(ctx, troupe.ActorConfig{Role: "reviewer"}) //nolint:errcheck // test setup

	agents := admin.Agents(ctx, troupe.AgentFilter{})
	if len(agents) < 2 {
		t.Fatalf("got %d agents, want >= 2", len(agents))
	}
}

func TestAdmin_Agents_FilterByRole(t *testing.T) {
	admin, lobby, _ := newTestAdmin(t)
	ctx := context.Background()

	lobby.Admit(ctx, troupe.ActorConfig{Role: "worker"})   //nolint:errcheck // test setup
	lobby.Admit(ctx, troupe.ActorConfig{Role: "reviewer"}) //nolint:errcheck // test setup

	agents := admin.Agents(ctx, troupe.AgentFilter{Role: "worker"})
	for _, a := range agents {
		if a.Role != "worker" {
			t.Errorf("got role %q, want worker", a.Role)
		}
	}
}

func TestAdmin_Inspect(t *testing.T) {
	admin, lobby, _ := newTestAdmin(t)
	ctx := context.Background()

	id, _ := lobby.Admit(ctx, troupe.ActorConfig{Role: "inspector"})
	detail, err := admin.Inspect(ctx, id)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if detail.ID != id {
		t.Errorf("ID = %d, want %d", detail.ID, id)
	}
	if detail.Alive != world.AliveRunning {
		t.Errorf("Alive = %q, want running", detail.Alive)
	}
}

func TestAdmin_Drain_Undrain(t *testing.T) {
	admin, lobby, w := newTestAdmin(t)
	ctx := context.Background()

	id, _ := lobby.Admit(ctx, troupe.ActorConfig{Role: "drainable"})

	admin.Drain(ctx, id) //nolint:errcheck // test
	ready, _ := world.TryGet[world.Ready](w, id)
	if ready.Ready {
		t.Error("should not be ready after drain")
	}
	if ready.Reason != world.ReasonDrained {
		t.Errorf("reason = %q, want drained", ready.Reason)
	}

	admin.Undrain(ctx, id) //nolint:errcheck // test
	ready, _ = world.TryGet[world.Ready](w, id)
	if !ready.Ready {
		t.Error("should be ready after undrain")
	}
}

func TestAdmin_Cordon_BlocksAdmission(t *testing.T) {
	admin, lobby, _ := newTestAdmin(t)
	ctx := context.Background()

	gate := admin.CordonGate()
	lobbyWithGate := broker.NewLobby(broker.LobbyConfig{
		World:     world.NewWorld(),
		Transport: transport.NewLocalTransport(),
		Gates:     []troupe.Gate{gate},
	})

	admin.Cordon(ctx, "maintenance") //nolint:errcheck // test
	if !admin.IsCordoned() {
		t.Fatal("should be cordoned")
	}

	_, err := lobbyWithGate.Admit(ctx, troupe.ActorConfig{Role: "blocked"})
	if err == nil {
		t.Fatal("admission should be rejected during cordon")
	}

	admin.Uncordon(ctx) //nolint:errcheck // test
	_ = lobby           // original lobby still works
}

func TestAdmin_SetQuota(t *testing.T) {
	admin, _, _ := newTestAdmin(t)
	admin.SetQuota(context.Background(), 5) //nolint:errcheck // test
}

func TestAdmin_KillAll(t *testing.T) {
	admin, lobby, _ := newTestAdmin(t)
	ctx := context.Background()

	lobby.Admit(ctx, troupe.ActorConfig{Role: "a"}) //nolint:errcheck // test setup
	lobby.Admit(ctx, troupe.ActorConfig{Role: "b"}) //nolint:errcheck // test setup

	admin.KillAll(ctx, "emergency") //nolint:errcheck // test
}

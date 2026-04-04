package collective

import (
	"context"

	"github.com/dpopsuev/troupe/internal/agent"
	"github.com/dpopsuev/troupe/internal/transport"
	"github.com/dpopsuev/troupe/internal/warden"
	"github.com/dpopsuev/troupe/signal"
	"github.com/dpopsuev/troupe/world"
)

// testBrokerParts creates the subsystems for testing (replaces Staff).
type testBrokerParts struct {
	world     *world.World
	warden    *warden.AgentWarden
	transport *transport.LocalTransport
	bus       signal.Bus
}

func newTestParts() *testBrokerParts {
	w := world.NewWorld()
	t := transport.NewLocalTransport()
	b := signal.NewMemBus()
	p := warden.NewWarden(w, t, b, newMockDriver())
	return &testBrokerParts{world: w, warden: p, transport: t, bus: b}
}

func (tp *testBrokerParts) spawn(ctx context.Context, role string) (*agent.Solo, error) {
	id, err := tp.warden.Fork(ctx, role, warden.AgentConfig{}, 0)
	if err != nil {
		return nil, err
	}
	return agent.NewSolo(id, role, tp.world, tp.warden, tp.transport), nil
}

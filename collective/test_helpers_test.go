package collective

import (
	"context"

	"github.com/dpopsuev/tangle/internal/agent"
	"github.com/dpopsuev/tangle/internal/transport"
	"github.com/dpopsuev/tangle/internal/warden"
	"github.com/dpopsuev/tangle/signal"
	"github.com/dpopsuev/tangle/world"
)

// testBrokerParts creates the subsystems for testing (replaces Staff).
type testBrokerParts struct {
	world     *world.World
	warden    *warden.AgentWarden
	transport *transport.LocalTransport
	log       signal.EventLog
}

func newTestParts() *testBrokerParts {
	w := world.NewWorld()
	t := transport.NewLocalTransport()
	l := signal.NewMemLog()
	p := warden.NewWarden(w, t, l, newMockDriver())
	return &testBrokerParts{world: w, warden: p, transport: t, log: l}
}

func (tp *testBrokerParts) spawn(ctx context.Context, role string) (*agent.Solo, error) {
	id, err := tp.warden.Fork(ctx, role, warden.AgentConfig{}, 0)
	if err != nil {
		return nil, err
	}
	return agent.NewSolo(id, role, tp.world, tp.warden, tp.transport), nil
}

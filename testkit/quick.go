package testkit

import (
	"fmt"

	"github.com/dpopsuev/jericho/identity"
	"github.com/dpopsuev/jericho/internal/transport"
	"github.com/dpopsuev/jericho/world"
)

// QuickWorld creates a World with n agents, each with ColorIdentity + Health.
// Returns the World and slice of EntityIDs.
func QuickWorld(n int, collective string) (*world.World, []world.EntityID) {
	w := world.NewWorld()
	reg := identity.NewRegistry()
	strategy := identity.NewDefaultStrategy(w, reg)

	agents := make([]world.EntityID, 0, n)
	for i := range n {
		id, err := strategy.Resolve(fmt.Sprintf("agent-%d", i), collective)
		if err != nil {
			panic(fmt.Sprintf("testkit: QuickWorld: %v", err))
		}
		agents = append(agents, id)
	}
	return w, agents
}

// QuickTransport creates a LocalTransport with EchoHandlers for all agents.
func QuickTransport(w *world.World, agents []world.EntityID) *transport.LocalTransport {
	tr := transport.NewLocalTransport()
	for _, id := range agents {
		color := world.Get[identity.Color](w, id)
		tr.Register(transport.AgentID(color.Short()), EchoHandler())
	}
	return tr
}

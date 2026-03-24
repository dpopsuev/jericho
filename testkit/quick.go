package testkit

import (
	"fmt"

	"github.com/dpopsuev/bugle"
	"github.com/dpopsuev/bugle/transport"
)

// QuickWorld creates a World with n agents, each with ColorIdentity + Health.
// Returns the World and slice of EntityIDs.
func QuickWorld(n int, collective string) (*bugle.World, []bugle.EntityID) {
	world := bugle.NewWorld()
	reg := bugle.NewRegistry()
	strategy := bugle.NewDefaultStrategy(world, reg)

	agents := make([]bugle.EntityID, 0, n)
	for i := range n {
		id, err := strategy.Resolve(fmt.Sprintf("agent-%d", i), collective)
		if err != nil {
			panic(fmt.Sprintf("testkit: QuickWorld: %v", err))
		}
		agents = append(agents, id)
	}
	return world, agents
}

// QuickTransport creates a LocalTransport with EchoHandlers for all agents.
func QuickTransport(world *bugle.World, agents []bugle.EntityID) *transport.LocalTransport {
	tr := transport.NewLocalTransport()
	for _, id := range agents {
		color := bugle.Get[bugle.ColorIdentity](world, id)
		tr.Register(color.Short(), EchoHandler())
	}
	return tr
}

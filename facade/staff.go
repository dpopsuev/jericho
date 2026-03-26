package facade

import (
	"context"
	"fmt"

	"github.com/dpopsuev/bugle/pool"
	"github.com/dpopsuev/bugle/signal"
	"github.com/dpopsuev/bugle/transport"
	"github.com/dpopsuev/bugle/world"
)

// Staff is the top-level entry point for the Bugle facade. It wires
// together World, AgentPool, LocalTransport, and signal Bus, then
// exposes a human-friendly API via AgentHandle.
type Staff struct {
	world     *world.World
	pool      *pool.AgentPool
	transport *transport.LocalTransport
	bus       signal.Bus
}

// NewStaff creates a fully-wired Staff with the given Launcher.
func NewStaff(launcher pool.Launcher) *Staff {
	w := world.NewWorld()
	t := transport.NewLocalTransport()
	b := signal.NewMemBus()
	p := pool.New(w, t, b, launcher)
	return &Staff{world: w, pool: p, transport: t, bus: b}
}

// Spawn creates a root agent (parentID == 0) and returns its handle.
func (s *Staff) Spawn(ctx context.Context, role string, config pool.LaunchConfig) (*AgentHandle, error) {
	id, err := s.pool.Fork(ctx, role, config, 0)
	if err != nil {
		return nil, err
	}
	return s.handleFor(id, role), nil
}

// SpawnUnder creates a child agent under the given parent.
func (s *Staff) SpawnUnder(ctx context.Context, parent *AgentHandle, role string, config pool.LaunchConfig) (*AgentHandle, error) {
	id, err := s.pool.Fork(ctx, role, config, parent.ID())
	if err != nil {
		return nil, err
	}
	return s.handleFor(id, role), nil
}

// SetSubreaper registers an agent as the orphan adopter.
func (s *Staff) SetSubreaper(agent *AgentHandle) {
	s.pool.SetSubreaper(agent.ID())
}

// KillAll stops all running agents.
func (s *Staff) KillAll(ctx context.Context) {
	s.pool.KillAll(ctx)
}

// Active returns handles for all running (non-zombie) agents.
func (s *Staff) Active() []*AgentHandle {
	ids := s.pool.Active()
	handles := make([]*AgentHandle, 0, len(ids))
	for _, id := range ids {
		role := s.transport.Roles().RoleOf(agentTransportID(id))
		handles = append(handles, s.handleFor(id, role))
	}
	return handles
}

// Count returns the number of running (non-zombie) agents.
func (s *Staff) Count() int {
	return s.pool.Count()
}

// FindByRole returns handles for all agents with the given role.
func (s *Staff) FindByRole(role string) []*AgentHandle {
	agentIDs := s.transport.Roles().AgentsForRole(role)
	handles := make([]*AgentHandle, 0, len(agentIDs))
	for _, aid := range agentIDs {
		// Parse entity ID from transport ID "agent-N".
		var eid world.EntityID
		if _, err := fmt.Sscanf(aid, "agent-%d", &eid); err != nil {
			continue
		}
		handles = append(handles, s.handleFor(eid, role))
	}
	return handles
}

// Tree returns the hierarchical process tree rooted at the given agent.
func (s *Staff) Tree(root *AgentHandle) *pool.TreeNode {
	return s.pool.Tree(root.ID())
}

// OnSignal registers a callback that fires on every signal emission.
func (s *Staff) OnSignal(fn func(signal.Signal)) {
	s.bus.OnEmit(fn)
}

// TreeFull returns the full hierarchical tree rooted at the given agent,
// including agents hidden inside FacadeAgent collectives.
// Tree() shows the collapsed view (facades as single nodes).
// TreeFull() shows every real agent.
func (s *Staff) TreeFull(root *AgentHandle) *pool.TreeNode {
	return s.pool.Tree(root.ID())
}

// ---------------------------------------------------------------------------
// Escape hatches — for advanced consumers who need the raw subsystems.
// ---------------------------------------------------------------------------

// World returns the underlying ECS world.
func (s *Staff) World() *world.World { return s.world }

// Pool returns the underlying AgentPool.
func (s *Staff) Pool() *pool.AgentPool { return s.pool }

// Transport returns the underlying LocalTransport.
func (s *Staff) Transport() *transport.LocalTransport { return s.transport }

// Bus returns the underlying signal Bus.
func (s *Staff) Bus() signal.Bus { return s.bus }

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// handleFor constructs an AgentHandle with all subsystem references.
func (s *Staff) handleFor(id world.EntityID, role string) *AgentHandle {
	return &AgentHandle{
		id:        id,
		role:      role,
		world:     s.world,
		pool:      s.pool,
		transport: s.transport,
	}
}

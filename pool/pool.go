// Package pool manages agent process lifecycles with Linux-inspired
// process supervision: parent-child tracking, zombie reaping, orphan
// adoption. Maps Bugle World entities to running processes via the
// AgentSupervisor interface. Process-agnostic: consumers inject their own AgentSupervisor.
package pool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dpopsuev/jericho/signal"
	"github.com/dpopsuev/jericho/transport"
	"github.com/dpopsuev/jericho/world"
)

// Sentinel errors.
var (
	ErrNotFound = errors.New("agent not found")
	ErrNotOwner = errors.New("caller is not the parent of this agent")
)

// agentEntry tracks a running or zombie agent.
type agentEntry struct {
	ID       world.EntityID
	ParentID world.EntityID // 0 = root agent (no parent)
	Role     string
	Config   AgentConfig
	Started  time.Time
	ExitCode ExitCode  // set when agent finishes
	ExitTime time.Time // zero = still running
}

// AgentPool manages agent process lifecycles with process supervision.
type AgentPool struct {
	world     *world.World
	transport *transport.LocalTransport
	bus       signal.Bus
	launcher  AgentSupervisor
	mu        sync.RWMutex
	agents    map[world.EntityID]*agentEntry   // running agents
	zombies   map[world.EntityID]*agentEntry   // finished but not reaped
	subreaper world.EntityID                   // orphan adopter (0 = pool-level)
	autoReap  map[world.EntityID]bool          // parents with auto-reap enabled
	waitCh    map[world.EntityID]chan struct{} // notify Wait() callers
}

// New creates an AgentPool.
func New(w *world.World, t *transport.LocalTransport, b signal.Bus, l AgentSupervisor) *AgentPool {
	return &AgentPool{
		world:     w,
		transport: t,
		bus:       b,
		launcher:  l,
		agents:    make(map[world.EntityID]*agentEntry),
		zombies:   make(map[world.EntityID]*agentEntry),
		autoReap:  map[world.EntityID]bool{0: true}, // root parent auto-reaps by default
		waitCh:    make(map[world.EntityID]chan struct{}),
	}
}

// Fork spawns a new agent with parent tracking: creates entity, attaches
// components, starts process, registers in transport, emits signal.
// parentID=0 means root agent (no parent).
func (p *AgentPool) Fork(ctx context.Context, role string, config AgentConfig, parentID world.EntityID) (world.EntityID, error) {
	// 1. Create entity.
	id := p.world.Spawn()

	// 2. Attach components.
	world.Attach(p.world, id, world.Alive{State: world.AliveRunning, Since: time.Now()})
	world.Attach(p.world, id, world.Ready{Ready: true, LastSeen: time.Now()})
	world.Attach(p.world, id, world.Hierarchy{Parent: parentID})
	if config.Budget > 0 {
		world.Attach(p.world, id, world.Budget{Ceiling: config.Budget})
	}

	// 3. Start process via launcher.
	if err := p.launcher.Start(ctx, id, config); err != nil {
		p.world.Despawn(id)
		return 0, fmt.Errorf("fork %s: %w", role, err)
	}

	// 4. Register in transport.
	agentID := agentTransportID(id)
	p.transport.Register(agentID, func(ctx context.Context, msg transport.Message) (transport.Message, error) {
		return transport.Message{From: agentID, Content: "ack"}, nil
	})
	p.transport.Roles().Register(agentID, role)

	// 5. Track with parent.
	p.mu.Lock()
	p.agents[id] = &agentEntry{
		ID:       id,
		ParentID: parentID,
		Role:     role,
		Config:   config,
		Started:  time.Now(),
	}
	// Prepare wait channel so Wait() can block.
	p.waitCh[id] = make(chan struct{})
	p.mu.Unlock()

	// 6. Emit signal with parent info.
	meta := map[string]string{
		signal.MetaKeyWorkerID: agentID,
		"role":                 role,
	}
	if parentID > 0 {
		meta["parent"] = agentTransportID(parentID)
	}
	p.bus.Emit(&signal.Signal{
		Timestamp: time.Now().Format(time.RFC3339),
		Event:     signal.EventWorkerStarted,
		Agent:     signal.AgentWorker,
		Meta:      meta,
	})

	return id, nil
}

// Kill stops an agent: stops process, moves to zombie state.
// The entry is NOT removed — parent must call Wait() to reap.
// If parent has AutoReap, the entry is removed immediately.
func (p *AgentPool) Kill(ctx context.Context, id world.EntityID) error {
	p.mu.Lock()
	entry, ok := p.agents[id]
	if !ok {
		p.mu.Unlock()
		return fmt.Errorf("%w: %d", ErrNotFound, id)
	}

	// Reparent orphans before removing parent.
	p.reparentOrphansLocked(id)

	// Move from agents → zombies.
	delete(p.agents, id)
	entry.ExitTime = time.Now()
	// ExitCode may already be set by KillWithCode. Don't overwrite.

	shouldAutoReap := p.autoReap[entry.ParentID]
	ch := p.waitCh[id]

	if !shouldAutoReap {
		p.zombies[id] = entry
	} else {
		delete(p.waitCh, id)
	}
	p.mu.Unlock()

	// Stop process.
	if err := p.launcher.Stop(ctx, id); err != nil {
		_ = err // log but continue cleanup
	}

	// Unregister transport.
	agentID := agentTransportID(id)
	p.transport.Roles().Unregister(agentID)
	p.transport.Unregister(agentID)

	// Update liveness — BEFORE notifying Wait() callers, because reap()
	// calls Despawn() which would make this Attach panic on a dead entity.
	world.Attach(p.world, id, world.Alive{State: world.AliveTerminated, ExitedAt: time.Now()})
	world.Attach(p.world, id, world.Ready{Ready: false, LastSeen: time.Now(), Reason: "terminated"})

	// Emit signal.
	p.bus.Emit(&signal.Signal{
		Timestamp: time.Now().Format(time.RFC3339),
		Event:     signal.EventWorkerStopped,
		Agent:     signal.AgentWorker,
		Meta: map[string]string{
			signal.MetaKeyWorkerID: agentID,
			"role":                 entry.Role,
		},
	})

	// Only despawn if auto-reaped (not zombie).
	if shouldAutoReap {
		p.world.Despawn(id)
	}

	// Notify Wait() callers — LAST, after all cleanup is done.
	// Wait() → reap() → Despawn() is safe because Health is already set.
	if ch != nil {
		close(ch)
	}

	return nil
}

// KillAll stops all running agents. Called on shutdown.
func (p *AgentPool) KillAll(ctx context.Context) {
	p.mu.RLock()
	ids := make([]world.EntityID, 0, len(p.agents))
	for id := range p.agents {
		ids = append(ids, id)
	}
	p.mu.RUnlock()

	for _, id := range ids {
		p.Kill(ctx, id) //nolint:errcheck // best-effort cleanup during shutdown
	}

	// Also clean up any remaining zombies.
	p.mu.Lock()
	for id := range p.zombies {
		p.world.Despawn(id)
	}
	p.zombies = make(map[world.EntityID]*agentEntry)
	p.mu.Unlock()
}

// Active returns all running (non-zombie) entity IDs.
func (p *AgentPool) Active() []world.EntityID {
	p.mu.RLock()
	defer p.mu.RUnlock()
	ids := make([]world.EntityID, 0, len(p.agents))
	for id := range p.agents {
		ids = append(ids, id)
	}
	return ids
}

// Count returns the number of running (non-zombie) agents.
func (p *AgentPool) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.agents)
}

// ZombieCount returns the number of zombie agents awaiting reaping.
func (p *AgentPool) ZombieCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.zombies)
}

// get returns the entry for a running agent.
func (p *AgentPool) get(id world.EntityID) (*agentEntry, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	e, ok := p.agents[id]
	return e, ok
}

func agentTransportID(id world.EntityID) string {
	return fmt.Sprintf("agent-%d", id)
}

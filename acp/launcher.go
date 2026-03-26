// launcher.go — ACPLauncher implements pool.Launcher.
//
// Spawns ACP agent processes — one process per entity.
// Consumers get a pool.Launcher that manages ACP-backed agents.
package acp

import (
	"context"
	"fmt"
	"sync"

	"github.com/dpopsuev/bugle/pool"
	"github.com/dpopsuev/bugle/world"
)

// ACPLauncher implements pool.Launcher by spawning ACP agent processes.
type ACPLauncher struct {
	mu      sync.RWMutex
	clients map[world.EntityID]*Client
}

// NewACPLauncher creates a launcher for ACP-based agents.
func NewACPLauncher() *ACPLauncher {
	return &ACPLauncher{
		clients: make(map[world.EntityID]*Client),
	}
}

// Start spawns an ACP agent process for the given entity.
func (l *ACPLauncher) Start(ctx context.Context, id world.EntityID, config pool.LaunchConfig) error {
	agentName := "cursor" // default
	if config.Model != "" {
		agentName = config.Model
	}

	client, err := NewClient(agentName,
		WithModel(config.Model),
	)
	if err != nil {
		return fmt.Errorf("create ACP client for entity %d: %w", id, err)
	}

	if err := client.Start(ctx); err != nil {
		return fmt.Errorf("start ACP agent for entity %d: %w", id, err)
	}

	l.mu.Lock()
	l.clients[id] = client
	l.mu.Unlock()

	return nil
}

// Stop kills the ACP agent process for the given entity.
func (l *ACPLauncher) Stop(ctx context.Context, id world.EntityID) error {
	l.mu.Lock()
	client, ok := l.clients[id]
	if ok {
		delete(l.clients, id)
	}
	l.mu.Unlock()

	if !ok {
		return nil
	}
	return client.Stop(ctx)
}

// Healthy returns true if the ACP agent process is still running.
// Checks both map presence and actual process state.
func (l *ACPLauncher) Healthy(_ context.Context, id world.EntityID) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	client, ok := l.clients[id]
	if !ok {
		return false
	}
	return client.ProcessAlive()
}

// Client returns the ACP Client for an entity (for sending messages).
func (l *ACPLauncher) Client(id world.EntityID) (*Client, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	c, ok := l.clients[id]
	return c, ok
}

// Compile-time check.
var _ pool.Launcher = (*ACPLauncher)(nil)

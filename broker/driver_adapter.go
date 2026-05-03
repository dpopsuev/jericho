package broker

import (
	"context"
	"fmt"
	"sync"

	tangle "github.com/dpopsuev/tangle"
	"github.com/dpopsuev/tangle/internal/warden"
	"github.com/dpopsuev/tangle/world"
)

// multiDriverAdapter wraps public Drivers as a warden.AgentSupervisor.
// Resolves the correct driver at Start() time based on a per-entity provider map.
type multiDriverAdapter struct {
	defaultDriver tangle.Driver
	drivers       map[string]tangle.Driver
	providers     map[world.EntityID]string // entity → provider, set before Fork
	mu            sync.Mutex
}

func (a *multiDriverAdapter) setProvider(id world.EntityID, provider string) {
	a.mu.Lock()
	a.providers[id] = provider
	a.mu.Unlock()
}

func (a *multiDriverAdapter) resolve(id world.EntityID) tangle.Driver {
	a.mu.Lock()
	provider := a.providers[id]
	a.mu.Unlock()
	if provider != "" && a.drivers != nil {
		if d, ok := a.drivers[provider]; ok {
			return d
		}
	}
	return a.defaultDriver
}

func (a *multiDriverAdapter) Start(ctx context.Context, id world.EntityID, config warden.AgentConfig) error {
	drv := a.defaultDriver
	if config.Provider != "" && a.drivers != nil {
		if d, ok := a.drivers[config.Provider]; ok {
			drv = d
		}
	}
	if drv == nil {
		return fmt.Errorf("no driver for entity %d: %w", id, tangle.ErrNoDriver)
	}
	a.setProvider(id, config.Provider)
	return drv.Start(ctx, id, tangle.AgentConfig{Model: config.Model, Role: config.Role, Provider: config.Provider})
}

func (a *multiDriverAdapter) Stop(ctx context.Context, id world.EntityID) error {
	drv := a.resolve(id)
	if drv == nil && a.defaultDriver != nil {
		drv = a.defaultDriver
	}
	if drv == nil {
		return nil
	}
	return drv.Stop(ctx, id)
}

func (a *multiDriverAdapter) Healthy(_ context.Context, _ world.EntityID) bool {
	return true
}

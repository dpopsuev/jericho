package troupe

import (
	"context"
	"fmt"
	"strings"

	"github.com/dpopsuev/troupe/identity"
	"github.com/dpopsuev/troupe/internal/acp"
	"github.com/dpopsuev/troupe/internal/agent"
	"github.com/dpopsuev/troupe/internal/transport"
	"github.com/dpopsuev/troupe/internal/warden"
	"github.com/dpopsuev/troupe/signal"
	"github.com/dpopsuev/troupe/world"
)

// driverAdapter wraps a public Driver as a warden.AgentSupervisor.
type driverAdapter struct {
	driver Driver
}

func (a *driverAdapter) Start(ctx context.Context, id world.EntityID, config warden.AgentConfig) error {
	return a.driver.Start(ctx, id, ActorConfig{Model: config.Model, Role: config.Role})
}

func (a *driverAdapter) Stop(ctx context.Context, id world.EntityID) error {
	return a.driver.Stop(ctx, id)
}

func (a *driverAdapter) Healthy(_ context.Context, _ world.EntityID) bool {
	return true // default: driver-managed agents are healthy
}

// DefaultBroker is the standard Broker implementation. Wires World, Warden,
// Transport, Driver, Registry, and Signal Bus internally.
type DefaultBroker struct {
	world     *world.World
	warden    *warden.AgentWarden
	transport *transport.LocalTransport
	bus       signal.Bus
	registry  *identity.Registry
	hooks     []Hook
	driver    Driver // original driver (for optional interface checks)
	meter     Meter
}

// BrokerOption configures a DefaultBroker.
type BrokerOption func(*brokerConfig)

type brokerConfig struct {
	driver       Driver
	hooks        []Hook
	pickStrategy PickStrategy
	meter        Meter
}

// WithDriver sets the agent driver. Default: ACP (subprocess + JSON-RPC).
func WithDriver(d Driver) BrokerOption {
	return func(c *brokerConfig) { c.driver = d }
}

// WithHook registers a lifecycle hook. Nil hooks are ignored.
func WithHook(h Hook) BrokerOption {
	return func(c *brokerConfig) {
		if h != nil {
			c.hooks = append(c.hooks, h)
		}
	}
}

// WithMeter sets the resource usage meter. Default: none.
func WithMeter(m Meter) BrokerOption {
	return func(c *brokerConfig) { c.meter = m }
}

// NewBroker creates a Broker. If the endpoint is a remote URL (https://),
// returns a RemoteBroker that proxies over HTTP. Otherwise, returns a
// local DefaultBroker. Default driver: ACP.
func NewBroker(endpoint string, opts ...BrokerOption) Broker {
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return newRemoteBroker(endpoint)
	}
	return newLocalBroker(opts...)
}

// newLocalBroker creates an in-process DefaultBroker.
func newLocalBroker(opts ...BrokerOption) *DefaultBroker {
	cfg := &brokerConfig{}
	for _, o := range opts {
		o(cfg)
	}

	// Resolve the warden supervisor: use custom Driver adapter or default ACP.
	var supervisor warden.AgentSupervisor
	if cfg.driver != nil {
		supervisor = &driverAdapter{driver: cfg.driver}
	} else {
		supervisor = acp.NewACPLauncher()
	}

	w := world.NewWorld()
	t := transport.NewLocalTransport()
	b := signal.NewMemBus()
	p := warden.NewWarden(w, t, b, supervisor)

	return &DefaultBroker{
		world:     w,
		warden:    p,
		transport: t,
		bus:       b,
		registry:  identity.NewRegistry(),
		hooks:     cfg.hooks,
		driver:    cfg.driver,
		meter:     cfg.meter,
	}
}

// Pick returns actor configs matching preferences.
func (b *DefaultBroker) Pick(_ context.Context, prefs Preferences) ([]ActorConfig, error) {
	count := prefs.Count
	if count <= 0 {
		count = 1
	}

	configs := make([]ActorConfig, count)
	for i := range count {
		configs[i] = ActorConfig{
			Model: prefs.Model,
			Role:  prefs.Role,
		}
	}
	return configs, nil
}

// Spawn creates a running actor.
func (b *DefaultBroker) Spawn(ctx context.Context, config ActorConfig) (Actor, error) {
	// Driver environment validation (optional interface).
	if b.driver != nil {
		if v, ok := b.driver.(DriverValidator); ok {
			if err := v.ValidateEnvironment(ctx); err != nil {
				return nil, fmt.Errorf("driver validate: %w", err)
			}
		}
	}

	// Pre-spawn hooks: any SpawnHook can reject.
	for _, h := range b.hooks {
		if sh, ok := h.(SpawnHook); ok {
			if err := sh.PreSpawn(ctx, config); err != nil {
				return nil, fmt.Errorf("hook %s pre-spawn: %w", sh.Name(), err)
			}
		}
	}

	role := config.Role
	if role == "" {
		role = "actor"
	}

	id, err := b.warden.Fork(ctx, role, warden.AgentConfig{
		Model: config.Model,
	}, 0)

	var actor Actor
	if err == nil {
		actor = agent.NewSolo(id, role, b.world, b.warden, b.transport)
	}

	// Post-spawn hooks: observe result.
	for _, h := range b.hooks {
		if sh, ok := h.(SpawnHook); ok {
			sh.PostSpawn(ctx, config, actor, err)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("broker spawn: %w", err)
	}

	// Wrap with perform hooks if any are registered.
	var performHooks []PerformHook
	for _, h := range b.hooks {
		if ph, ok := h.(PerformHook); ok {
			performHooks = append(performHooks, ph)
		}
	}
	if len(performHooks) > 0 {
		actor = newHookedActor(actor, performHooks)
	}

	return actor, nil
}

// Meter returns the resource usage meter (nil if none configured).
func (b *DefaultBroker) Meter() Meter { return b.meter }

// Signal returns the event bus.
func (b *DefaultBroker) Signal() signal.Bus { return b.bus }

// World returns the underlying ECS world (for advanced consumers).
func (b *DefaultBroker) World() *world.World { return b.world }

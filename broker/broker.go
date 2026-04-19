package broker

import (
	"context"
	"fmt"

	troupe "github.com/dpopsuev/troupe"
	"github.com/dpopsuev/troupe/visual"
	"github.com/dpopsuev/troupe/internal/acp"
	"github.com/dpopsuev/troupe/internal/agent"
	"github.com/dpopsuev/troupe/internal/transport"
	"github.com/dpopsuev/troupe/internal/warden"
	"github.com/dpopsuev/troupe/signal"
	"github.com/dpopsuev/troupe/world"
)

// DefaultBroker is the standard Broker implementation. Wires World, Warden,
// Transport, Driver, Registry, and Signal Bus internally.
type DefaultBroker struct {
	world        *world.World
	warden       *warden.AgentWarden
	transport    transport.Transport
	buses        signal.BusSet
	controlLog   signal.EventLog
	registry     *visual.Registry
	hooks        []Hook
	driver       troupe.Driver // default driver (for optional interface checks)
	adapter      *multiDriverAdapter
	meter        troupe.Meter
	spawnGate    troupe.Gate
	performGate  troupe.Gate
	admission    troupe.Admission
}

// Option configures a DefaultBroker.
type Option func(*config)

type config struct {
	driver       troupe.Driver
	drivers      map[string]troupe.Driver // provider → driver
	hooks        []Hook
	pickStrategy PickStrategy
	meter        troupe.Meter
	transport    transport.Transport
	spawnGates   []troupe.Gate
	performGates []troupe.Gate
	controlLog   signal.EventLog
	admission    troupe.Admission
}

// WithDriver sets the agent driver. Default: ACP (subprocess + JSON-RPC).
func WithDriver(d troupe.Driver) Option {
	return func(c *config) { c.driver = d }
}

// WithHook registers a lifecycle hook. Nil hooks are ignored.
func WithHook(h Hook) Option {
	return func(c *config) {
		if h != nil {
			c.hooks = append(c.hooks, h)
		}
	}
}

// WithDriverFor registers a driver for a specific provider.
// Broker.Spawn routes to the matching driver based on ActorConfig.Provider.
func WithDriverFor(provider string, d troupe.Driver) Option {
	return func(c *config) {
		if c.drivers == nil {
			c.drivers = make(map[string]troupe.Driver)
		}
		c.drivers[provider] = d
	}
}

// WithTransport sets the agent transport. Default: LocalTransport (in-process).
// Use NewHTTPTransport() for cross-process A2A communication.
func WithTransport(t transport.Transport) Option {
	return func(c *config) { c.transport = t }
}

// WithMeter sets the resource usage meter. Default: none.
func WithMeter(m troupe.Meter) Option {
	return func(c *config) { c.meter = m }
}

// WithControlLog sets the control bus for routing decision events.
func WithControlLog(l signal.EventLog) Option {
	return func(c *config) { c.controlLog = l }
}

// WithAdmission sets the agent admission system. When set, Broker.Spawn
// uses Admission.Admit for entity creation instead of Warden.Fork.
func WithAdmission(a troupe.Admission) Option {
	return func(c *config) { c.admission = a }
}

// WithSpawnGate adds a Gate that must pass before Broker.Spawn proceeds.
// Multiple gates compose with short-circuit AND (first rejection stops).
func WithSpawnGate(g troupe.Gate) Option {
	return func(c *config) { c.spawnGates = append(c.spawnGates, g) }
}

// WithPerformGate adds a Gate that must pass before Actor.Perform proceeds.
// Multiple gates compose with short-circuit AND (first rejection stops).
func WithPerformGate(g troupe.Gate) Option {
	return func(c *config) { c.performGates = append(c.performGates, g) }
}

// New creates a local Broker. The endpoint parameter is reserved for
// future use — pass empty string. Use A2A (GOL-29/30) for remote agents.
func New(endpoint string, opts ...Option) troupe.Broker {
	return newLocalBroker(opts...)
}

// newLocalBroker creates an in-process DefaultBroker.
func newLocalBroker(opts ...Option) *DefaultBroker {
	cfg := &config{}
	for _, o := range opts {
		o(cfg)
	}

	// Resolve the warden supervisor: multi-driver adapter or default ACP.
	adapter := &multiDriverAdapter{
		defaultDriver: cfg.driver,
		drivers:       cfg.drivers,
		providers:     make(map[world.EntityID]string),
	}
	var supervisor warden.AgentSupervisor
	if cfg.driver != nil || len(cfg.drivers) > 0 {
		supervisor = adapter
	} else {
		supervisor = acp.NewACPLauncher()
	}

	w := world.NewWorld()
	var t transport.Transport
	if cfg.transport != nil {
		t = cfg.transport
	} else {
		t = transport.NewLocalTransport()
	}
	buses := signal.NewBusSet()
	p := warden.NewWarden(w, t, buses.Status, supervisor)

	reg := visual.NewRegistry()
	p.SetRegistry(reg)

	var spawnGate troupe.Gate
	if len(cfg.spawnGates) > 0 {
		spawnGate = troupe.ComposeGates(cfg.spawnGates...)
	}
	var performGate troupe.Gate
	if len(cfg.performGates) > 0 {
		performGate = troupe.ComposeGates(cfg.performGates...)
	}

	return &DefaultBroker{
		world:       w,
		warden:      p,
		transport:   t,
		buses:       buses,
		controlLog:  cfg.controlLog,
		registry:    reg,
		hooks:       cfg.hooks,
		driver:      cfg.driver,
		adapter:     adapter,
		meter:       cfg.meter,
		spawnGate:   spawnGate,
		performGate: performGate,
		admission:   cfg.admission,
	}
}

// Pick returns actor configs matching preferences.
func (b *DefaultBroker) Pick(_ context.Context, prefs troupe.Preferences) ([]troupe.ActorConfig, error) {
	count := prefs.Count
	if count <= 0 {
		count = 1
	}

	configs := make([]troupe.ActorConfig, count)
	for i := range count {
		configs[i] = troupe.ActorConfig{
			Model: prefs.Model,
			Role:  prefs.Role,
		}
	}
	return configs, nil
}

// Spawn creates a running actor.
func (b *DefaultBroker) Spawn(ctx context.Context, cfg troupe.ActorConfig) (troupe.Actor, error) {
	// Driver environment validation (optional interface).
	drv := b.adapter.resolve(0) // check default driver
	if cfg.Provider != "" && b.adapter.drivers != nil {
		if d, ok := b.adapter.drivers[cfg.Provider]; ok {
			drv = d
		}
	}
	if drv != nil {
		if v, ok := drv.(troupe.DriverValidator); ok {
			if err := v.ValidateEnvironment(ctx); err != nil {
				return nil, fmt.Errorf("driver validate: %w", err)
			}
		}
	}

	// Spawn gate: Gate predicates checked before hooks.
	if b.spawnGate != nil {
		allowed, reason, err := b.spawnGate(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("spawn gate: %w", err)
		}
		if !allowed {
			b.emitControl(signal.EventVetoApplied, map[string]string{
				"role": cfg.Role, "reason": reason,
			})
			return nil, fmt.Errorf("spawn gate rejected: %s", reason)
		}
	}

	// Pre-spawn hooks: any SpawnHook can reject.
	for _, h := range b.hooks {
		if sh, ok := h.(SpawnHook); ok {
			if err := sh.PreSpawn(ctx, cfg); err != nil {
				return nil, fmt.Errorf("hook %s pre-spawn: %w", sh.Name(), err)
			}
		}
	}

	role := cfg.Role
	if role == "" {
		role = "actor"
	}

	var id world.EntityID
	var err error

	if b.admission != nil {
		id, err = b.admission.Admit(ctx, cfg)
		if err == nil {
			err = b.warden.StartProcess(ctx, id, role, warden.AgentConfig{
				Model:    cfg.Model,
				Provider: cfg.Provider,
			}, 0)
		}
	} else {
		id, err = b.warden.Fork(ctx, role, warden.AgentConfig{
			Model:    cfg.Model,
			Provider: cfg.Provider,
		}, 0)
	}

	var actor troupe.Actor
	if err == nil {
		actor = agent.NewSolo(id, role, b.world, b.warden, b.transport)
	}

	// Post-spawn hooks: observe result.
	for _, h := range b.hooks {
		if sh, ok := h.(SpawnHook); ok {
			sh.PostSpawn(ctx, cfg, actor, err)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("broker spawn: %w", err)
	}

	// Wrap with perform hooks/gates if any are registered.
	var performHooks []PerformHook
	for _, h := range b.hooks {
		if ph, ok := h.(PerformHook); ok {
			performHooks = append(performHooks, ph)
		}
	}
	if len(performHooks) > 0 || b.performGate != nil {
		actor = newHookedActor(actor, performHooks, b.performGate)
	}

	b.emitControl(signal.EventDispatchRouted, map[string]string{
		"role": cfg.Role, signal.MetaKeyDispatchReason: "spawn",
	})

	return actor, nil
}

func (b *DefaultBroker) emitControl(kind string, meta map[string]string) {
	if b.controlLog == nil {
		return
	}
	b.controlLog.Emit(signal.Event{
		Source: "broker",
		Kind:   kind,
		Data:   signal.Signal{Agent: "broker", Event: kind, Meta: meta},
	})
}

// Discover returns agent cards for live agents, optionally filtered by role.
func (b *DefaultBroker) Discover(role string) []troupe.AgentCard {
	ids := world.Query[visual.Color](b.world)
	cards := make([]troupe.AgentCard, 0, len(ids))
	for _, id := range ids {
		color, _ := world.TryGet[visual.Color](b.world, id)
		if role != "" && color.Role != role {
			continue
		}
		cards = append(cards, &simpleCard{
			name: color.Title(),
			role: color.Role,
		})
	}
	return cards
}

type simpleCard struct {
	name   string
	role   string
	skills []string
}

func (c *simpleCard) Name() string     { return c.name }
func (c *simpleCard) Role() string     { return c.role }
func (c *simpleCard) Skills() []string { return c.skills }

// Meter returns the resource usage meter (nil if none configured).
func (b *DefaultBroker) Meter() troupe.Meter { return b.meter }

// Buses returns the three-bus set (Control, Work, Status).
func (b *DefaultBroker) Buses() signal.BusSet { return b.buses }

// World returns the underlying ECS world (for advanced consumers).
func (b *DefaultBroker) World() *world.World { return b.world }

// SpawnGate returns the composed spawn gate, or nil if none configured.
func (b *DefaultBroker) SpawnGate() troupe.Gate { return b.spawnGate }

// PerformGate returns the composed perform gate, or nil if none configured.
func (b *DefaultBroker) PerformGate() troupe.Gate { return b.performGate }

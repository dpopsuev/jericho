package jericho

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dpopsuev/jericho/agent"
	"github.com/dpopsuev/jericho/identity"
	"github.com/dpopsuev/jericho/warden"
)

// ErrNoLauncher is returned when Spawn is called without a configured launcher.
var ErrNoLauncher = errors.New("broker: no launcher configured")

// DefaultBroker is the standard Broker implementation. It wires World, Warden,
// Transport, Registry, and Signal Bus internally. Consumers see Broker + Actor.
type DefaultBroker struct {
	staff    *agent.Staff
	registry *identity.Registry
}

// BrokerOption configures a DefaultBroker.
type BrokerOption func(*brokerConfig)

type brokerConfig struct {
	launcher warden.AgentSupervisor
}

// WithLauncher sets the agent launcher (e.g., ACP).
func WithLauncher(l warden.AgentSupervisor) BrokerOption {
	return func(c *brokerConfig) { c.launcher = l }
}

// NewBroker creates a Broker. If the endpoint is a remote URL (https://),
// returns a RemoteBroker that proxies over HTTP. Otherwise, returns a
// local DefaultBroker.
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

	var staff *agent.Staff
	if cfg.launcher != nil {
		staff = agent.NewStaff(cfg.launcher)
	}

	return &DefaultBroker{
		staff:    staff,
		registry: identity.NewRegistry(),
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
	if b.staff == nil {
		return nil, ErrNoLauncher
	}

	role := config.Role
	if role == "" {
		role = "actor"
	}

	handle, err := b.staff.Spawn(ctx, role, warden.AgentConfig{
		Model: config.Model,
	})
	if err != nil {
		return nil, fmt.Errorf("broker spawn: %w", err)
	}

	return handle, nil
}

// Staff returns the underlying Staff for advanced consumers.
func (b *DefaultBroker) Staff() *agent.Staff { return b.staff }

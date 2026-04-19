package troupe

import (
	"context"
	"errors"
	"log/slog"

	"github.com/dpopsuev/troupe/world"
)

var (
	ErrNoAdmission = errors.New("troupe: no admission configured")
	ErrNoBroker    = errors.New("troupe: no broker configured")
	ErrNotFound    = errors.New("troupe: not found")
)

// Troupe is the unified Facade over Broker, Admission, and the agent
// ecosystem. CLI, MCP bindings, A2A, and tests all talk to this.
type Troupe struct {
	broker    Broker
	admission Admission
}

// Option configures a Troupe instance.
type TroupeOption func(*Troupe)

// WithBroker sets the Broker implementation.
func WithBroker(b Broker) TroupeOption {
	return func(t *Troupe) { t.broker = b }
}

// WithAdmission sets the Admission implementation.
func WithAdmission(a Admission) TroupeOption {
	return func(t *Troupe) { t.admission = a }
}

// New creates a Troupe Facade.
func New(opts ...TroupeOption) *Troupe {
	t := &Troupe{}
	for _, o := range opts {
		o(t)
	}
	return t
}

// Admit registers an agent into the World via Admission.
func (t *Troupe) Admit(ctx context.Context, config ActorConfig) (world.EntityID, error) {
	if t.admission == nil {
		return 0, ErrNoAdmission
	}
	id, err := t.admission.Admit(ctx, config)
	if err != nil {
		slog.WarnContext(ctx, "troupe admit failed",
			slog.String("role", config.Role),
			slog.String("error", err.Error()),
		)
		return 0, err
	}
	slog.InfoContext(ctx, "troupe admit",
		slog.String("role", config.Role),
		slog.Uint64("entity_id", uint64(id)),
	)
	return id, nil
}

// Kick removes an agent from the World via Admission.
func (t *Troupe) Kick(ctx context.Context, id world.EntityID) error {
	if t.admission == nil {
		return ErrNoAdmission
	}
	return t.admission.Kick(ctx, id)
}

// Ban kicks an agent and prevents re-admission.
func (t *Troupe) Ban(ctx context.Context, id world.EntityID, reason string) error {
	if t.admission == nil {
		return ErrNoAdmission
	}
	return t.admission.Ban(ctx, id, reason)
}

// Unban removes an agent from the deny list.
func (t *Troupe) Unban(ctx context.Context, id world.EntityID) error {
	if t.admission == nil {
		return ErrNoAdmission
	}
	return t.admission.Unban(ctx, id)
}

// Spawn creates a running actor via Broker.
func (t *Troupe) Spawn(ctx context.Context, config ActorConfig) (Actor, error) {
	if t.broker == nil {
		return nil, ErrNoBroker
	}
	return t.broker.Spawn(ctx, config)
}

// Discover returns agent cards for live agents via Broker.
func (t *Troupe) Discover(role string) []AgentCard {
	if t.broker == nil {
		return nil
	}
	return t.broker.Discover(role)
}

// Perform sends a prompt to an actor and returns the response.
func (t *Troupe) Perform(ctx context.Context, actor Actor, prompt string) (string, error) {
	return actor.Perform(ctx, prompt)
}

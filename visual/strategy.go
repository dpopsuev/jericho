package visual

import (
	"time"

	"github.com/dpopsuev/tangle/world"
)

// DefaultStrategy assigns random colors from the palette.
type DefaultStrategy struct {
	w   *world.World
	reg *Registry
}

// NewDefaultStrategy creates a strategy that assigns random color identities.
func NewDefaultStrategy(w *world.World, r *Registry) *DefaultStrategy {
	return &DefaultStrategy{w: w, reg: r}
}

// Resolve creates a new entity with Color + Alive + Ready.
func (s *DefaultStrategy) Resolve(role, collective string) (world.EntityID, error) {
	color, err := s.reg.Assign(role, collective)
	if err != nil {
		return 0, err
	}

	id := s.w.Spawn()
	world.Attach(s.w, id, color)
	world.Attach(s.w, id, world.Alive{State: world.AliveRunning, Since: time.Now()})
	world.Attach(s.w, id, world.Ready{Ready: true, LastSeen: time.Now()})
	return id, nil
}

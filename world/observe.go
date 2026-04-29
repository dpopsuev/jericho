// observe.go — World emits component mutations as events.
//
// Industry pattern: Bevy world.trigger(), Flecs world.observer().event(OnSet).
// The World IS the event source. Observers register on the World.
package world

import (
	"github.com/dpopsuev/tangle/signal"
)

// ComponentMutation is the typed payload for component change events.
type ComponentMutation struct {
	EntityID      uint64 `json:"entity_id"`
	ComponentType string `json:"component_type"`
}

// EmitDiffsTo registers a DiffHook that emits component mutations
// as events to the given EventLog. Accepts any EventLog implementation
// including signal.StatusLog for typed bus routing.
func (w *World) EmitDiffsTo(log signal.EventLog) {
	w.OnDiff(func(id EntityID, ct ComponentType, kind DiffKind, _, _ Component) {
		log.Emit(signal.Event{
			Source: "world",
			Kind:   "component." + string(kind),
			Data: ComponentMutation{
				EntityID:      uint64(id),
				ComponentType: string(ct),
			},
		})
	})
}

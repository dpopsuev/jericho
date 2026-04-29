package world_test

import (
	"sync"
	"testing"

	"github.com/dpopsuev/tangle/testkit"
	"github.com/dpopsuev/tangle/world"
)

func TestEmitDiffsTo_Attach(t *testing.T) {
	w := world.NewWorld()
	log := testkit.NewStubEventLog()
	w.EmitDiffsTo(log)

	id := w.Spawn()
	world.Attach(w, id, world.Alive{State: world.AliveRunning})

	events := log.Since(0)
	if len(events) == 0 {
		t.Fatal("no events after Attach")
	}
	if events[0].Kind != "component.attached" {
		t.Fatalf("Kind = %q, want component.attached", events[0].Kind)
	}
}

func TestEmitDiffsTo_Detach(t *testing.T) {
	w := world.NewWorld()
	log := testkit.NewStubEventLog()
	w.EmitDiffsTo(log)

	id := w.Spawn()
	world.Attach(w, id, world.Alive{State: world.AliveRunning})
	world.Detach[world.Alive](w, id)

	var found bool
	for _, e := range log.Since(0) {
		if e.Kind == "component.detached" {
			found = true
		}
	}
	if !found {
		t.Fatal("no component.detached event")
	}
}

func TestEmitDiffsTo_Update(t *testing.T) {
	w := world.NewWorld()
	log := testkit.NewStubEventLog()
	w.EmitDiffsTo(log)

	id := w.Spawn()
	world.Attach(w, id, world.Alive{State: world.AliveRunning})
	world.Attach(w, id, world.Alive{State: world.AliveTerminated})

	var found bool
	for _, e := range log.Since(0) {
		if e.Kind == "component.updated" {
			found = true
		}
	}
	if !found {
		t.Fatal("no component.updated event")
	}
}

func TestEmitDiffsTo_TypedData(t *testing.T) {
	w := world.NewWorld()
	log := testkit.NewStubEventLog()
	w.EmitDiffsTo(log)

	id := w.Spawn()
	world.Attach(w, id, world.Alive{State: world.AliveRunning})

	events := log.Since(0)
	if len(events) == 0 {
		t.Fatal("no events")
	}

	e := events[0]
	if e.Source != "world" {
		t.Fatalf("Source = %q, want world", e.Source)
	}

	data, ok := e.Data.(world.ComponentMutation)
	if !ok {
		t.Fatalf("Data is %T, want ComponentMutation", e.Data)
	}
	if data.EntityID == 0 {
		t.Fatal("EntityID is 0")
	}
	if data.ComponentType != "alive" {
		t.Fatalf("ComponentType = %q, want alive", data.ComponentType)
	}
}

func TestEmitDiffsTo_MemLog(t *testing.T) {
	w := world.NewWorld()
	log := testkit.NewStubEventLog()
	w.EmitDiffsTo(log)

	id := w.Spawn()
	world.Attach(w, id, world.Alive{State: world.AliveRunning})

	events := log.Since(0)
	if len(events) == 0 {
		t.Fatal("no events via MemLog")
	}
}

func TestEmitDiffsTo_ConcurrentSafe(t *testing.T) {
	w := world.NewWorld()
	log := testkit.NewStubEventLog()
	w.EmitDiffsTo(log)

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			id := w.Spawn()
			world.Attach(w, id, world.Alive{State: world.AliveRunning})
			world.Attach(w, id, world.Ready{Ready: true})
			world.Detach[world.Ready](w, id)
		})
	}
	wg.Wait()

	if log.Len() < 60 {
		t.Fatalf("events = %d, want >= 60", log.Len())
	}
}

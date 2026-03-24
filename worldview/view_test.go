package worldview_test

import (
	"testing"
	"time"

	"github.com/dpopsuev/bugle/palette"
	"github.com/dpopsuev/bugle/world"
	"github.com/dpopsuev/bugle/worldview"
)

// ---------------------------------------------------------------------------
// Snapshot tests
// ---------------------------------------------------------------------------

func TestSnapshot_MatchesComponentTypes(t *testing.T) {
	w := world.NewWorld()
	a := w.Spawn()
	b := w.Spawn()
	c := w.Spawn()

	world.Attach(w, a, world.Health{State: world.Active})
	world.Attach(w, a, palette.ColorIdentity{Colour: "Denim", Collective: "Refactor"})

	world.Attach(w, b, world.Health{State: world.Idle})
	world.Attach(w, b, palette.ColorIdentity{Colour: "Scarlet", Collective: "Triage"})

	// c has only Health — should NOT match a 2-type query.
	world.Attach(w, c, world.Health{State: world.Done})

	v := worldview.NewView(w)
	snaps := v.Snapshot(world.HealthType, palette.ColorIdentityType)

	if len(snaps) != 2 {
		t.Fatalf("Snapshot returned %d entities, want 2", len(snaps))
	}

	ids := make(map[world.EntityID]bool)
	for _, s := range snaps {
		ids[s.ID] = true
		if len(s.Components) != 2 {
			t.Errorf("entity %d has %d components, want 2", s.ID, len(s.Components))
		}
	}
	if !ids[a] || !ids[b] {
		t.Errorf("expected entities %d and %d, got %v", a, b, ids)
	}
	if ids[c] {
		t.Error("entity c should not be in the snapshot")
	}
}

func TestSnapshot_NoMatches(t *testing.T) {
	w := world.NewWorld()
	w.Spawn() // entity with no components

	v := worldview.NewView(w)
	snaps := v.Snapshot(world.BudgetType)

	if len(snaps) != 0 {
		t.Errorf("Snapshot returned %d entities, want 0", len(snaps))
	}
}

func TestSnapshot_ReflectsLatestState(t *testing.T) {
	w := world.NewWorld()
	id := w.Spawn()
	world.Attach(w, id, world.Health{State: world.Active})

	v := worldview.NewView(w)

	// First snapshot.
	snaps := v.Snapshot(world.HealthType)
	if len(snaps) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snaps))
	}
	h := snaps[0].Components[world.HealthType].(world.Health)
	if h.State != world.Active {
		t.Errorf("state = %s, want active", h.State)
	}

	// Update and re-snapshot.
	world.Attach(w, id, world.Health{State: world.Errored, Error: "timeout"})

	snaps = v.Snapshot(world.HealthType)
	h = snaps[0].Components[world.HealthType].(world.Health)
	if h.State != world.Errored {
		t.Errorf("state = %s, want errored", h.State)
	}
}

// ---------------------------------------------------------------------------
// Subscribe tests
// ---------------------------------------------------------------------------

func TestSubscribe_AttachEmitsDiff(t *testing.T) {
	w := world.NewWorld()
	v := worldview.NewView(w)
	ch := v.Subscribe()

	id := w.Spawn()
	world.Attach(w, id, world.Health{State: world.Active})

	select {
	case d := <-ch:
		if d.Kind != world.DiffAttached {
			t.Errorf("kind = %s, want attached", d.Kind)
		}
		if d.Entity != id {
			t.Errorf("entity = %d, want %d", d.Entity, id)
		}
		if d.Component != world.HealthType {
			t.Errorf("component = %s, want %s", d.Component, world.HealthType)
		}
		if d.Old != nil {
			t.Error("Old should be nil for attached")
		}
		if d.New == nil {
			t.Error("New should not be nil for attached")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for diff")
	}
}

func TestSubscribe_UpdateEmitsDiff(t *testing.T) {
	w := world.NewWorld()
	v := worldview.NewView(w)

	id := w.Spawn()
	world.Attach(w, id, world.Health{State: world.Active})

	ch := v.Subscribe()

	// Second attach triggers DiffUpdated.
	world.Attach(w, id, world.Health{State: world.Errored, Error: "timeout"})

	select {
	case d := <-ch:
		if d.Kind != world.DiffUpdated {
			t.Errorf("kind = %s, want updated", d.Kind)
		}
		if d.Old == nil {
			t.Fatal("Old should not be nil for updated")
		}
		oldH := d.Old.(world.Health)
		if oldH.State != world.Active {
			t.Errorf("old state = %s, want active", oldH.State)
		}
		newH := d.New.(world.Health)
		if newH.State != world.Errored {
			t.Errorf("new state = %s, want errored", newH.State)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for diff")
	}
}

func TestSubscribe_DetachEmitsDiff(t *testing.T) {
	w := world.NewWorld()
	v := worldview.NewView(w)

	id := w.Spawn()
	world.Attach(w, id, world.Health{State: world.Active})

	ch := v.Subscribe()
	world.Detach[world.Health](w, id)

	select {
	case d := <-ch:
		if d.Kind != world.DiffDetached {
			t.Errorf("kind = %s, want detached", d.Kind)
		}
		if d.Old == nil {
			t.Fatal("Old should not be nil for detached")
		}
		if d.New != nil {
			t.Error("New should be nil for detached")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for diff")
	}
}

func TestSubscribe_FiltersByType(t *testing.T) {
	w := world.NewWorld()
	v := worldview.NewView(w)

	// Subscribe to Health only.
	ch := v.Subscribe(world.HealthType)

	id := w.Spawn()
	// Attach a ColorIdentity (should NOT trigger diff on this channel).
	world.Attach(w, id, palette.ColorIdentity{Colour: "Denim"})

	select {
	case d := <-ch:
		t.Errorf("should not have received diff, got %+v", d)
	case <-time.After(50 * time.Millisecond):
		// Expected: no diff received.
	}

	// Now attach Health — should trigger.
	world.Attach(w, id, world.Health{State: world.Active})

	select {
	case d := <-ch:
		if d.Component != world.HealthType {
			t.Errorf("component = %s, want %s", d.Component, world.HealthType)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Health diff")
	}
}

func TestSubscribe_Unsubscribe(t *testing.T) {
	w := world.NewWorld()
	v := worldview.NewView(w)

	ch := v.Subscribe()
	v.Unsubscribe(ch)

	// Channel should be closed.
	_, ok := <-ch
	if ok {
		t.Error("channel should be closed after Unsubscribe")
	}

	// Attach should not panic (no subscriber).
	id := w.Spawn()
	world.Attach(w, id, world.Health{State: world.Active})
}

func TestSubscribe_MultipleSubs(t *testing.T) {
	w := world.NewWorld()
	v := worldview.NewView(w)

	ch1 := v.Subscribe()
	ch2 := v.Subscribe()

	id := w.Spawn()
	world.Attach(w, id, world.Health{State: world.Active})

	for i, ch := range []<-chan worldview.Diff{ch1, ch2} {
		select {
		case d := <-ch:
			if d.Kind != world.DiffAttached {
				t.Errorf("sub %d: kind = %s, want attached", i, d.Kind)
			}
		case <-time.After(time.Second):
			t.Fatalf("sub %d: timed out waiting for diff", i)
		}
	}
}

// ---------------------------------------------------------------------------
// Hierarchy tests
// ---------------------------------------------------------------------------

func TestHierarchy_BuildsTree(t *testing.T) {
	w := world.NewWorld()
	parent := w.Spawn()
	child := w.Spawn()
	grandchild := w.Spawn()

	world.Attach(w, parent, world.Hierarchy{Parent: 0})
	world.Attach(w, child, world.Hierarchy{Parent: parent})
	world.Attach(w, grandchild, world.Hierarchy{Parent: child})

	v := worldview.NewView(w)
	tree := v.Hierarchy()

	if len(tree) != 1 {
		t.Fatalf("expected 1 root, got %d", len(tree))
	}
	root := tree[0]
	if root.ID != parent {
		t.Errorf("root ID = %d, want %d", root.ID, parent)
	}
	if len(root.Children) != 1 {
		t.Fatalf("root children = %d, want 1", len(root.Children))
	}
	childNode := root.Children[0]
	if childNode.ID != child {
		t.Errorf("child ID = %d, want %d", childNode.ID, child)
	}
	if len(childNode.Children) != 1 {
		t.Fatalf("child children = %d, want 1", len(childNode.Children))
	}
	if childNode.Children[0].ID != grandchild {
		t.Errorf("grandchild ID = %d, want %d", childNode.Children[0].ID, grandchild)
	}
}

func TestHierarchy_RootsHaveNoParent(t *testing.T) {
	w := world.NewWorld()
	a := w.Spawn()
	b := w.Spawn()
	c := w.Spawn()

	// a and b are roots (Parent=0), c is child of a.
	world.Attach(w, a, world.Hierarchy{Parent: 0})
	world.Attach(w, b, world.Hierarchy{Parent: 0})
	world.Attach(w, c, world.Hierarchy{Parent: a})

	v := worldview.NewView(w)
	tree := v.Hierarchy()

	if len(tree) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(tree))
	}

	rootIDs := make(map[world.EntityID]bool)
	for _, n := range tree {
		rootIDs[n.ID] = true
	}
	if !rootIDs[a] || !rootIDs[b] {
		t.Errorf("expected roots %d and %d, got %v", a, b, rootIDs)
	}
}

// ---------------------------------------------------------------------------
// Stats tests
// ---------------------------------------------------------------------------

func TestStats_CountsByState(t *testing.T) {
	w := world.NewWorld()

	// 3 active, 2 idle, 1 errored.
	for range 3 {
		id := w.Spawn()
		world.Attach(w, id, world.Health{State: world.Active})
	}
	for range 2 {
		id := w.Spawn()
		world.Attach(w, id, world.Health{State: world.Idle})
	}
	{
		id := w.Spawn()
		world.Attach(w, id, world.Health{State: world.Errored})
	}

	v := worldview.NewView(w)
	s := v.Stats()

	if s.TotalEntities != 6 {
		t.Errorf("TotalEntities = %d, want 6", s.TotalEntities)
	}
	if s.ByState[world.Active] != 3 {
		t.Errorf("Active = %d, want 3", s.ByState[world.Active])
	}
	if s.ByState[world.Idle] != 2 {
		t.Errorf("Idle = %d, want 2", s.ByState[world.Idle])
	}
	if s.ByState[world.Errored] != 1 {
		t.Errorf("Errored = %d, want 1", s.ByState[world.Errored])
	}
}

func TestStats_CountsCollectives(t *testing.T) {
	w := world.NewWorld()

	// 3 in "Refactor", 2 in "Triage".
	for range 3 {
		id := w.Spawn()
		world.Attach(w, id, palette.ColorIdentity{Colour: "A", Collective: "Refactor"})
	}
	for range 2 {
		id := w.Spawn()
		world.Attach(w, id, palette.ColorIdentity{Colour: "B", Collective: "Triage"})
	}

	v := worldview.NewView(w)
	s := v.Stats()

	if s.Collectives != 2 {
		t.Errorf("Collectives = %d, want 2", s.Collectives)
	}
}

// ---------------------------------------------------------------------------
// Acceptance test
// ---------------------------------------------------------------------------

func TestAcceptance_MinimapPattern(t *testing.T) {
	// Full pattern: spawn agents with identity+health, create View,
	// snapshot, verify readable output.
	w := world.NewWorld()

	agents := []struct {
		colour     string
		shade      string
		role       string
		collective string
		state      world.AgentState
	}{
		{"Denim", "Indigo", "Writer", "Refactor", world.Active},
		{"Scarlet", "Crimson", "Reviewer", "Refactor", world.Active},
		{"Cerulean", "Azure", "Coder", "Triage", world.Idle},
	}

	for _, a := range agents {
		id := w.Spawn()
		world.Attach(w, id, palette.ColorIdentity{
			Shade:      a.shade,
			Colour:     a.colour,
			Role:       a.role,
			Collective: a.collective,
		})
		world.Attach(w, id, world.Health{State: a.state, LastSeen: time.Now()})
	}

	v := worldview.NewView(w)

	// Snapshot all agents with both components.
	snaps := v.Snapshot(palette.ColorIdentityType, world.HealthType)
	if len(snaps) != 3 {
		t.Fatalf("expected 3 snapshots, got %d", len(snaps))
	}

	// Verify each snapshot has readable data.
	for _, s := range snaps {
		ci := s.Components[palette.ColorIdentityType].(palette.ColorIdentity)
		h := s.Components[world.HealthType].(world.Health)
		if ci.Colour == "" {
			t.Errorf("entity %d: empty Colour", s.ID)
		}
		if h.State == "" {
			t.Errorf("entity %d: empty State", s.ID)
		}
		t.Logf("entity %d: %s — %s", s.ID, ci.Title(), h.State)
	}

	// Stats should reflect the world.
	stats := v.Stats()
	if stats.TotalEntities != 3 {
		t.Errorf("TotalEntities = %d, want 3", stats.TotalEntities)
	}
	if stats.ByState[world.Active] != 2 {
		t.Errorf("Active = %d, want 2", stats.ByState[world.Active])
	}
	if stats.Collectives != 2 {
		t.Errorf("Collectives = %d, want 2", stats.Collectives)
	}
}

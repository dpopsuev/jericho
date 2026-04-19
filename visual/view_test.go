package visual_test

import (
	"testing"
	"time"

	"github.com/dpopsuev/troupe/visual"
	"github.com/dpopsuev/troupe/world"
)

// ---------------------------------------------------------------------------
// Snapshot tests
// ---------------------------------------------------------------------------

func TestSnapshot_MatchesComponentTypes(t *testing.T) {
	w := world.NewWorld()
	a := w.Spawn()
	b := w.Spawn()
	c := w.Spawn()

	world.Attach(w, a, world.Alive{State: world.AliveRunning, Since: time.Now()})
	world.Attach(w, a, visual.Color{Name: "Denim", Collective: "Refactor"})

	world.Attach(w, b, world.Alive{State: world.AliveRunning, Since: time.Now()})
	world.Attach(w, b, world.Ready{Ready: false, LastSeen: time.Now(), Reason: world.ReasonIdle})
	world.Attach(w, b, visual.Color{Name: "Scarlet", Collective: "Triage"})

	// c has only Health — should NOT match a 2-type query.
	world.Attach(w, c, world.Alive{State: world.AliveTerminated, ExitedAt: time.Now()})

	v := visual.NewView(w)
	snaps := v.Snapshot(world.AliveType, visual.ColorType)

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

	v := visual.NewView(w)
	snaps := v.Snapshot(world.BudgetType)

	if len(snaps) != 0 {
		t.Errorf("Snapshot returned %d entities, want 0", len(snaps))
	}
}

func TestSnapshot_ReflectsLatestState(t *testing.T) {
	w := world.NewWorld()
	id := w.Spawn()
	world.Attach(w, id, world.Alive{State: world.AliveRunning, Since: time.Now()})

	v := visual.NewView(w)

	// First snapshot.
	snaps := v.Snapshot(world.AliveType)
	if len(snaps) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snaps))
	}
	alive := snaps[0].Components[world.AliveType].(world.Alive)
	if alive.State != world.AliveRunning {
		t.Errorf("state = %s, want running", alive.State)
	}

	// Update ready and re-snapshot.
	world.Attach(w, id, world.Ready{Ready: false, LastSeen: time.Now(), Reason: world.ReasonErrored, Error: "timeout"})

	snaps = v.Snapshot(world.ReadyType)
	r := snaps[0].Components[world.ReadyType].(world.Ready)
	if r.Reason != world.ReasonErrored {
		t.Errorf("reason = %s, want errored", r.Reason)
	}
}

// ---------------------------------------------------------------------------
// Subscribe tests
// ---------------------------------------------------------------------------

func TestSubscribe_AttachEmitsDiff(t *testing.T) {
	w := world.NewWorld()
	v := visual.NewView(w)
	ch := v.Subscribe()

	id := w.Spawn()
	world.Attach(w, id, world.Alive{State: world.AliveRunning, Since: time.Now()})

	select {
	case d := <-ch:
		if d.Kind != world.DiffAttached {
			t.Errorf("kind = %s, want attached", d.Kind)
		}
		if d.Entity != id {
			t.Errorf("entity = %d, want %d", d.Entity, id)
		}
		if d.Component != world.AliveType {
			t.Errorf("component = %s, want %s", d.Component, world.AliveType)
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
	v := visual.NewView(w)

	id := w.Spawn()
	world.Attach(w, id, world.Ready{Ready: true, LastSeen: time.Now()})

	ch := v.Subscribe()

	// Second attach of same component triggers DiffUpdated.
	world.Attach(w, id, world.Ready{Ready: false, LastSeen: time.Now(), Reason: world.ReasonErrored, Error: "timeout"})

	select {
	case d := <-ch:
		if d.Kind != world.DiffUpdated {
			t.Errorf("kind = %s, want updated", d.Kind)
		}
		if d.Old == nil {
			t.Fatal("Old should not be nil for updated")
		}
		oldR := d.Old.(world.Ready)
		if !oldR.Ready {
			t.Errorf("old ready = false, want true")
		}
		newR := d.New.(world.Ready)
		if newR.Reason != world.ReasonErrored {
			t.Errorf("new reason = %s, want errored", newR.Reason)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for diff")
	}
}

func TestSubscribe_DetachEmitsDiff(t *testing.T) {
	w := world.NewWorld()
	v := visual.NewView(w)

	id := w.Spawn()
	world.Attach(w, id, world.Alive{State: world.AliveRunning, Since: time.Now()})

	ch := v.Subscribe()
	world.Detach[world.Alive](w, id)

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
	v := visual.NewView(w)

	// Subscribe to Health only.
	ch := v.Subscribe(world.AliveType)

	id := w.Spawn()
	// Attach a ColorIdentity (should NOT trigger diff on this channel).
	world.Attach(w, id, visual.Color{Name: "Denim"})

	select {
	case d := <-ch:
		t.Errorf("should not have received diff, got %+v", d)
	case <-time.After(50 * time.Millisecond):
		// Expected: no diff received.
	}

	// Now attach Health — should trigger.
	world.Attach(w, id, world.Alive{State: world.AliveRunning, Since: time.Now()})

	select {
	case d := <-ch:
		if d.Component != world.AliveType {
			t.Errorf("component = %s, want %s", d.Component, world.AliveType)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Health diff")
	}
}

func TestSubscribe_Unsubscribe(t *testing.T) {
	w := world.NewWorld()
	v := visual.NewView(w)

	ch := v.Subscribe()
	v.Unsubscribe(ch)

	// Channel should be closed.
	_, ok := <-ch
	if ok {
		t.Error("channel should be closed after Unsubscribe")
	}

	// Attach should not panic (no subscriber).
	id := w.Spawn()
	world.Attach(w, id, world.Alive{State: world.AliveRunning, Since: time.Now()})
}

func TestSubscribe_MultipleSubs(t *testing.T) {
	w := world.NewWorld()
	v := visual.NewView(w)

	ch1 := v.Subscribe()
	ch2 := v.Subscribe()

	id := w.Spawn()
	world.Attach(w, id, world.Alive{State: world.AliveRunning, Since: time.Now()})

	for i, ch := range []<-chan visual.Diff{ch1, ch2} {
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

	_ = w.Link(parent, world.Supervises, child)
	_ = w.Link(child, world.Supervises, grandchild)

	v := visual.NewView(w)
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

	// a and b are roots (no inbound supervises), c is child of a.
	_ = w.Link(a, world.Supervises, c)

	v := visual.NewView(w)
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

func TestStats_CountsByAliveAndReady(t *testing.T) {
	w := world.NewWorld()

	// 3 running + ready, 2 running + not-ready, 1 terminated.
	for range 3 {
		id := w.Spawn()
		world.Attach(w, id, world.Alive{State: world.AliveRunning, Since: time.Now()})
		world.Attach(w, id, world.Ready{Ready: true, LastSeen: time.Now()})
	}
	for range 2 {
		id := w.Spawn()
		world.Attach(w, id, world.Alive{State: world.AliveRunning, Since: time.Now()})
		world.Attach(w, id, world.Ready{Ready: false, LastSeen: time.Now(), Reason: world.ReasonIdle})
	}
	{
		id := w.Spawn()
		world.Attach(w, id, world.Alive{State: world.AliveTerminated, ExitedAt: time.Now()})
		world.Attach(w, id, world.Ready{Ready: false, LastSeen: time.Now(), Reason: world.ReasonTerminated})
	}

	v := visual.NewView(w)
	s := v.Stats()

	if s.TotalEntities != 6 {
		t.Errorf("TotalEntities = %d, want 6", s.TotalEntities)
	}
	if s.ByAlive[world.AliveRunning] != 5 {
		t.Errorf("Running = %d, want 5", s.ByAlive[world.AliveRunning])
	}
	if s.ByAlive[world.AliveTerminated] != 1 {
		t.Errorf("Terminated = %d, want 1", s.ByAlive[world.AliveTerminated])
	}
	if s.ReadyCount != 3 {
		t.Errorf("Ready = %d, want 3", s.ReadyCount)
	}
	if s.NotReadyCount != 3 {
		t.Errorf("NotReady = %d, want 3", s.NotReadyCount)
	}
}

func TestStats_CountsCollectives(t *testing.T) {
	w := world.NewWorld()

	// 3 in "Refactor", 2 in "Triage".
	for range 3 {
		id := w.Spawn()
		world.Attach(w, id, visual.Color{Name: "A", Collective: "Refactor"})
	}
	for range 2 {
		id := w.Spawn()
		world.Attach(w, id, visual.Color{Name: "B", Collective: "Triage"})
	}

	v := visual.NewView(w)
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
		color      string
		shade      string
		role       string
		collective string
		ready      bool
	}{
		{"Denim", "Indigo", "Writer", "Refactor", true},
		{"Scarlet", "Crimson", "Reviewer", "Refactor", true},
		{"Cerulean", "Azure", "Coder", "Triage", false},
	}

	for _, a := range agents {
		id := w.Spawn()
		world.Attach(w, id, visual.Color{
			Shade:      a.shade,
			Name:       a.color,
			Role:       a.role,
			Collective: a.collective,
		})
		world.Attach(w, id, world.Alive{State: world.AliveRunning, Since: time.Now()})
		world.Attach(w, id, world.Ready{Ready: a.ready, LastSeen: time.Now()})
	}

	v := visual.NewView(w)

	// Snapshot all agents with both components.
	snaps := v.Snapshot(visual.ColorType, world.AliveType)
	if len(snaps) != 3 {
		t.Fatalf("expected 3 snapshots, got %d", len(snaps))
	}

	// Verify each snapshot has readable data.
	for _, s := range snaps {
		ci := s.Components[visual.ColorType].(visual.Color)
		alive := s.Components[world.AliveType].(world.Alive)
		if ci.Name == "" {
			t.Errorf("entity %d: empty Color", s.ID)
		}
		if alive.State == "" {
			t.Errorf("entity %d: empty State", s.ID)
		}
		t.Logf("entity %d: %s — %s", s.ID, ci.Title(), alive.State)
	}

	// Stats should reflect the world.
	stats := v.Stats()
	if stats.TotalEntities != 3 {
		t.Errorf("TotalEntities = %d, want 3", stats.TotalEntities)
	}
	if stats.ByAlive[world.AliveRunning] != 3 {
		t.Errorf("Running = %d, want 3", stats.ByAlive[world.AliveRunning])
	}
	if stats.Collectives != 2 {
		t.Errorf("Collectives = %d, want 2", stats.Collectives)
	}
}

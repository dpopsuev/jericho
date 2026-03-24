package worldview

import (
	"log"
	"sort"
	"sync"

	"github.com/dpopsuev/bugle"
)

// EntitySnapshot is a point-in-time capture of an entity's components.
type EntitySnapshot struct {
	ID         bugle.EntityID
	Components map[bugle.ComponentType]bugle.Component
}

// Diff describes a single component change on an entity.
type Diff struct {
	Entity    bugle.EntityID
	Component bugle.ComponentType
	Kind      bugle.DiffKind
	Old       bugle.Component
	New       bugle.Component
}

// TreeNode represents a node in the entity hierarchy.
type TreeNode struct {
	ID       bugle.EntityID
	Children []TreeNode
}

// Stats provides aggregate counters for the world.
type Stats struct {
	TotalEntities int
	ByState       map[bugle.AgentState]int
	Collectives   int
}

// View provides read-only projections of an ECS World. It supports
// snapshots, live diff subscriptions, hierarchy trees, and aggregate stats.
type View struct {
	world *bugle.World
	mu    sync.RWMutex
	subs  map[<-chan Diff]*subscription
}

type subscription struct {
	ch    chan Diff
	types map[bugle.ComponentType]bool // nil = all types
}

// NewView creates a View and registers a DiffHook on the World that
// forwards component diffs to active subscribers.
func NewView(w *bugle.World) *View {
	v := &View{
		world: w,
		subs:  make(map[<-chan Diff]*subscription),
	}
	w.OnDiff(func(id bugle.EntityID, ct bugle.ComponentType, kind bugle.DiffKind, old, new bugle.Component) {
		d := Diff{
			Entity:    id,
			Component: ct,
			Kind:      kind,
			Old:       old,
			New:       new,
		}
		v.mu.RLock()
		defer v.mu.RUnlock()
		for _, sub := range v.subs {
			if sub.types != nil && !sub.types[ct] {
				continue
			}
			select {
			case sub.ch <- d:
			default:
				log.Printf("worldview: subscriber channel full, dropping diff for entity %d component %s", id, ct)
			}
		}
	})
	return v
}

// Snapshot returns point-in-time snapshots of all entities that possess
// every one of the requested component types. If no types are specified,
// all entities are returned with empty component maps.
func (v *View) Snapshot(types ...bugle.ComponentType) []EntitySnapshot {
	if len(types) == 0 {
		ids := v.world.All()
		result := make([]EntitySnapshot, len(ids))
		for i, id := range ids {
			result[i] = EntitySnapshot{
				ID:         id,
				Components: make(map[bugle.ComponentType]bugle.Component),
			}
		}
		return result
	}

	// Start with entities that have the first type.
	candidates := v.world.QueryType(types[0])
	if len(candidates) == 0 {
		return nil
	}

	// Intersect with remaining types.
	candidateSet := make(map[bugle.EntityID]bool, len(candidates))
	for _, id := range candidates {
		candidateSet[id] = true
	}
	for _, ct := range types[1:] {
		ids := v.world.QueryType(ct)
		next := make(map[bugle.EntityID]bool, len(ids))
		for _, id := range ids {
			if candidateSet[id] {
				next[id] = true
			}
		}
		candidateSet = next
		if len(candidateSet) == 0 {
			return nil
		}
	}

	// Build snapshots.
	result := make([]EntitySnapshot, 0, len(candidateSet))
	for id := range candidateSet {
		snap := EntitySnapshot{
			ID:         id,
			Components: make(map[bugle.ComponentType]bugle.Component, len(types)),
		}
		for _, ct := range types {
			if c, ok := v.world.GetType(id, ct); ok {
				snap.Components[ct] = c
			}
		}
		result = append(result, snap)
	}

	// Sort by ID for deterministic output.
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}

// Subscribe returns a channel that receives Diff values for matching
// component types. If no types are specified, all diffs are forwarded.
// The channel is buffered (cap 64); diffs are dropped if full.
func (v *View) Subscribe(types ...bugle.ComponentType) <-chan Diff {
	ch := make(chan Diff, 64)
	sub := &subscription{ch: ch}
	if len(types) > 0 {
		sub.types = make(map[bugle.ComponentType]bool, len(types))
		for _, ct := range types {
			sub.types[ct] = true
		}
	}
	v.mu.Lock()
	v.subs[ch] = sub
	v.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscription and closes its channel.
func (v *View) Unsubscribe(ch <-chan Diff) {
	v.mu.Lock()
	defer v.mu.Unlock()
	sub, ok := v.subs[ch]
	if !ok {
		return
	}
	delete(v.subs, ch)
	close(sub.ch)
}

// Hierarchy builds a tree of all entities that have a Hierarchy component.
// Roots are entities whose Parent is 0 or whose Parent is not alive.
func (v *View) Hierarchy() []TreeNode {
	ids := v.world.QueryType(bugle.HierarchyType)
	if len(ids) == 0 {
		return nil
	}

	// Collect parent info.
	type entry struct {
		id     bugle.EntityID
		parent bugle.EntityID
	}
	entries := make([]entry, 0, len(ids))
	for _, id := range ids {
		c, ok := v.world.GetType(id, bugle.HierarchyType)
		if !ok {
			continue
		}
		h := c.(bugle.Hierarchy) //nolint:errcheck // type guaranteed by QueryType
		entries = append(entries, entry{id: id, parent: h.Parent})
	}

	// Build parent→children map.
	children := make(map[bugle.EntityID][]bugle.EntityID)
	var roots []bugle.EntityID
	for _, e := range entries {
		if e.parent == 0 || !v.world.Alive(e.parent) {
			roots = append(roots, e.id)
		} else {
			children[e.parent] = append(children[e.parent], e.id)
		}
	}

	// Sort roots for deterministic output.
	sort.Slice(roots, func(i, j int) bool { return roots[i] < roots[j] })
	for k := range children {
		sort.Slice(children[k], func(i, j int) bool { return children[k][i] < children[k][j] })
	}

	var buildTree func(id bugle.EntityID) TreeNode
	buildTree = func(id bugle.EntityID) TreeNode {
		node := TreeNode{ID: id}
		for _, childID := range children[id] {
			node.Children = append(node.Children, buildTree(childID))
		}
		return node
	}

	tree := make([]TreeNode, 0, len(roots))
	for _, r := range roots {
		tree = append(tree, buildTree(r))
	}
	return tree
}

// Stats returns aggregate counters for the world.
func (v *View) Stats() Stats {
	s := Stats{
		TotalEntities: v.world.Count(),
		ByState:       make(map[bugle.AgentState]int),
	}

	healthIDs := v.world.QueryType(bugle.HealthType)
	for _, id := range healthIDs {
		c, ok := v.world.GetType(id, bugle.HealthType)
		if !ok {
			continue
		}
		h := c.(bugle.Health) //nolint:errcheck // type guaranteed by QueryType
		s.ByState[h.State]++
	}

	colorIDs := v.world.QueryType(bugle.ColorIdentityType)
	collectives := make(map[string]bool)
	for _, id := range colorIDs {
		c, ok := v.world.GetType(id, bugle.ColorIdentityType)
		if !ok {
			continue
		}
		ci := c.(bugle.ColorIdentity) //nolint:errcheck // type guaranteed by QueryType
		if ci.Collective != "" {
			collectives[ci.Collective] = true
		}
	}
	s.Collectives = len(collectives)
	return s
}

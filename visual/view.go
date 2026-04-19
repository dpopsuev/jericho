package visual

import (
	"log"
	"sort"
	"sync"

	
	"github.com/dpopsuev/troupe/world"
)

// EntitySnapshot is a point-in-time capture of an entity's components.
type EntitySnapshot struct {
	ID         world.EntityID
	Components map[world.ComponentType]world.Component
}

// Diff describes a single component change on an entity.
type Diff struct {
	Entity    world.EntityID
	Component world.ComponentType
	Kind      world.DiffKind
	Old       world.Component
	New       world.Component
}

// TreeNode represents a node in the entity hierarchy.
type TreeNode struct {
	ID       world.EntityID
	Children []TreeNode
}

// Stats provides aggregate counters for the world.
type Stats struct {
	TotalEntities int
	ByAlive       map[world.AliveState]int
	ReadyCount    int
	NotReadyCount int
	Collectives   int
}

// View provides read-only projections of an ECS World. It supports
// snapshots, live diff subscriptions, hierarchy trees, and aggregate stats.
type View struct {
	world *world.World
	mu    sync.RWMutex
	subs  map[<-chan Diff]*subscription
}

type subscription struct {
	ch    chan Diff
	types map[world.ComponentType]bool // nil = all types
}

// NewView creates a View and registers a DiffHook on the World that
// forwards component diffs to active subscribers.
func NewView(w *world.World) *View {
	v := &View{
		world: w,
		subs:  make(map[<-chan Diff]*subscription),
	}
	w.OnDiff(func(id world.EntityID, ct world.ComponentType, kind world.DiffKind, old, new world.Component) {
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
func (v *View) Snapshot(types ...world.ComponentType) []EntitySnapshot {
	if len(types) == 0 {
		ids := v.world.All()
		result := make([]EntitySnapshot, len(ids))
		for i, id := range ids {
			result[i] = EntitySnapshot{
				ID:         id,
				Components: make(map[world.ComponentType]world.Component),
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
	candidateSet := make(map[world.EntityID]bool, len(candidates))
	for _, id := range candidates {
		candidateSet[id] = true
	}
	for _, ct := range types[1:] {
		ids := v.world.QueryType(ct)
		next := make(map[world.EntityID]bool, len(ids))
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
			Components: make(map[world.ComponentType]world.Component, len(types)),
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
func (v *View) Subscribe(types ...world.ComponentType) <-chan Diff {
	ch := make(chan Diff, 64)
	sub := &subscription{ch: ch}
	if len(types) > 0 {
		sub.types = make(map[world.ComponentType]bool, len(types))
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

// Hierarchy builds a tree from supervises edges (GOL-14).
// Roots are entities with no inbound supervises edges.
func (v *View) Hierarchy() []TreeNode {
	allIDs := v.world.All()
	if len(allIDs) == 0 {
		return nil
	}

	// Find roots: entities with no inbound supervises edge.
	var roots []world.EntityID
	for _, id := range allIDs {
		parents := v.world.Neighbors(id, world.Supervises, world.Inbound)
		if len(parents) == 0 {
			roots = append(roots, id)
		}
	}

	sort.Slice(roots, func(i, j int) bool { return roots[i] < roots[j] })

	var buildTree func(id world.EntityID) TreeNode
	buildTree = func(id world.EntityID) TreeNode {
		node := TreeNode{ID: id}
		children := v.world.Neighbors(id, world.Supervises, world.Outbound)
		sort.Slice(children, func(i, j int) bool { return children[i] < children[j] })
		for _, childID := range children {
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
		ByAlive:       make(map[world.AliveState]int),
	}

	aliveIDs := v.world.QueryType(world.AliveType)
	for _, id := range aliveIDs {
		c, ok := v.world.GetType(id, world.AliveType)
		if !ok {
			continue
		}
		a := c.(world.Alive) //nolint:errcheck // type guaranteed by QueryType
		s.ByAlive[a.State]++
	}

	readyIDs := v.world.QueryType(world.ReadyType)
	for _, id := range readyIDs {
		c, ok := v.world.GetType(id, world.ReadyType)
		if !ok {
			continue
		}
		r := c.(world.Ready) //nolint:errcheck // type guaranteed by QueryType
		if r.Ready {
			s.ReadyCount++
		} else {
			s.NotReadyCount++
		}
	}

	colorIDs := v.world.QueryType(ColorType)
	collectives := make(map[string]bool)
	for _, id := range colorIDs {
		c, ok := v.world.GetType(id, ColorType)
		if !ok {
			continue
		}
		ci := c.(Color) //nolint:errcheck // type guaranteed by QueryType
		if ci.Collective != "" {
			collectives[ci.Collective] = true
		}
	}
	s.Collectives = len(collectives)
	return s
}

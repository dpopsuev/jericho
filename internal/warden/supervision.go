// supervision.go — Linux-inspired process supervision for AgentWarden.
// Wait/WaitAny (zombie reaping), KillAs (ownership), KillChildren (process group),
// Children/Tree (hierarchy), SetSubreaper (orphan adoption), SetAutoReap.
package warden

import (
	"context"
	"fmt"
	"time"

	"github.com/dpopsuev/tangle/world"
)

// Wait blocks until the specified child finishes, returns its exit status,
// and removes the zombie entry. Like waitpid(pid).
func (p *AgentWarden) Wait(ctx context.Context, childID world.EntityID) (*ExitStatus, error) {
	// Check if already a zombie.
	p.mu.RLock()
	if z, ok := p.zombies[childID]; ok {
		p.mu.RUnlock()
		return p.reap(childID, z), nil
	}

	// Check if the agent exists and get its wait channel.
	ch, ok := p.waitCh[childID]
	p.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%w: %d", ErrNotFound, childID)
	}

	// Block until child finishes or context is canceled.
	select {
	case <-ch:
		p.mu.RLock()
		z, ok := p.zombies[childID]
		p.mu.RUnlock()
		if !ok {
			// Auto-reaped — no status available.
			return nil, nil
		}
		return p.reap(childID, z), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// WaitAny returns the exit status of any finished child of parentID
// without blocking. Returns nil if no zombies. Like waitpid(-1, WNOHANG).
func (p *AgentWarden) WaitAny(parentID world.EntityID) *ExitStatus {
	p.mu.RLock()
	var found world.EntityID
	var entry *agentEntry
	for id, z := range p.zombies {
		if z.ParentID == parentID {
			found = id
			entry = z
			break
		}
	}
	p.mu.RUnlock()

	if entry == nil {
		return nil
	}
	return p.reap(found, entry)
}

// reap removes a zombie entry and returns its exit status.
func (p *AgentWarden) reap(id world.EntityID, entry *agentEntry) *ExitStatus {
	p.mu.Lock()
	delete(p.zombies, id)
	delete(p.waitCh, id)
	p.mu.Unlock()

	// Despawn the entity now that it's fully reaped.
	p.world.Despawn(id)

	return &ExitStatus{
		AgentID:  entry.ID,
		ParentID: entry.ParentID,
		Role:     entry.Role,
		Code:     entry.ExitCode,
		Duration: entry.ExitTime.Sub(entry.Started),
	}
}

// KillAs stops an agent, but only if callerID is the parent or subreaper.
// Returns ErrNotOwner if the caller doesn't own the agent.
func (p *AgentWarden) KillAs(ctx context.Context, childID, callerID world.EntityID) error {
	p.mu.RLock()
	entry, ok := p.agents[childID]
	p.mu.RUnlock()

	if !ok {
		return fmt.Errorf("%w: %d", ErrNotFound, childID)
	}
	if entry.ParentID != callerID && callerID != p.subreaper {
		return fmt.Errorf("%w: caller %d is not parent %d of agent %d", ErrNotOwner, callerID, entry.ParentID, childID)
	}
	return p.Kill(ctx, childID)
}

// KillWithCode stops an agent and sets a specific exit code.
func (p *AgentWarden) KillWithCode(ctx context.Context, id world.EntityID, code ExitCode) error {
	p.mu.Lock()
	entry, ok := p.agents[id]
	if ok {
		entry.ExitCode = code
	}
	p.mu.Unlock()

	return p.Kill(ctx, id)
}

// KillChildren stops all direct children of parentID.
func (p *AgentWarden) KillChildren(ctx context.Context, parentID world.EntityID) error {
	children := p.Children(parentID)
	var firstErr error
	for _, childID := range children {
		if err := p.Kill(ctx, childID); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Children returns the entity IDs of all direct children of parentID.
// Uses supervises edges from World (GOL-14).
func (p *AgentWarden) Children(parentID world.EntityID) []world.EntityID {
	return p.world.Neighbors(parentID, world.Supervises, world.Outbound)
}

// Tree returns the hierarchical process tree rooted at rootID.
func (p *AgentWarden) Tree(rootID world.EntityID) *TreeNode {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.buildTreeLocked(rootID)
}

func (p *AgentWarden) buildTreeLocked(id world.EntityID) *TreeNode {
	entry, ok := p.agents[id]
	if !ok {
		return nil
	}

	node := &TreeNode{
		ID:       entry.ID,
		ParentID: entry.ParentID,
		Role:     entry.Role,
		State:    "running",
	}

	for _, child := range p.agents {
		if child.ParentID == id && child.ID != id {
			if childNode := p.buildTreeLocked(child.ID); childNode != nil {
				node.Children = append(node.Children, childNode)
			}
		}
	}

	return node
}

// SetSubreaper registers an agent as the orphan adopter.
// When any parent is killed, its children are reparented to the subreaper.
func (p *AgentWarden) SetSubreaper(id world.EntityID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.subreaper = id
}

// Subreaper returns the current subreaper agent ID.
func (p *AgentWarden) Subreaper() world.EntityID {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.subreaper
}

// Reparent changes a child's parent to a new parent.
func (p *AgentWarden) Reparent(childID, newParentID world.EntityID) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	entry, ok := p.agents[childID]
	if !ok {
		return fmt.Errorf("%w: %d", ErrNotFound, childID)
	}
	oldParent := entry.ParentID
	entry.ParentID = newParentID

	// Update supervises edges.
	if oldParent > 0 {
		_ = p.world.Unlink(oldParent, world.Supervises, childID)
	}
	if newParentID > 0 {
		_ = p.world.Link(newParentID, world.Supervises, childID)
	}
	return nil
}

// reparentOrphansLocked reparents all children of deadParentID to the subreaper.
// Must be called with p.mu held.
func (p *AgentWarden) reparentOrphansLocked(deadParentID world.EntityID) {
	for _, entry := range p.agents {
		if entry.ParentID == deadParentID && entry.ID != deadParentID {
			entry.ParentID = p.subreaper
			// Update supervises edges.
			_ = p.world.Unlink(deadParentID, world.Supervises, entry.ID)
			if p.subreaper > 0 {
				_ = p.world.Link(p.subreaper, world.Supervises, entry.ID)
			}
		}
	}
}

// SetAutoReap enables or disables automatic zombie cleanup for a parent's children.
// When enabled, children are fully cleaned up on exit without requiring Wait().
func (p *AgentWarden) SetAutoReap(parentID world.EntityID, enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if enabled {
		p.autoReap[parentID] = true
	} else {
		delete(p.autoReap, parentID)
	}
}

// ParentOf returns the parent ID of an agent, or 0 if not found.
func (p *AgentWarden) ParentOf(id world.EntityID) world.EntityID {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if e, ok := p.agents[id]; ok {
		return e.ParentID
	}
	if z, ok := p.zombies[id]; ok {
		return z.ParentID
	}
	return 0
}

// IsZombie returns true if the agent is finished but not yet reaped.
func (p *AgentWarden) IsZombie(id world.EntityID) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, ok := p.zombies[id]
	return ok
}

// Uptime returns how long an agent has been running, or its total runtime if finished.
func (p *AgentWarden) Uptime(id world.EntityID) time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if e, ok := p.agents[id]; ok {
		return time.Since(e.Started)
	}
	if z, ok := p.zombies[id]; ok {
		return z.ExitTime.Sub(z.Started)
	}
	return 0
}

package warden

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/dpopsuev/tangle/internal/transport"
	"github.com/dpopsuev/tangle/signal"
	"github.com/dpopsuev/tangle/world"
)

func setupWithTransport() (*AgentWarden, *transport.LocalTransport) {
	w := world.NewWorld()
	tr := transport.NewLocalTransport()
	log := signal.NewMemLog()
	launcher := newMockLauncher()
	pool := NewWarden(w, tr, log, launcher)
	return pool, tr
}

// TestEdgeE2E_SpawnTreeVerifyTopology proves the full edge lifecycle:
// Fork creates supervises edges, Children() queries edges, Kill removes edges.
func TestEdgeE2E_SpawnTreeVerifyTopology(t *testing.T) {
	pool, _ := setupWithTransport()
	ctx := context.Background()

	// Spawn: GenSec → 2Sec → Coder.
	gensec, _ := pool.Fork(ctx, "gensec", AgentConfig{}, 0)
	sec2, _ := pool.Fork(ctx, "2sec", AgentConfig{}, gensec)
	coder, _ := pool.Fork(ctx, "coder", AgentConfig{}, sec2)

	// Verify supervises topology.
	gensecChildren := pool.Children(gensec)
	if len(gensecChildren) != 1 || gensecChildren[0] != sec2 {
		t.Fatalf("gensec children = %v, want [%d]", gensecChildren, sec2)
	}

	sec2Children := pool.Children(sec2)
	if len(sec2Children) != 1 || sec2Children[0] != coder {
		t.Fatalf("2sec children = %v, want [%d]", sec2Children, coder)
	}

	coderChildren := pool.Children(coder)
	if len(coderChildren) != 0 {
		t.Fatalf("coder children = %v, want []", coderChildren)
	}

	// Total edges: gensec→2sec, 2sec→coder = 2.
	if pool.world.EdgeCount() != 2 {
		t.Fatalf("EdgeCount = %d, want 2", pool.world.EdgeCount())
	}

	// Kill coder — edge 2sec→coder removed.
	pool.Kill(ctx, coder)
	if pool.world.EdgeCount() != 1 {
		t.Fatalf("EdgeCount after kill coder = %d, want 1", pool.world.EdgeCount())
	}

	// Kill 2sec — edge gensec→2sec removed.
	pool.Kill(ctx, sec2)
	if pool.world.EdgeCount() != 0 {
		t.Fatalf("EdgeCount after kill 2sec = %d, want 0", pool.world.EdgeCount())
	}
}

// TestEdgeE2E_RouteGuardEnforcesDownwardVisibility proves that edge-aware
// routing blocks messages without communicates_with edges.
// GenSec → 2Sec: allowed (supervises implies communication).
// Coder → GenSec: blocked (no upward edge).
func TestEdgeE2E_RouteGuardEnforcesDownwardVisibility(t *testing.T) {
	pool, tr := setupWithTransport()
	ctx := context.Background()

	gensec, _ := pool.Fork(ctx, "gensec", AgentConfig{}, 0)
	coder, _ := pool.Fork(ctx, "coder", AgentConfig{}, gensec)

	// Add explicit communicates_with edge: gensec → coder (downward).
	_ = pool.world.Link(gensec, world.CommunicatesWith, coder)

	// Set route guard that requires communicates_with OR supervises edge.
	tr.SetRouteGuard(func(from, to transport.AgentID) error {
		fromID := agentIDToEntity(from)
		toID := agentIDToEntity(to)

		// Check outbound communicates_with (directional — sender must have the edge).
		for _, neighbor := range pool.world.Neighbors(fromID, world.CommunicatesWith, world.Outbound) {
			if neighbor == toID {
				return nil
			}
		}
		// Check supervises (parent can talk to child).
		for _, child := range pool.world.Neighbors(fromID, world.Supervises, world.Outbound) {
			if child == toID {
				return nil
			}
		}
		return errors.New("no edge: communication denied")
	})

	gensecTransportID := agentTransportID(gensec)
	coderTransportID := agentTransportID(coder)

	// GenSec → Coder: allowed (has communicates_with edge).
	_, err := tr.SendMessage(ctx, coderTransportID, transport.Message{From: gensecTransportID, Content: "task for you"})
	if err != nil {
		t.Fatalf("gensec→coder should be allowed: %v", err)
	}

	// Coder → GenSec: blocked (no upward edge).
	_, err = tr.SendMessage(ctx, gensecTransportID, transport.Message{From: coderTransportID, Content: "hey boss"})
	if err == nil {
		t.Fatal("coder→gensec should be blocked (no upward edge)")
	}
}

// agentIDToEntity extracts the entity ID from a transport agent ID ("agent-N" → N).
func agentIDToEntity(id transport.AgentID) world.EntityID {
	var n uint64
	fmt.Sscanf(string(id), "agent-%d", &n)
	return world.EntityID(n)
}

// facade_e2e_test.go — full-pipe E2E for Bugle v0.9.0+v0.10.0.
// Water through the pipe: Staff → Spawn → Listen → Ask → Broadcast →
// Kill → Wait → Orphan → KillAll. Proves the facade works end-to-end.
package testkit

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dpopsuev/jericho/internal/agent"
	"github.com/dpopsuev/jericho/internal/warden"
	"github.com/dpopsuev/jericho/signal"
	"github.com/dpopsuev/jericho/world"
)

type pipeLauncher struct {
	mu      sync.Mutex
	started map[world.EntityID]bool
	stopped map[world.EntityID]bool
}

func newPipeLauncher() *pipeLauncher {
	return &pipeLauncher{
		started: make(map[world.EntityID]bool),
		stopped: make(map[world.EntityID]bool),
	}
}

func (l *pipeLauncher) Start(_ context.Context, id world.EntityID, _ warden.AgentConfig) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.started[id] = true
	return nil
}

func (l *pipeLauncher) Stop(_ context.Context, id world.EntityID) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stopped[id] = true
	return nil
}

func (l *pipeLauncher) Healthy(_ context.Context, id world.EntityID) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.started[id] && !l.stopped[id]
}

// TestFacadeE2E_FullPipe tests the ENTIRE pipe from Staff creation
// to clean shutdown. This is the "turn on the water" test.
func TestFacadeE2E_FullPipe(t *testing.T) { //nolint:gocyclo // full-pipe E2E test is intentionally comprehensive
	launcher := newPipeLauncher()
	staff := agent.NewStaff(launcher)
	ctx := context.Background()

	// === 1. Spawn root agent (GenSec = PID 1) ===
	gensec, err := staff.Spawn(ctx, "gensec", warden.AgentConfig{})
	if err != nil {
		t.Fatalf("spawn gensec: %v", err)
	}
	if gensec.Role() != "gensec" {
		t.Fatalf("role = %q", gensec.Role())
	}
	if !gensec.IsAlive() {
		t.Fatal("gensec should be alive")
	}
	staff.SetSubreaper(gensec)

	// === 2. Spawn children under GenSec ===
	executor1, _ := gensec.Spawn(ctx, "executor", warden.AgentConfig{})
	executor2, _ := gensec.Spawn(ctx, "executor", warden.AgentConfig{})
	inspector, _ := gensec.Spawn(ctx, "inspector", warden.AgentConfig{})

	if staff.Count() != 4 {
		t.Fatalf("count = %d, want 4", staff.Count())
	}

	// === 3. Wire up handlers (Listen) ===
	executor1.Listen(func(content string) string {
		return "exec1: " + content
	})
	executor2.Listen(func(content string) string {
		return "exec2: " + content
	})
	inspector.Listen(func(content string) string {
		if strings.Contains(content, "FAIL") {
			return "REJECTED"
		}
		return "APPROVED"
	})

	// === 4. Ask — synchronous request-reply ===
	resp1, err := executor1.Perform(ctx, "build auth module")
	if err != nil {
		t.Fatalf("ask exec1: %v", err)
	}
	if resp1 != "exec1: build auth module" {
		t.Fatalf("resp1 = %q", resp1)
	}

	resp2, err := executor2.Perform(ctx, "build user module")
	if err != nil {
		t.Fatalf("ask exec2: %v", err)
	}
	if resp2 != "exec2: build user module" {
		t.Fatalf("resp2 = %q", resp2)
	}

	// === 5. Inspector reviews ===
	review, err := inspector.Perform(ctx, "code looks good")
	if err != nil {
		t.Fatalf("ask inspector: %v", err)
	}
	if review != "APPROVED" {
		t.Fatalf("review = %q, want APPROVED", review)
	}

	reviewFail, _ := inspector.Perform(ctx, "code has FAIL")
	if reviewFail != "REJECTED" {
		t.Fatalf("reviewFail = %q, want REJECTED", reviewFail)
	}

	// === 6. Broadcast to all executors ===
	var broadcastCount atomic.Int32
	// Re-register handlers that count broadcasts
	executor1.Listen(func(content string) string {
		broadcastCount.Add(1)
		return "ack"
	})
	executor2.Listen(func(content string) string {
		broadcastCount.Add(1)
		return "ack"
	})

	err = executor1.Broadcast(ctx, "rebuild all")
	if err != nil {
		t.Fatalf("broadcast: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if broadcastCount.Load() != 2 {
		t.Fatalf("broadcast received = %d, want 2", broadcastCount.Load())
	}

	// === 7. Tell (fire-and-forget) ===
	var tellReceived atomic.Value
	inspector.Listen(func(content string) string {
		tellReceived.Store(content)
		return ""
	})
	inspector.Tell("heads up: new PR")
	time.Sleep(50 * time.Millisecond)
	if tellReceived.Load() != "heads up: new PR" {
		t.Fatalf("tell not received")
	}

	// === 8. Hierarchy ===
	children := gensec.Children()
	if len(children) != 3 {
		t.Fatalf("gensec children = %d, want 3", len(children))
	}
	if executor1.Parent().ID() != gensec.ID() {
		t.Fatal("executor1 parent should be gensec")
	}

	// === 9. Progress tracking ===
	executor1.SetProgress(7, 10)
	prog, ok := executor1.Progress()
	if !ok || prog.Current != 7 || prog.Total != 10 {
		t.Fatalf("progress = %v/%v, want 7/10", prog.Current, prog.Total)
	}

	// === 10. Kill + Wait with exit code ===
	staff.Pool().SetAutoReap(gensec.ID(), false)
	executor1.KillWithReason(ctx, warden.ExitBudget)
	status1, err := executor1.Wait(ctx)
	if err != nil {
		t.Fatalf("wait exec1: %v", err)
	}
	if status1.Code != warden.ExitBudget {
		t.Fatalf("exit code = %d, want ExitBudget", status1.Code)
	}
	// Already reaped by Wait — should NOT be zombie anymore.

	// === 11. Orphan reparenting ===
	scheduler, _ := gensec.Spawn(ctx, "scheduler", warden.AgentConfig{})
	orphan1, _ := scheduler.Spawn(ctx, "worker", warden.AgentConfig{})
	orphan2, _ := scheduler.Spawn(ctx, "worker", warden.AgentConfig{})

	scheduler.Kill(ctx)

	// Orphans should be reparented to gensec (subreaper)
	if staff.Pool().ParentOf(orphan1.ID()) != gensec.ID() {
		t.Fatal("orphan1 should be reparented to gensec")
	}
	if staff.Pool().ParentOf(orphan2.ID()) != gensec.ID() {
		t.Fatal("orphan2 should be reparented to gensec")
	}

	// === 12. FindByRole ===
	workers := staff.FindByRole("worker")
	if len(workers) != 2 {
		t.Fatalf("workers = %d, want 2", len(workers))
	}

	// === 13. Signal observation ===
	var signalCount atomic.Int32
	staff.OnSignal(func(_ signal.Signal) {
		signalCount.Add(1)
	})

	// === 14. Clean shutdown ===
	staff.KillAll(ctx)
	if staff.Count() != 0 {
		t.Fatalf("count = %d after KillAll", staff.Count())
	}

	t.Logf("full pipe test passed: 4 agents spawned, messages exchanged, hierarchy verified, orphans reparented, clean shutdown")
}

// TestFacadeE2E_AIAsOperator — Agent 0 is an AI (bugle.Responder), not a human.
// Proves the World doesn't care whether Agent 0 is human or AI.
// This is the Origami production pattern: Claude/Cursor acts as operator.
// Per JRC-NED-4: World = primordial collective, Agent 0 = PID 0 (kernel).
func TestFacadeE2E_AIAsOperator(t *testing.T) {
	staff := agent.NewStaff(newPipeLauncher())
	ctx := context.Background()

	// AI operator (Agent 0) creates a GenSec (Agent 1).
	gensec, err := staff.Spawn(ctx, "gensec", warden.AgentConfig{
		RestartPolicy: warden.RestartOnFailure,
	})
	if err != nil {
		t.Fatalf("spawn gensec: %v", err)
	}
	gensec.Listen(func(content string) string {
		return "gensec: acknowledged " + content
	})

	// GenSec spawns workers — recursive agent creation (AI spawning AI).
	workers := make([]*agent.Solo, 0, 3)
	for i := range 3 {
		w, err := gensec.Spawn(ctx, fmt.Sprintf("worker-%d", i), warden.AgentConfig{})
		if err != nil {
			t.Fatalf("spawn worker-%d: %v", i, err)
		}
		idx := i
		w.Listen(func(content string) string {
			return fmt.Sprintf("worker-%d: done with %s", idx, content)
		})
		workers = append(workers, w)
	}

	// AI operator sends work through GenSec to workers.
	resp, err := gensec.Perform(ctx, "classify PTP defects")
	if err != nil {
		t.Fatalf("ask gensec: %v", err)
	}
	if !strings.Contains(resp, "acknowledged") {
		t.Fatalf("gensec resp = %q", resp)
	}

	// Workers do work (RespondTo pattern).
	for _, w := range workers {
		resp, err := w.Perform(ctx, "investigate")
		if err != nil {
			t.Fatalf("ask worker: %v", err)
		}
		if !strings.Contains(resp, "done with") {
			t.Fatalf("worker resp = %q", resp)
		}
	}

	// Verify hierarchy: all workers are children of GenSec.
	children := gensec.Children()
	if len(children) != 3 {
		t.Fatalf("gensec children = %d, want 3", len(children))
	}

	// Kill an intermediate worker — orphan reparenting should work.
	workers[1].Kill(ctx) //nolint:errcheck // test cleanup

	// Spawn a child under worker-0 to test recursive hierarchy.
	subWorker, err := workers[0].Spawn(ctx, "sub-worker", warden.AgentConfig{})
	if err != nil {
		t.Fatalf("spawn sub-worker: %v", err)
	}
	subWorker.Listen(func(content string) string {
		return "sub: " + content
	})

	// Kill worker-0 — sub-worker should be reparented.
	workers[0].Kill(ctx) //nolint:errcheck // test cleanup

	// Sub-worker is alive, reparented to subreaper (0 by default).
	if !subWorker.IsAlive() {
		t.Fatal("sub-worker should survive parent death (orphan reparenting)")
	}

	// AI operator does graceful shutdown.
	staff.KillAll(ctx)
	if staff.Count() != 0 {
		t.Fatalf("count after AI operator shutdown = %d", staff.Count())
	}
}

// TestFacadeE2E_StressTest — 20 agents, concurrent Ask, no races.
func TestFacadeE2E_StressTest(t *testing.T) {
	staff := agent.NewStaff(newPipeLauncher())
	ctx := context.Background()

	root, _ := staff.Spawn(ctx, "root", warden.AgentConfig{})

	// Spawn 20 workers under root.
	agents := make([]*agent.Solo, 0, 20)
	for i := range 20 {
		a, _ := root.Spawn(ctx, fmt.Sprintf("worker-%d", i), warden.AgentConfig{})
		a.Listen(func(content string) string {
			return "processed: " + content
		})
		agents = append(agents, a)
	}

	if staff.Count() != 21 {
		t.Fatalf("count = %d, want 21", staff.Count())
	}

	// Concurrent Ask to all 20 agents.
	var wg sync.WaitGroup
	var success atomic.Int32
	for _, a := range agents {
		wg.Add(1)
		go func(agent *agent.Solo) {
			defer wg.Done()
			resp, err := agent.Perform(ctx, "work")
			if err == nil && strings.HasPrefix(resp, "processed:") {
				success.Add(1)
			}
		}(a)
	}
	wg.Wait()

	if success.Load() != 20 {
		t.Fatalf("successful asks = %d, want 20", success.Load())
	}

	staff.KillAll(ctx)
	if staff.Count() != 0 {
		t.Fatalf("count after stress KillAll = %d", staff.Count())
	}
}

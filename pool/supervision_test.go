package pool

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/dpopsuev/jericho/signal"
	"github.com/dpopsuev/jericho/transport"
	"github.com/dpopsuev/jericho/world"
)

func setupSupervision() (*AgentPool, *mockLauncher) {
	w := world.NewWorld()
	t := transport.NewLocalTransport()
	bus := signal.NewMemBus()
	launcher := newMockLauncher()
	pool := New(w, t, bus, launcher)
	return pool, launcher
}

// === ZOMBIE REAPING ===

func TestWait_ReapsZombie(t *testing.T) {
	pool, _ := setupSupervision()
	ctx := context.Background()

	// GenSec (root) spawns Executor.
	gensec, _ := pool.Fork(ctx, "gensec", LaunchConfig{}, 0)
	pool.SetAutoReap(gensec, false) // explicit reaping
	executor, _ := pool.Fork(ctx, "executor", LaunchConfig{}, gensec)

	// Kill executor → becomes zombie.
	pool.Kill(ctx, executor)

	if !pool.IsZombie(executor) {
		t.Fatal("executor should be zombie after kill")
	}
	if pool.ZombieCount() != 1 {
		t.Fatalf("zombie count = %d, want 1", pool.ZombieCount())
	}

	// Wait reaps the zombie.
	status, err := pool.Wait(ctx, executor)
	if err != nil {
		t.Fatal(err)
	}
	if status.AgentID != executor {
		t.Fatalf("status.AgentID = %d, want %d", status.AgentID, executor)
	}
	if status.Role != "executor" {
		t.Fatalf("status.Role = %q, want executor", status.Role)
	}
	if status.Code != ExitSuccess {
		t.Fatalf("status.Code = %d, want ExitSuccess", status.Code)
	}
	if status.Duration <= 0 {
		t.Fatal("duration should be positive")
	}

	// After reap, zombie is gone.
	if pool.IsZombie(executor) {
		t.Fatal("executor should not be zombie after Wait")
	}
	if pool.ZombieCount() != 0 {
		t.Fatalf("zombie count = %d after reap, want 0", pool.ZombieCount())
	}
}

func TestWaitAny_NonBlocking(t *testing.T) {
	pool, _ := setupSupervision()
	ctx := context.Background()

	gensec, _ := pool.Fork(ctx, "gensec", LaunchConfig{}, 0)
	pool.SetAutoReap(gensec, false)

	// No zombies yet.
	status := pool.WaitAny(gensec)
	if status != nil {
		t.Fatal("WaitAny should return nil when no zombies")
	}

	// Spawn and kill a child.
	child, _ := pool.Fork(ctx, "worker", LaunchConfig{}, gensec)
	pool.Kill(ctx, child)

	// Now WaitAny should find the zombie.
	status = pool.WaitAny(gensec)
	if status == nil {
		t.Fatal("WaitAny should return zombie status")
	}
	if status.Role != "worker" {
		t.Fatalf("role = %q, want worker", status.Role)
	}
}

func TestWait_BlocksUntilDone(t *testing.T) {
	pool, _ := setupSupervision()
	ctx := context.Background()

	gensec, _ := pool.Fork(ctx, "gensec", LaunchConfig{}, 0)
	pool.SetAutoReap(gensec, false)
	child, _ := pool.Fork(ctx, "worker", LaunchConfig{}, gensec)

	var status *ExitStatus
	done := make(chan struct{})
	go func() {
		var err error
		status, err = pool.Wait(ctx, child)
		if err != nil {
			t.Error(err)
		}
		close(done)
	}()

	// Wait should be blocking.
	select {
	case <-done:
		t.Fatal("Wait should block until child is killed")
	case <-time.After(50 * time.Millisecond):
		// Good — still blocking.
	}

	// Kill the child.
	pool.Kill(ctx, child)

	// Wait should unblock.
	select {
	case <-done:
		if status == nil {
			t.Fatal("status should not be nil")
		}
	case <-time.After(time.Second):
		t.Fatal("Wait should unblock after Kill")
	}
}

// === EXIT CODES ===

func TestKillWithCode_Error(t *testing.T) {
	pool, _ := setupSupervision()
	ctx := context.Background()

	gensec, _ := pool.Fork(ctx, "gensec", LaunchConfig{}, 0)
	pool.SetAutoReap(gensec, false)
	child, _ := pool.Fork(ctx, "worker", LaunchConfig{}, gensec)

	pool.KillWithCode(ctx, child, ExitError)

	status, _ := pool.Wait(ctx, child)
	if status.Code != ExitError {
		t.Fatalf("code = %d, want ExitError", status.Code)
	}
}

func TestKillWithCode_Budget(t *testing.T) {
	pool, _ := setupSupervision()
	ctx := context.Background()

	gensec, _ := pool.Fork(ctx, "gensec", LaunchConfig{}, 0)
	pool.SetAutoReap(gensec, false)
	child, _ := pool.Fork(ctx, "worker", LaunchConfig{}, gensec)

	pool.KillWithCode(ctx, child, ExitBudget)
	status, _ := pool.Wait(ctx, child)
	if status.Code != ExitBudget {
		t.Fatalf("code = %d, want ExitBudget", status.Code)
	}
}

func TestKillWithCode_Timeout(t *testing.T) {
	pool, _ := setupSupervision()
	ctx := context.Background()

	gensec, _ := pool.Fork(ctx, "gensec", LaunchConfig{}, 0)
	pool.SetAutoReap(gensec, false)
	child, _ := pool.Fork(ctx, "worker", LaunchConfig{}, gensec)

	pool.KillWithCode(ctx, child, ExitTimeout)
	status, _ := pool.Wait(ctx, child)
	if status.Code != ExitTimeout {
		t.Fatalf("code = %d, want ExitTimeout", status.Code)
	}
}

// === OWNERSHIP KILL ===

func TestKillAs_OwnerSucceeds(t *testing.T) {
	pool, _ := setupSupervision()
	ctx := context.Background()

	parent, _ := pool.Fork(ctx, "gensec", LaunchConfig{}, 0)
	child, _ := pool.Fork(ctx, "executor", LaunchConfig{}, parent)

	err := pool.KillAs(ctx, child, parent)
	if err != nil {
		t.Fatalf("owner should be able to kill child: %v", err)
	}
}

func TestKillAs_NonOwnerFails(t *testing.T) {
	pool, _ := setupSupervision()
	ctx := context.Background()

	parent, _ := pool.Fork(ctx, "gensec", LaunchConfig{}, 0)
	child, _ := pool.Fork(ctx, "executor", LaunchConfig{}, parent)
	sibling, _ := pool.Fork(ctx, "auditor", LaunchConfig{}, parent)

	err := pool.KillAs(ctx, child, sibling)
	if err == nil {
		t.Fatal("sibling should NOT be able to kill child")
	}
	if !errors.Is(err, ErrNotOwner) {
		t.Fatalf("err = %v, want ErrNotOwner", err)
	}
}

func TestKillAs_SubreaperCanKill(t *testing.T) {
	pool, _ := setupSupervision()
	ctx := context.Background()

	gensec, _ := pool.Fork(ctx, "gensec", LaunchConfig{}, 0)
	pool.SetSubreaper(gensec)

	other, _ := pool.Fork(ctx, "scheduler", LaunchConfig{}, 0)
	child, _ := pool.Fork(ctx, "executor", LaunchConfig{}, other)

	// Subreaper can kill anyone's children.
	err := pool.KillAs(ctx, child, gensec)
	if err != nil {
		t.Fatalf("subreaper should be able to kill: %v", err)
	}
}

func TestKillChildren(t *testing.T) {
	pool, _ := setupSupervision()
	ctx := context.Background()

	parent, _ := pool.Fork(ctx, "gensec", LaunchConfig{}, 0)
	pool.Fork(ctx, "executor", LaunchConfig{}, parent)
	pool.Fork(ctx, "auditor", LaunchConfig{}, parent)
	pool.Fork(ctx, "scheduler", LaunchConfig{}, parent)

	if len(pool.Children(parent)) != 3 {
		t.Fatalf("children = %d, want 3", len(pool.Children(parent)))
	}

	pool.KillChildren(ctx, parent)

	if len(pool.Children(parent)) != 0 {
		t.Fatalf("children = %d after KillChildren, want 0", len(pool.Children(parent)))
	}
}

// === ORPHAN REPARENTING ===

func TestOrphan_ReparentedToSubreaper(t *testing.T) {
	pool, _ := setupSupervision()
	ctx := context.Background()

	gensec, _ := pool.Fork(ctx, "gensec", LaunchConfig{}, 0)
	pool.SetSubreaper(gensec)

	scheduler, _ := pool.Fork(ctx, "scheduler", LaunchConfig{}, gensec)
	exec1, _ := pool.Fork(ctx, "executor-1", LaunchConfig{}, scheduler)
	exec2, _ := pool.Fork(ctx, "executor-2", LaunchConfig{}, scheduler)

	// Kill scheduler → orphans should be reparented to subreaper (gensec).
	pool.Kill(ctx, scheduler)

	if pool.ParentOf(exec1) != gensec {
		t.Fatalf("exec1 parent = %d, want gensec %d", pool.ParentOf(exec1), gensec)
	}
	if pool.ParentOf(exec2) != gensec {
		t.Fatalf("exec2 parent = %d, want gensec %d", pool.ParentOf(exec2), gensec)
	}

	// Orphans are now children of gensec.
	children := pool.Children(gensec)
	if len(children) != 2 {
		t.Fatalf("gensec children = %d, want 2 (adopted orphans)", len(children))
	}
}

func TestOrphan_DefaultSubreaperIsZero(t *testing.T) {
	pool, _ := setupSupervision()
	ctx := context.Background()

	parent, _ := pool.Fork(ctx, "parent", LaunchConfig{}, 0)
	child, _ := pool.Fork(ctx, "child", LaunchConfig{}, parent)

	// No subreaper set — orphans go to root (0).
	pool.Kill(ctx, parent)
	if pool.ParentOf(child) != 0 {
		t.Fatalf("orphan parent = %d, want 0 (default)", pool.ParentOf(child))
	}
}

// === TREE ===

func TestTree_ThreeLevels(t *testing.T) {
	pool, _ := setupSupervision()
	ctx := context.Background()

	root, _ := pool.Fork(ctx, "gensec", LaunchConfig{}, 0)
	mid, _ := pool.Fork(ctx, "scheduler", LaunchConfig{}, root)
	pool.Fork(ctx, "executor-1", LaunchConfig{}, mid)
	pool.Fork(ctx, "executor-2", LaunchConfig{}, mid)

	tree := pool.Tree(root)
	if tree == nil {
		t.Fatal("tree should not be nil")
	}
	if tree.Role != "gensec" {
		t.Fatalf("root role = %q", tree.Role)
	}
	if len(tree.Children) != 1 {
		t.Fatalf("root children = %d, want 1 (scheduler)", len(tree.Children))
	}
	if len(tree.Children[0].Children) != 2 {
		t.Fatalf("scheduler children = %d, want 2", len(tree.Children[0].Children))
	}
}

func TestChildren_DirectOnly(t *testing.T) {
	pool, _ := setupSupervision()
	ctx := context.Background()

	root, _ := pool.Fork(ctx, "gensec", LaunchConfig{}, 0)
	mid, _ := pool.Fork(ctx, "scheduler", LaunchConfig{}, root)
	pool.Fork(ctx, "executor", LaunchConfig{}, mid)

	// Root's direct children = [scheduler]. NOT [scheduler, executor].
	children := pool.Children(root)
	if len(children) != 1 {
		t.Fatalf("direct children = %d, want 1", len(children))
	}
}

// === AUTO-REAP ===

func TestAutoReap_SkipsZombie(t *testing.T) {
	pool, _ := setupSupervision()
	ctx := context.Background()

	parent, _ := pool.Fork(ctx, "gensec", LaunchConfig{}, 0)
	pool.SetAutoReap(parent, true)
	child, _ := pool.Fork(ctx, "worker", LaunchConfig{}, parent)

	pool.Kill(ctx, child)

	// With auto-reap, no zombie should be created.
	if pool.IsZombie(child) {
		t.Fatal("auto-reaped child should not be zombie")
	}
	if pool.ZombieCount() != 0 {
		t.Fatalf("zombie count = %d, want 0", pool.ZombieCount())
	}
}

func TestAutoReap_DisabledCreatesZombie(t *testing.T) {
	pool, _ := setupSupervision()
	ctx := context.Background()

	parent, _ := pool.Fork(ctx, "gensec", LaunchConfig{}, 0)
	pool.SetAutoReap(parent, false) // explicit
	child, _ := pool.Fork(ctx, "worker", LaunchConfig{}, parent)

	pool.Kill(ctx, child)

	if !pool.IsZombie(child) {
		t.Fatal("without auto-reap, child should be zombie")
	}
}

// === PARENT TRACKING ===

func TestParentOf(t *testing.T) {
	pool, _ := setupSupervision()
	ctx := context.Background()

	parent, _ := pool.Fork(ctx, "gensec", LaunchConfig{}, 0)
	child, _ := pool.Fork(ctx, "executor", LaunchConfig{}, parent)

	if pool.ParentOf(child) != parent {
		t.Fatalf("parent = %d, want %d", pool.ParentOf(child), parent)
	}
	if pool.ParentOf(parent) != 0 {
		t.Fatalf("root parent = %d, want 0", pool.ParentOf(parent))
	}
}

func TestFork_AttachesHierarchyComponent(t *testing.T) {
	pool, _ := setupSupervision()
	ctx := context.Background()

	parent, _ := pool.Fork(ctx, "gensec", LaunchConfig{}, 0)
	child, _ := pool.Fork(ctx, "executor", LaunchConfig{}, parent)

	hier, ok := world.TryGet[world.Hierarchy](pool.world, child)
	if !ok {
		t.Fatal("Hierarchy component should be attached")
	}
	if hier.Parent != parent {
		t.Fatalf("Hierarchy.Parent = %d, want %d", hier.Parent, parent)
	}
}

// === CONCURRENT SUPERVISION ===

func TestConcurrent_ForkWaitKill(t *testing.T) {
	pool, _ := setupSupervision()
	ctx := context.Background()

	gensec, _ := pool.Fork(ctx, "gensec", LaunchConfig{}, 0)
	pool.SetAutoReap(gensec, false)

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			child, err := pool.Fork(ctx, "worker", LaunchConfig{}, gensec)
			if err != nil {
				t.Error(err)
				return
			}
			pool.Kill(ctx, child)
			pool.Wait(ctx, child) //nolint:errcheck // test cleanup, error irrelevant
		}()
	}
	wg.Wait()

	if pool.Count() != 1 { // only gensec remains
		t.Fatalf("count = %d, want 1 (gensec only)", pool.Count())
	}
	if pool.ZombieCount() != 0 {
		t.Fatalf("zombie count = %d, want 0 (all reaped)", pool.ZombieCount())
	}
}

// === FULL LIFECYCLE SIMULATION ===

func TestSupervision_FullLifecycleSimulation(t *testing.T) {
	pool, launcher := setupSupervision()
	ctx := context.Background()

	// 1. GenSec boots as root agent (PID 1).
	gensec, err := pool.Fork(ctx, "gensec", LaunchConfig{Model: "haiku"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	pool.SetSubreaper(gensec)
	pool.SetAutoReap(gensec, false) // GenSec explicitly reaps

	// 2. GenSec spawns Scheduler.
	scheduler, _ := pool.Fork(ctx, "scheduler", LaunchConfig{Model: "sonnet"}, gensec)

	// 3. Scheduler spawns 3 Executors.
	pool.SetAutoReap(scheduler, false)
	exec1, _ := pool.Fork(ctx, "executor-1", LaunchConfig{Model: "opus"}, scheduler)
	exec2, _ := pool.Fork(ctx, "executor-2", LaunchConfig{Model: "opus"}, scheduler)
	exec3, _ := pool.Fork(ctx, "executor-3", LaunchConfig{Model: "opus"}, scheduler)

	// Verify tree: gensec → scheduler → [exec1, exec2, exec3]
	if pool.Count() != 5 {
		t.Fatalf("count = %d, want 5", pool.Count())
	}
	tree := pool.Tree(gensec)
	if len(tree.Children) != 1 {
		t.Fatalf("gensec children = %d, want 1", len(tree.Children))
	}
	if len(tree.Children[0].Children) != 3 {
		t.Fatalf("scheduler children = %d, want 3", len(tree.Children[0].Children))
	}

	// 4. Executor-1 finishes successfully.
	pool.KillWithCode(ctx, exec1, ExitSuccess)
	status := pool.WaitAny(scheduler)
	if status == nil {
		t.Fatal("should have zombie to reap")
	}
	if status.Code != ExitSuccess {
		t.Fatalf("exec1 exit = %d, want success", status.Code)
	}

	// 5. Executor-2 hits budget limit.
	pool.KillWithCode(ctx, exec2, ExitBudget)
	status, _ = pool.Wait(ctx, exec2)
	if status.Code != ExitBudget {
		t.Fatalf("exec2 exit = %d, want budget", status.Code)
	}

	// 6. Executor-3 errors.
	pool.KillWithCode(ctx, exec3, ExitError)
	status, _ = pool.Wait(ctx, exec3)
	if status.Code != ExitError {
		t.Fatalf("exec3 exit = %d, want error", status.Code)
	}

	// 7. All executors reaped — no zombies under scheduler.
	if pool.ZombieCount() != 0 {
		t.Fatalf("zombies = %d, want 0", pool.ZombieCount())
	}

	// 8. Scheduler crashes → its orphans would be reparented.
	// (No orphans now since all executors are dead.)
	pool.Kill(ctx, scheduler)
	schedStatus := pool.WaitAny(gensec)
	if schedStatus == nil {
		t.Fatal("scheduler should be zombie under gensec")
	}

	// 9. Verify cleanup: only GenSec remains.
	if pool.Count() != 1 {
		t.Fatalf("count = %d, want 1 (gensec only)", pool.Count())
	}

	// 10. Verify all launchers were stopped.
	if !launcher.stopped[exec1] {
		t.Fatal("exec1 not stopped")
	}
	if !launcher.stopped[exec2] {
		t.Fatal("exec2 not stopped")
	}
	if !launcher.stopped[exec3] {
		t.Fatal("exec3 not stopped")
	}
	if !launcher.stopped[scheduler] {
		t.Fatal("scheduler not stopped")
	}

	// 11. GenSec shuts down.
	pool.KillAll(ctx)
	if pool.Count() != 0 {
		t.Fatalf("count = %d after KillAll, want 0", pool.Count())
	}
	if pool.ZombieCount() != 0 {
		t.Fatalf("zombies = %d after KillAll, want 0", pool.ZombieCount())
	}
}

// === ORPHAN SIMULATION ===

func TestSupervision_OrphanSimulation(t *testing.T) {
	pool, _ := setupSupervision()
	ctx := context.Background()

	// GenSec as subreaper.
	gensec, _ := pool.Fork(ctx, "gensec", LaunchConfig{}, 0)
	pool.SetSubreaper(gensec)
	pool.SetAutoReap(gensec, false)

	// Scheduler spawns executors.
	scheduler, _ := pool.Fork(ctx, "scheduler", LaunchConfig{}, gensec)
	exec1, _ := pool.Fork(ctx, "executor-1", LaunchConfig{}, scheduler)
	exec2, _ := pool.Fork(ctx, "executor-2", LaunchConfig{}, scheduler)

	// Scheduler dies unexpectedly → executors orphaned → reparented to GenSec.
	pool.Kill(ctx, scheduler)

	// Executors should now belong to GenSec.
	if pool.ParentOf(exec1) != gensec {
		t.Fatal("exec1 should be reparented to gensec")
	}
	if pool.ParentOf(exec2) != gensec {
		t.Fatal("exec2 should be reparented to gensec")
	}

	// GenSec can now manage them.
	children := pool.Children(gensec)
	if len(children) != 2 {
		t.Fatalf("gensec children = %d, want 2", len(children))
	}

	// GenSec kills adopted orphans.
	pool.KillChildren(ctx, gensec)

	// Reap scheduler zombie.
	pool.WaitAny(gensec) // scheduler

	// Verify all clean.
	if pool.Count() != 1 { // only gensec
		t.Fatalf("count = %d, want 1", pool.Count())
	}
}

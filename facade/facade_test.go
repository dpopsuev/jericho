package facade

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dpopsuev/jericho/pool"
	"github.com/dpopsuev/jericho/signal"
	"github.com/dpopsuev/jericho/world"
)

// ---------------------------------------------------------------------------
// mockLauncher — same pattern as pool/pool_test.go
// ---------------------------------------------------------------------------

type mockLauncher struct {
	mu      sync.Mutex
	started map[world.EntityID]bool
	stopped map[world.EntityID]bool
}

func newMockLauncher() *mockLauncher {
	return &mockLauncher{
		started: make(map[world.EntityID]bool),
		stopped: make(map[world.EntityID]bool),
	}
}

func (m *mockLauncher) Start(_ context.Context, id world.EntityID, _ pool.LaunchConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started[id] = true
	return nil
}

func (m *mockLauncher) Stop(_ context.Context, id world.EntityID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped[id] = true
	return nil
}

func (m *mockLauncher) Healthy(_ context.Context, id world.EntityID) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.started[id] && !m.stopped[id]
}

// setup creates a Staff with a mock launcher.
func setup() *Staff {
	ml := newMockLauncher()
	return NewStaff(ml)
}

// ---------------------------------------------------------------------------
// Staff tests
// ---------------------------------------------------------------------------

func TestStaff_SpawnReturnsHandle(t *testing.T) {
	s := setup()
	ctx := context.Background()

	h, err := s.Spawn(ctx, "executor", pool.LaunchConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if h.ID() == 0 {
		t.Fatal("handle ID should not be 0")
	}
	if h.Role() != "executor" {
		t.Fatalf("role = %q, want executor", h.Role())
	}
}

func TestStaff_SpawnUnder_CreatesChild(t *testing.T) {
	s := setup()
	ctx := context.Background()

	parent, err := s.Spawn(ctx, "manager", pool.LaunchConfig{})
	if err != nil {
		t.Fatal(err)
	}

	child, err := s.SpawnUnder(ctx, parent, "executor", pool.LaunchConfig{})
	if err != nil {
		t.Fatal(err)
	}

	p := child.Parent()
	if p == nil {
		t.Fatal("child.Parent() should not be nil")
	}
	if p.ID() != parent.ID() {
		t.Fatalf("child parent = %d, want %d", p.ID(), parent.ID())
	}
}

func TestStaff_Active(t *testing.T) {
	s := setup()
	ctx := context.Background()

	for range 3 {
		if _, err := s.Spawn(ctx, "worker", pool.LaunchConfig{}); err != nil {
			t.Fatal(err)
		}
	}

	active := s.Active()
	if len(active) != 3 {
		t.Fatalf("active = %d, want 3", len(active))
	}
}

func TestStaff_FindByRole(t *testing.T) {
	s := setup()
	ctx := context.Background()

	s.Spawn(ctx, "executor", pool.LaunchConfig{})
	s.Spawn(ctx, "executor", pool.LaunchConfig{})
	s.Spawn(ctx, "inspector", pool.LaunchConfig{})

	executors := s.FindByRole("executor")
	if len(executors) != 2 {
		t.Fatalf("FindByRole(executor) = %d, want 2", len(executors))
	}

	inspectors := s.FindByRole("inspector")
	if len(inspectors) != 1 {
		t.Fatalf("FindByRole(inspector) = %d, want 1", len(inspectors))
	}
}

func TestStaff_KillAll(t *testing.T) {
	s := setup()
	ctx := context.Background()

	for range 3 {
		s.Spawn(ctx, "worker", pool.LaunchConfig{})
	}
	if s.Count() != 3 {
		t.Fatalf("count = %d, want 3", s.Count())
	}

	s.KillAll(ctx)
	if s.Count() != 0 {
		t.Fatalf("count = %d after KillAll, want 0", s.Count())
	}
}

func TestStaff_Tree(t *testing.T) {
	s := setup()
	ctx := context.Background()

	root, _ := s.Spawn(ctx, "manager", pool.LaunchConfig{})
	mid, _ := s.SpawnUnder(ctx, root, "executor", pool.LaunchConfig{})
	s.SpawnUnder(ctx, mid, "inspector", pool.LaunchConfig{})

	tree := s.Tree(root)
	if tree == nil {
		t.Fatal("tree should not be nil")
	}
	if tree.Role != "manager" {
		t.Fatalf("root role = %q", tree.Role)
	}
	if len(tree.Children) != 1 {
		t.Fatalf("root children = %d, want 1", len(tree.Children))
	}
	if tree.Children[0].Role != "executor" {
		t.Fatalf("mid role = %q", tree.Children[0].Role)
	}
	if len(tree.Children[0].Children) != 1 {
		t.Fatalf("mid children = %d, want 1", len(tree.Children[0].Children))
	}
	if tree.Children[0].Children[0].Role != "inspector" {
		t.Fatalf("leaf role = %q", tree.Children[0].Children[0].Role)
	}
}

func TestStaff_Count(t *testing.T) {
	s := setup()
	ctx := context.Background()

	if s.Count() != 0 {
		t.Fatalf("initial count = %d", s.Count())
	}

	h, _ := s.Spawn(ctx, "worker", pool.LaunchConfig{})
	if s.Count() != 1 {
		t.Fatalf("count = %d after spawn", s.Count())
	}

	h.Kill(ctx)
	if s.Count() != 0 {
		t.Fatalf("count = %d after kill", s.Count())
	}
}

// ---------------------------------------------------------------------------
// Handle tests
// ---------------------------------------------------------------------------

func TestHandle_String(t *testing.T) {
	s := setup()
	ctx := context.Background()

	h, _ := s.Spawn(ctx, "executor", pool.LaunchConfig{})
	want := fmt.Sprintf("executor(agent-%d)", h.ID())
	if h.String() != want {
		t.Fatalf("String() = %q, want %q", h.String(), want)
	}
}

func TestHandle_IsAlive(t *testing.T) {
	s := setup()
	ctx := context.Background()

	h, _ := s.Spawn(ctx, "worker", pool.LaunchConfig{})
	if !h.IsAlive() {
		t.Fatal("should be alive after spawn")
	}

	h.Kill(ctx)
	if h.IsAlive() {
		t.Fatal("should not be alive after kill")
	}
}

func TestHandle_IsHealthy(t *testing.T) {
	s := setup()
	ctx := context.Background()

	h, _ := s.Spawn(ctx, "worker", pool.LaunchConfig{})
	if !h.IsHealthy() {
		t.Fatal("should be healthy after spawn")
	}
}

func TestHandle_IsZombie(t *testing.T) {
	s := setup()
	ctx := context.Background()

	// Disable auto-reap for parent=0 so zombies accumulate.
	s.Pool().SetAutoReap(0, false)

	h, _ := s.Spawn(ctx, "worker", pool.LaunchConfig{})
	if h.IsZombie() {
		t.Fatal("should not be zombie before kill")
	}

	h.Kill(ctx)
	if !h.IsZombie() {
		t.Fatal("should be zombie after kill (no reap)")
	}

	// Wait reaps the zombie.
	h.Wait(ctx)
	if h.IsZombie() {
		t.Fatal("should not be zombie after Wait")
	}
}

func TestHandle_Ask(t *testing.T) {
	s := setup()
	ctx := context.Background()

	h, _ := s.Spawn(ctx, "echo", pool.LaunchConfig{})
	// Register echo handler.
	h.Listen(func(content string) string {
		return "echo:" + content
	})

	resp, err := h.Ask(ctx, "hello")
	if err != nil {
		t.Fatal(err)
	}
	if resp != "echo:hello" {
		t.Fatalf("response = %q, want echo:hello", resp)
	}
}

func TestHandle_Tell(t *testing.T) {
	s := setup()
	ctx := context.Background()

	h, _ := s.Spawn(ctx, "worker", pool.LaunchConfig{})

	var received atomic.Value
	h.Listen(func(content string) string {
		received.Store(content)
		return "ok"
	})

	err := h.Tell("ping")
	if err != nil {
		t.Fatal(err)
	}

	// Give the async handler time to execute.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if v := received.Load(); v != nil {
			if v.(string) == "ping" {
				return
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("handler did not receive Tell message")
}

func TestHandle_Broadcast(t *testing.T) {
	s := setup()
	ctx := context.Background()

	var counters [3]atomic.Int32
	for i := range 3 {
		h, _ := s.Spawn(ctx, "listener", pool.LaunchConfig{})
		idx := i
		h.Listen(func(_ string) string {
			counters[idx].Add(1)
			return "ok"
		})
	}

	// Broadcast from any handle with role "listener".
	first := s.FindByRole("listener")[0]
	err := first.Broadcast(ctx, "ping-all")
	if err != nil {
		t.Fatal(err)
	}

	// Wait for all handlers to fire.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		total := int32(0)
		for i := range 3 {
			total += counters[i].Load()
		}
		if total >= 3 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("broadcast did not reach all 3 agents")
}

func TestHandle_Listen(t *testing.T) {
	s := setup()
	ctx := context.Background()

	h, _ := s.Spawn(ctx, "responder", pool.LaunchConfig{})
	h.Listen(func(content string) string {
		return "got:" + content
	})

	resp, err := h.Ask(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	if resp != "got:test" {
		t.Fatalf("response = %q, want got:test", resp)
	}
}

func TestHandle_Spawn_Child(t *testing.T) {
	s := setup()
	ctx := context.Background()

	parent, _ := s.Spawn(ctx, "manager", pool.LaunchConfig{})
	child, err := parent.Spawn(ctx, "executor", pool.LaunchConfig{})
	if err != nil {
		t.Fatal(err)
	}

	children := parent.Children()
	if len(children) != 1 {
		t.Fatalf("children = %d, want 1", len(children))
	}
	if children[0].ID() != child.ID() {
		t.Fatalf("child ID = %d, want %d", children[0].ID(), child.ID())
	}
}

func TestHandle_Kill_Wait(t *testing.T) {
	s := setup()
	ctx := context.Background()

	// Disable auto-reap so we can Wait.
	s.Pool().SetAutoReap(0, false)

	h, _ := s.Spawn(ctx, "worker", pool.LaunchConfig{})
	h.Kill(ctx)

	status, err := h.Wait(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status == nil {
		t.Fatal("status should not be nil")
	}
	if status.Code != pool.ExitSuccess {
		t.Fatalf("exit code = %d, want ExitSuccess(0)", status.Code)
	}
}

func TestHandle_KillWithReason(t *testing.T) {
	s := setup()
	ctx := context.Background()

	// Disable auto-reap so we can Wait.
	s.Pool().SetAutoReap(0, false)

	h, _ := s.Spawn(ctx, "worker", pool.LaunchConfig{})
	h.KillWithReason(ctx, pool.ExitBudget)

	status, err := h.Wait(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status == nil {
		t.Fatal("status should not be nil")
	}
	if status.Code != pool.ExitBudget {
		t.Fatalf("exit code = %d, want ExitBudget(2)", status.Code)
	}
}

func TestHandle_Children_Parent(t *testing.T) {
	s := setup()
	ctx := context.Background()

	parent, _ := s.Spawn(ctx, "manager", pool.LaunchConfig{})
	child, _ := s.SpawnUnder(ctx, parent, "executor", pool.LaunchConfig{})

	// Parent → Children includes child.
	children := parent.Children()
	found := false
	for _, c := range children {
		if c.ID() == child.ID() {
			found = true
		}
	}
	if !found {
		t.Fatal("parent.Children should include child")
	}

	// Child → Parent points back.
	p := child.Parent()
	if p == nil {
		t.Fatal("child.Parent should not be nil")
	}
	if p.ID() != parent.ID() {
		t.Fatalf("child.Parent().ID() = %d, want %d", p.ID(), parent.ID())
	}

	// Root agent has no parent.
	if parent.Parent() != nil {
		t.Fatal("root agent should have nil Parent()")
	}
}

func TestHandle_SetProgress(t *testing.T) {
	s := setup()
	ctx := context.Background()

	h, _ := s.Spawn(ctx, "worker", pool.LaunchConfig{})
	h.SetProgress(5, 10)

	p, ok := h.Progress()
	if !ok {
		t.Fatal("progress not attached")
	}
	if p.Current != 5 {
		t.Fatalf("current = %d, want 5", p.Current)
	}
	if p.Total != 10 {
		t.Fatalf("total = %d, want 10", p.Total)
	}
	if p.Percent != 50 {
		t.Fatalf("percent = %f, want 50", p.Percent)
	}
}

// ---------------------------------------------------------------------------
// Integration tests
// ---------------------------------------------------------------------------

func TestFacade_FullPipeline(t *testing.T) {
	s := setup()
	ctx := context.Background()

	// GenSec spawns an Executor.
	gensec, _ := s.Spawn(ctx, "gensec", pool.LaunchConfig{})
	executor, _ := gensec.Spawn(ctx, "executor", pool.LaunchConfig{})

	// Executor listens and echoes.
	executor.Listen(func(content string) string {
		return "executed:" + content
	})

	// GenSec asks Executor.
	resp, err := executor.Ask(ctx, "compile")
	if err != nil {
		t.Fatal(err)
	}
	if resp != "executed:compile" {
		t.Fatalf("response = %q", resp)
	}

	// Disable auto-reap so we can Wait on executor.
	s.Pool().SetAutoReap(gensec.ID(), false)

	// Kill Executor and Wait.
	executor.Kill(ctx)
	status, err := executor.Wait(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status == nil {
		t.Fatal("status should not be nil")
	}

	// KillAll.
	s.KillAll(ctx)
	if s.Count() != 0 {
		t.Fatalf("count = %d after KillAll", s.Count())
	}
}

func TestFacade_ConcurrentSpawnAskKill(t *testing.T) {
	s := setup()
	ctx := context.Background()

	var wg sync.WaitGroup

	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			h, err := s.Spawn(ctx, "worker", pool.LaunchConfig{})
			if err != nil {
				t.Error(err)
				return
			}

			h.Listen(func(content string) string {
				return "reply:" + content
			})

			resp, err := h.Ask(ctx, "ping")
			if err != nil {
				t.Error(err)
				return
			}
			if resp != "reply:ping" {
				t.Errorf("response = %q", resp)
				return
			}

			if err := h.Kill(ctx); err != nil {
				t.Error(err)
			}
		}()
	}

	wg.Wait()

	if s.Count() != 0 {
		t.Fatalf("count = %d after concurrent test", s.Count())
	}
}

func TestStaff_OnSignal(t *testing.T) {
	s := setup()
	ctx := context.Background()

	var signals []signal.Signal
	var mu sync.Mutex
	s.OnSignal(func(sig signal.Signal) {
		mu.Lock()
		signals = append(signals, sig)
		mu.Unlock()
	})

	h, _ := s.Spawn(ctx, "worker", pool.LaunchConfig{})
	h.Kill(ctx)

	mu.Lock()
	count := len(signals)
	mu.Unlock()

	if count < 2 {
		t.Fatalf("expected at least 2 signals (started+stopped), got %d", count)
	}
}

func TestStaff_EscapeHatches(t *testing.T) {
	s := setup()

	if s.World() == nil {
		t.Fatal("World() should not be nil")
	}
	if s.Pool() == nil {
		t.Fatal("Pool() should not be nil")
	}
	if s.Transport() == nil {
		t.Fatal("Transport() should not be nil")
	}
	if s.Bus() == nil {
		t.Fatal("Bus() should not be nil")
	}
}

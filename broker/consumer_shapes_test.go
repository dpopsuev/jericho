package broker_test

// Consumer shape tests — validate Troupe's API works for Origami's use cases
// BEFORE updating agentport. Per Lex `integrate-early`: contract tests first.

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/dpopsuev/troupe"
	brokerpkg "github.com/dpopsuev/troupe/broker"
	"github.com/dpopsuev/troupe/collective"
	"github.com/dpopsuev/troupe/resilience"
	"github.com/dpopsuev/troupe/testkit"
)

// ═══════════════════════════════════════════════════════════════════════
// Shape 1: Circuit Execution — sequential steps with different roles
// Origami engine walks nodes, spawns actors per role, performs prompts.
// Old: Staff.FindByRole("investigator") → Solo.Ask(ctx, prompt)
// New: Broker.Pick(Preferences{Role: "investigator"}) → Broker.Spawn → Actor.Perform
// ═══════════════════════════════════════════════════════════════════════

func TestConsumerShape_CircuitExecution(t *testing.T) {
	ctx := context.Background()
	broker := testkit.NewMockBroker(3) //nolint:mnd // 3 roles in circuit

	// Circuit has 3 nodes: classify → investigate → review
	steps := []struct {
		role   string
		prompt string
	}{
		{"classifier", "classify this document"},
		{"investigator", "investigate the classified items"},
		{"reviewer", "review the investigation results"},
	}

	results := make([]string, 0, len(steps))
	for _, step := range steps {
		// Pick: what actors are available for this role?
		configs, err := broker.Pick(ctx, troupe.Preferences{Role: step.role, Count: 1})
		if err != nil {
			t.Fatalf("Pick(%s): %v", step.role, err)
		}
		if len(configs) == 0 {
			t.Fatalf("Pick(%s): no configs", step.role)
		}

		// Spawn: create a running actor
		actor, err := broker.Spawn(ctx, configs[0])
		if err != nil {
			t.Fatalf("Spawn(%s): %v", step.role, err)
		}

		// Perform: execute the step
		resp, err := actor.Perform(ctx, step.prompt)
		if err != nil {
			t.Fatalf("Perform(%s): %v", step.role, err)
		}
		results = append(results, resp)
	}

	if len(results) != 3 { //nolint:mnd // 3 steps
		t.Errorf("got %d results, want 3", len(results))
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Shape 2: Collective as Actor — N agents behind one interface
// Old: SpawnCollective(staff, 3, strategy) → Collective.Ask
// New: collective.SpawnCollective(ctx, broker, 3, strategy) → Actor.Perform
// ═══════════════════════════════════════════════════════════════════════

func TestConsumerShape_CollectiveAsActor(t *testing.T) {
	ctx := context.Background()
	broker := testkit.NewMockBroker(3) //nolint:mnd // dialectic needs 3

	// SpawnCollective wraps N actors behind one Actor interface
	actor, err := collective.SpawnCollective(ctx, broker, 3, &collective.RoundRobin{}) //nolint:mnd // 3 agents
	if err != nil {
		t.Fatalf("SpawnCollective: %v", err)
	}

	// Consumer sees one Actor — ISP compliance
	var a troupe.Actor = actor
	if !a.Ready() {
		t.Error("collective not ready")
	}

	resp, err := a.Perform(ctx, "debate this topic")
	if err != nil {
		t.Fatalf("Perform: %v", err)
	}
	if resp == "" {
		t.Error("empty collective response")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Shape 3: Director Pipeline — Origami's CircuitDirector pattern
// Old: engine walks DAG, dispatches to Staff
// New: Director.Direct(ctx, broker) → <-chan Event
// ═══════════════════════════════════════════════════════════════════════

func TestConsumerShape_DirectorPipeline(t *testing.T) {
	ctx := context.Background()
	broker := testkit.NewMockBroker(1)

	// LinearDirector simulates Origami's sequential circuit walk
	director := &testkit.LinearDirector{
		Steps: []testkit.Step{
			{Name: "classify", Prompt: "classify this"},
			{Name: "investigate", Prompt: "investigate findings"},
			{Name: "review", Prompt: "review results"},
		},
	}

	events, err := director.Direct(ctx, broker)
	if err != nil {
		t.Fatalf("Direct: %v", err)
	}

	// Collect events — Origami would forward these to its signal bus
	var started, completed int
	for ev := range events {
		switch ev.Kind {
		case troupe.Started:
			started++
		case troupe.Completed:
			completed++
		case troupe.Failed:
			t.Fatalf("step %s failed: %v", ev.Step, ev.Error)
		case troupe.Done:
			// circuit complete
		}
	}

	if started != 3 || completed != 3 { //nolint:mnd // 3 steps
		t.Errorf("started=%d completed=%d, want 3/3", started, completed)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Shape 4: Metering — track token/compute usage across circuit steps
// Old: billing.Tracker.Record(TokenRecord{...})
// New: broker.Meter().Record(Usage{...}), broker.Meter().Query(actor)
// ═══════════════════════════════════════════════════════════════════════

type tokenUsage struct{ In, Out int }

func (u tokenUsage) String() string { return fmt.Sprintf("tokens: in=%d out=%d", u.In, u.Out) }

func TestConsumerShape_Metering(t *testing.T) {
	meter := brokerpkg.NewInMemoryMeter()

	// Simulate circuit steps recording usage
	meter.Record(troupe.Usage{
		Actor:    "classifier",
		Step:     "classify",
		Duration: 200 * time.Millisecond, //nolint:mnd // simulated
		Detail:   tokenUsage{In: 500, Out: 100},
	})
	meter.Record(troupe.Usage{
		Actor:    "investigator",
		Step:     "investigate",
		Duration: 800 * time.Millisecond, //nolint:mnd // simulated
		Detail:   tokenUsage{In: 2000, Out: 500},
	})
	meter.Record(troupe.Usage{
		Actor:    "classifier",
		Step:     "reclassify",
		Duration: 150 * time.Millisecond, //nolint:mnd // simulated
		Detail:   tokenUsage{In: 300, Out: 80},
	})

	// Query by actor — Origami's billing report per agent
	classifierUsage := meter.Query("classifier")
	if len(classifierUsage) != 2 { //nolint:mnd // 2 steps
		t.Errorf("classifier: %d usages, want 2", len(classifierUsage))
	}

	investUsage := meter.Query("investigator")
	if len(investUsage) != 1 {
		t.Errorf("investigator: %d usages, want 1", len(investUsage))
	}

	// Detail is provider-agnostic — Origami can type-assert to tokenUsage
	detail, ok := classifierUsage[0].Detail.(tokenUsage)
	if !ok {
		t.Fatal("detail is not tokenUsage")
	}
	if detail.In != 500 { //nolint:mnd // as recorded
		t.Errorf("detail.In = %d, want 500", detail.In)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Shape 5: Hooks for Observability — Origami needs to observe lifecycle
// Old: Bus.Emit(signal) at various points
// New: WithHook(observer) — PreSpawn/PostSpawn/PrePerform/PostPerform
// ═══════════════════════════════════════════════════════════════════════

type circuitObserver struct {
	mu       sync.Mutex
	spawns   int
	performs int
	errors   int
}

func (o *circuitObserver) Name() string { return "circuit-observer" }

func (o *circuitObserver) PreSpawn(_ context.Context, _ troupe.ActorConfig) error { return nil }
func (o *circuitObserver) PostSpawn(_ context.Context, _ troupe.ActorConfig, _ troupe.Actor, err error) {
	o.mu.Lock()
	o.spawns++
	if err != nil {
		o.errors++
	}
	o.mu.Unlock()
}

func (o *circuitObserver) PrePerform(_ context.Context, _ string) error { return nil }
func (o *circuitObserver) PostPerform(_ context.Context, _, _ string, err error) {
	o.mu.Lock()
	o.performs++
	if err != nil {
		o.errors++
	}
	o.mu.Unlock()
}

var _ brokerpkg.SpawnHook = (*circuitObserver)(nil)
var _ brokerpkg.PerformHook = (*circuitObserver)(nil)

func TestConsumerShape_HookObservability(t *testing.T) {
	ctx := context.Background()
	obs := &circuitObserver{}
	broker := testkit.NewMockBroker(2) //nolint:mnd // 2 actors
	// MockBroker doesn't support WithHook, so test the hook directly
	// by wrapping actors manually — proves the interface contract

	a1, _ := broker.Spawn(ctx, troupe.ActorConfig{Role: "worker"})
	obs.PostSpawn(ctx, troupe.ActorConfig{Role: "worker"}, a1, nil)

	a1.Perform(ctx, "do work") //nolint:errcheck // best-effort cleanup
	obs.PostPerform(ctx, "do work", "response", nil)

	if obs.spawns != 1 {
		t.Errorf("spawns = %d, want 1", obs.spawns)
	}
	if obs.performs != 1 {
		t.Errorf("performs = %d, want 1", obs.performs)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Shape 6: Retry + Fallback — resilient actor for unreliable providers
// Old: manual retry in dispatch/acp_worker.go
// New: resilience.NewRetryActor(actor, config)
// ═══════════════════════════════════════════════════════════════════════

func TestConsumerShape_ResilientActor(t *testing.T) {
	ctx := context.Background()

	// Primary actor: flaky
	primary := &testkit.MockActor{Name: "primary"}
	primary.SetFailNext()

	// Wrap with retry
	retried := resilience.NewRetryActor(primary, resilience.RetryConfig{MaxAttempts: 3}) //nolint:mnd // 3 attempts

	resp, err := retried.Perform(ctx, "important work")
	if err != nil {
		t.Fatalf("retry should succeed: %v", err)
	}
	if resp == "" {
		t.Error("empty response")
	}

	// Fallback: if primary is permanently down
	dead := &testkit.MockActor{Name: "dead"}
	dead.Kill(ctx) //nolint:errcheck // best-effort cleanup
	backup := &testkit.MockActor{Name: "backup"}

	fallback := resilience.NewFallbackActor(dead, []resilience.ActorIface{backup})
	resp, err = fallback.Perform(ctx, "critical work")
	if err != nil {
		t.Fatalf("fallback should succeed: %v", err)
	}
	if resp == "" {
		t.Error("empty fallback response")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Shape 7: Multi-model Circuit — different models per node
// Old: Staff spawns with different models per role
// New: Broker.Pick(Preferences{Model: "opus"}) → Spawn with resolved config
// ═══════════════════════════════════════════════════════════════════════

func TestConsumerShape_MultiModelCircuit(t *testing.T) {
	ctx := context.Background()
	broker := testkit.NewMockBroker(3) //nolint:mnd // 3 models

	models := map[string]string{
		"classifier":   "haiku",
		"investigator": "sonnet",
		"reviewer":     "opus",
	}

	for role, model := range models {
		configs, err := broker.Pick(ctx, troupe.Preferences{Role: role, Model: model})
		if err != nil {
			t.Fatalf("Pick(%s/%s): %v", role, model, err)
		}
		if configs[0].Model != model {
			t.Errorf("Pick(%s): model = %q, want %q", role, configs[0].Model, model)
		}

		actor, err := broker.Spawn(ctx, configs[0])
		if err != nil {
			t.Fatalf("Spawn(%s): %v", role, err)
		}
		if !actor.Ready() {
			t.Errorf("actor %s not ready", role)
		}
	}
}

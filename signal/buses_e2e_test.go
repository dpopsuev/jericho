package signal_test

import (
	"testing"

	"github.com/dpopsuev/tangle/signal"
)

func TestE2E_ThreeBus_EventRouting(t *testing.T) {
	bs := signal.NewBusSet()

	// Simulate broker emitting control events.
	bs.Control.Emit(signal.Event{Kind: signal.EventDispatchRouted, Source: "broker"})
	bs.Control.Emit(signal.Event{Kind: signal.EventVetoApplied, Source: "broker"})

	// Simulate hookedActor emitting work events.
	bs.Work.Emit(signal.Event{Kind: signal.EventWorkerStart, Source: "actor"})
	bs.Work.Emit(signal.Event{Kind: signal.EventWorkerDone, Source: "actor"})
	bs.Work.Emit(signal.Event{Kind: signal.EventWorkerError, Source: "actor"})

	// Simulate warden emitting status events.
	bs.Status.Emit(signal.Event{Kind: signal.EventWorkerStarted, Source: "warden"})
	bs.Status.Emit(signal.Event{Kind: signal.EventWorkerStopped, Source: "warden"})
	bs.Status.Emit(signal.Event{Kind: signal.EventBudgetUpdate, Source: "supervisor"})

	// Verify isolation: each bus only has its own events.
	if bs.Control.Len() != 2 {
		t.Fatalf("ControlLog has %d events, want 2", bs.Control.Len())
	}
	if bs.Work.Len() != 3 {
		t.Fatalf("WorkLog has %d events, want 3", bs.Work.Len())
	}
	if bs.Status.Len() != 3 {
		t.Fatalf("StatusLog has %d events, want 3", bs.Status.Len())
	}

	// Verify correct event kinds per bus.
	controlEvents := bs.Control.Since(0)
	for _, e := range controlEvents {
		switch e.Kind {
		case signal.EventDispatchRouted, signal.EventVetoApplied, signal.EventHookExecuted:
		default:
			t.Errorf("ControlLog contains non-control event: %s", e.Kind)
		}
	}

	workEvents := bs.Work.Since(0)
	for _, e := range workEvents {
		switch e.Kind {
		case signal.EventWorkerStart, signal.EventWorkerDone, signal.EventWorkerError:
		default:
			t.Errorf("WorkLog contains non-work event: %s", e.Kind)
		}
	}

	statusEvents := bs.Status.Since(0)
	for _, e := range statusEvents {
		switch e.Kind {
		case signal.EventWorkerStarted, signal.EventWorkerStopped, signal.EventShouldStop, signal.EventBudgetUpdate, signal.EventZoneShift:
		default:
			t.Errorf("StatusLog contains non-status event: %s", e.Kind)
		}
	}
}

func TestE2E_ThreeBus_ProjectionPattern(t *testing.T) {
	bs := signal.NewBusSet()

	// Warden emits lifecycle to StatusLog AND projects to WorkLog.
	// This simulates the dual-emit pattern from TRP-TSK-148.
	bs.Status.Emit(signal.Event{Kind: signal.EventWorkerStarted, Source: "warden"})
	bs.Work.Emit(signal.Event{Kind: signal.EventWorkerStart, Source: "warden"})

	if bs.Status.Len() != 1 {
		t.Fatalf("StatusLog has %d events, want 1", bs.Status.Len())
	}
	if bs.Work.Len() != 1 {
		t.Fatalf("WorkLog has %d events, want 1", bs.Work.Len())
	}
	if bs.Control.Len() != 0 {
		t.Fatalf("ControlLog has %d events, want 0", bs.Control.Len())
	}
}

func TestE2E_ThreeBus_RefereeOnlySeesStatus(t *testing.T) {
	bs := signal.NewBusSet()

	// Simulate full pipeline.
	bs.Control.Emit(signal.Event{Kind: signal.EventDispatchRouted, Source: "broker"})
	bs.Work.Emit(signal.Event{Kind: signal.EventWorkerStart, Source: "actor"})
	bs.Work.Emit(signal.Event{Kind: signal.EventWorkerDone, Source: "actor"})
	bs.Status.Emit(signal.Event{Kind: signal.EventWorkerStarted, Source: "warden"})
	bs.Status.Emit(signal.Event{Kind: signal.EventWorkerStopped, Source: "warden"})

	// Referee subscribes to StatusLog only.
	scored := 0
	bs.Status.OnEmit(func(_ signal.Event) { scored++ })

	// More events arrive.
	bs.Control.Emit(signal.Event{Kind: signal.EventVetoApplied, Source: "broker"})
	bs.Work.Emit(signal.Event{Kind: signal.EventWorkerError, Source: "actor"})
	bs.Status.Emit(signal.Event{Kind: signal.EventBudgetUpdate, Source: "supervisor"})

	if scored != 1 {
		t.Fatalf("Referee scored %d events, want 1 (only StatusLog)", scored)
	}
}

package signal_test

import (
	"context"
	"testing"

	"github.com/dpopsuev/tangle/signal"
)

func TestOTelLog_ImplementsEventLog(t *testing.T) {
	log, err := signal.NewOTelLog(context.Background(), "test")
	if err != nil {
		t.Fatalf("NewOTelLog: %v", err)
	}

	idx := log.Emit(signal.Event{
		Kind:   signal.EventDispatchRouted,
		Source: "broker",
	})
	if idx != 0 {
		t.Errorf("first emit index = %d, want 0", idx)
	}

	if log.Len() != 1 {
		t.Errorf("Len() = %d, want 1", log.Len())
	}

	events := log.Since(0)
	if len(events) != 1 {
		t.Fatalf("Since(0) = %d events, want 1", len(events))
	}
	if events[0].Kind != signal.EventDispatchRouted {
		t.Errorf("kind = %q, want %q", events[0].Kind, signal.EventDispatchRouted)
	}
}

func TestOTelLog_OnEmitCallbackFires(t *testing.T) {
	log, err := signal.NewOTelLog(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}

	var received signal.Event
	log.OnEmit(func(e signal.Event) {
		received = e
	})

	log.Emit(signal.Event{Kind: signal.EventWorkerStart, Source: "w1"})

	if received.Kind != signal.EventWorkerStart {
		t.Errorf("callback got kind = %q, want %q", received.Kind, signal.EventWorkerStart)
	}
}

func TestOTelBusSet_AllBusesWork(t *testing.T) {
	buses, err := signal.NewOTelBusSet(context.Background())
	if err != nil {
		t.Fatalf("NewOTelBusSet: %v", err)
	}

	buses.Control.Emit(signal.Event{Kind: signal.EventDispatchRouted, Source: "broker"})
	buses.Work.Emit(signal.Event{Kind: signal.EventWorkerStart, Source: "w1"})
	buses.Work.Emit(signal.Event{Kind: signal.EventWorkerError, Source: "w1"})
	buses.Status.Emit(signal.Event{Kind: signal.EventWorkerStarted, Source: "warden"})

	if buses.Control.Len() != 1 {
		t.Errorf("control = %d, want 1", buses.Control.Len())
	}
	if buses.Work.Len() != 2 {
		t.Errorf("work = %d, want 2", buses.Work.Len())
	}
	if buses.Status.Len() != 1 {
		t.Errorf("status = %d, want 1", buses.Status.Len())
	}
}

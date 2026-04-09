package signal_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dpopsuev/battery/event"
	"github.com/dpopsuev/battery/testkit"
	"github.com/dpopsuev/troupe/signal"
)

// Run Battery's canonical contract test against MemBus adapter.
func TestEventLogContract_MemBus(t *testing.T) {
	bus := signal.NewMemBus()
	log := signal.NewBusEventLog(bus)
	testkit.RunEventLogContract(t, log)
}

// Run Battery's canonical contract test against DurableBus adapter.
func TestEventLogContract_DurableBus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	bus, err := signal.NewDurableBus(path)
	if err != nil {
		t.Fatalf("NewDurableBus: %v", err)
	}
	defer bus.Close()

	log := signal.NewBusEventLog(bus)
	testkit.RunEventLogContract(t, log)

	// Verify persistence — file should have content
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("durable log file is empty after emits")
	}
}

// Test round-trip fidelity: Event → Signal → Event preserves fields.
func TestBusEventLog_RoundTrip(t *testing.T) {
	bus := signal.NewMemBus()
	log := signal.NewBusEventLog(bus)

	log.Emit(event.Event{
		ID:       "evt-1",
		ParentID: "parent-1",
		Source:   "agent",
		Kind:     "tool.call",
		Meta:     map[string]string{"tool": "Write", "path": "/tmp/x.go"},
	})

	events := log.Since(0)
	if len(events) != 1 {
		t.Fatalf("events = %d", len(events))
	}

	e := events[0]
	if e.ID != "evt-1" {
		t.Fatalf("ID = %q", e.ID)
	}
	if e.ParentID != "parent-1" {
		t.Fatalf("ParentID = %q", e.ParentID)
	}
	if e.Source != "agent" {
		t.Fatalf("Source = %q", e.Source)
	}
	if e.Kind != "tool.call" {
		t.Fatalf("Kind = %q", e.Kind)
	}
	if e.Meta["tool"] != "Write" {
		t.Fatalf("Meta[tool] = %q", e.Meta["tool"])
	}
	if e.Timestamp.IsZero() {
		t.Fatal("Timestamp should be auto-set")
	}
}

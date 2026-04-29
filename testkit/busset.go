package testkit

import (
	"testing"

	"github.com/dpopsuev/tangle/signal"
)

// NewTestBusSet creates a BusSet backed by StubEventLogs for testing.
func NewTestBusSet() signal.BusSet {
	return signal.BusSet{
		Control: signal.ControlLog{EventLog: NewStubEventLog()},
		Work:    signal.WorkLog{EventLog: NewStubEventLog()},
		Status:  signal.StatusLog{EventLog: NewStubEventLog()},
	}
}

// AssertControlEvent asserts that the ControlLog contains an event with the given kind.
func AssertControlEvent(t *testing.T, bs signal.BusSet, kind string) {
	t.Helper()
	assertBusContains(t, "ControlLog", bs.Control, kind)
}

// AssertWorkEvent asserts that the WorkLog contains an event with the given kind.
func AssertWorkEvent(t *testing.T, bs signal.BusSet, kind string) {
	t.Helper()
	assertBusContains(t, "WorkLog", bs.Work, kind)
}

// AssertStatusEvent asserts that the StatusLog contains an event with the given kind.
func AssertStatusEvent(t *testing.T, bs signal.BusSet, kind string) {
	t.Helper()
	assertBusContains(t, "StatusLog", bs.Status, kind)
}

// AssertBusEmpty asserts that a specific bus has no events.
func AssertBusEmpty(t *testing.T, name string, log signal.EventLog) {
	t.Helper()
	if log.Len() != 0 {
		t.Errorf("%s has %d events, want 0", name, log.Len())
	}
}

func assertBusContains(t *testing.T, name string, log signal.EventLog, kind string) {
	t.Helper()
	events := log.Since(0)
	for _, e := range events {
		if e.Kind == kind {
			return
		}
	}
	t.Errorf("%s does not contain event kind %q (has %d events)", name, kind, len(events))
}

package testkit_test

import (
	"testing"

	"github.com/dpopsuev/tangle/signal"
	"github.com/dpopsuev/tangle/testkit"
)

func TestNewTestBusSet_Independent(t *testing.T) {
	bs := testkit.NewTestBusSet()
	bs.Control.Emit(signal.Event{Kind: "dispatch_routed"})
	bs.Work.Emit(signal.Event{Kind: "start"})
	bs.Status.Emit(signal.Event{Kind: "worker_started"})

	testkit.AssertControlEvent(t, bs, "dispatch_routed")
	testkit.AssertWorkEvent(t, bs, "start")
	testkit.AssertStatusEvent(t, bs, "worker_started")

	testkit.AssertBusEmpty(t, "ControlLog after clear check", testkit.NewTestBusSet().Control)
}

func TestAssertBusEmpty(t *testing.T) {
	bs := testkit.NewTestBusSet()
	testkit.AssertBusEmpty(t, "ControlLog", bs.Control)
	testkit.AssertBusEmpty(t, "WorkLog", bs.Work)
	testkit.AssertBusEmpty(t, "StatusLog", bs.Status)
}

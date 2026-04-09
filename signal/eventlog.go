// eventlog.go — BusEventLog adapts signal.Bus as battery/event.EventLog.
//
// Strangler Fig: Bus interface kept as-is. BusEventLog wraps any Bus
// to satisfy EventLog. Consumers migrate to EventLog at their own pace.
package signal

import (
	"time"

	"github.com/dpopsuev/battery/event"
)

// BusEventLog wraps a signal.Bus as an event.EventLog.
// Converts between Signal and Event transparently.
type BusEventLog struct {
	bus Bus
}

var _ event.EventLog = (*BusEventLog)(nil)

// NewBusEventLog wraps any Bus (MemBus, DurableBus) as an EventLog.
func NewBusEventLog(bus Bus) *BusEventLog {
	return &BusEventLog{bus: bus}
}

// Emit converts an Event to a Signal and appends to the bus.
func (l *BusEventLog) Emit(e event.Event) int {
	s := eventToSignal(e)
	return l.bus.Emit(&s)
}

// Since returns events from index onward, converting Signals to Events.
func (l *BusEventLog) Since(index int) []event.Event {
	signals := l.bus.Since(index)
	if signals == nil {
		return nil
	}
	events := make([]event.Event, len(signals))
	for i := range signals {
		events[i] = signalToEvent(signals[i])
	}
	return events
}

// Len returns the total number of events.
func (l *BusEventLog) Len() int {
	return l.bus.Len()
}

// OnEmit registers a callback, converting Signal to Event.
func (l *BusEventLog) OnEmit(fn func(event.Event)) {
	l.bus.OnEmit(func(s Signal) {
		fn(signalToEvent(s))
	})
}

// eventToSignal converts a battery Event to a troupe Signal.
func eventToSignal(e event.Event) Signal {
	ts := e.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	return Signal{
		Timestamp: ts.Format(time.RFC3339Nano),
		Event:     e.Kind,
		Agent:     e.Source,
		CaseID:    e.ID,
		Step:      e.ParentID,
		Meta:      e.Meta,
	}
}

// signalToEvent converts a troupe Signal to a battery Event.
func signalToEvent(s Signal) event.Event {
	ts, _ := time.Parse(time.RFC3339Nano, s.Timestamp)
	return event.Event{
		ID:        s.CaseID,
		ParentID:  s.Step,
		Timestamp: ts,
		Source:    s.Agent,
		Kind:     s.Event,
		Meta:     s.Meta,
	}
}

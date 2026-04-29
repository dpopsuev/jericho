package signal

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/dpopsuev/tangle/signal"

var _ EventLog = (*OTelLog)(nil)

// OTelLog is an EventLog that pipes every event through OpenTelemetry.
// Each Emit creates a span + increments metrics. Also stores events
// in memory (strangler fig — MemLog underneath, OTel on top).
type OTelLog struct {
	mu     sync.Mutex
	inner  MemLog
	bus    string
	tracer trace.Tracer
	ctx    context.Context

	eventCount metric.Int64Counter
	errorCount metric.Int64Counter
}

// NewOTelLog creates an EventLog that pipes to OTel.
// busName identifies this log (control, work, status) in span names.
func NewOTelLog(ctx context.Context, busName string) (*OTelLog, error) {
	tracer := otel.Tracer(instrumentationName)
	meter := otel.Meter(instrumentationName)

	eventCount, err := meter.Int64Counter("troupe.events.total",
		metric.WithDescription("Total signal events emitted"),
	)
	if err != nil {
		return nil, err
	}

	errorCount, err := meter.Int64Counter("troupe.errors.total",
		metric.WithDescription("Total error events"),
	)
	if err != nil {
		return nil, err
	}

	return &OTelLog{
		bus:        busName,
		tracer:     tracer,
		ctx:        ctx,
		eventCount: eventCount,
		errorCount: errorCount,
	}, nil
}

// Emit appends the event AND creates an OTel span + metrics.
func (l *OTelLog) Emit(e Event) int {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}

	_, span := l.tracer.Start(l.ctx, "troupe."+l.bus+"."+e.Kind,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithTimestamp(e.Timestamp),
	)
	span.SetAttributes(
		attribute.String("troupe.bus", l.bus),
		attribute.String("troupe.event.kind", e.Kind),
		attribute.String("troupe.event.source", e.Source),
	)
	if e.ID != "" {
		span.SetAttributes(attribute.String("troupe.event.id", e.ID))
	}
	if e.TraceID != "" {
		span.SetAttributes(attribute.String("troupe.trace_id", e.TraceID))
	}
	span.End()

	attrs := metric.WithAttributes(
		attribute.String("bus", l.bus),
		attribute.String("kind", e.Kind),
	)
	l.eventCount.Add(l.ctx, 1, attrs)

	if e.Kind == EventWorkerError {
		l.errorCount.Add(l.ctx, 1,
			metric.WithAttributes(attribute.String("source", e.Source)),
		)
	}

	return l.inner.Emit(e)
}

func (l *OTelLog) Since(index int) []Event { return l.inner.Since(index) }
func (l *OTelLog) Len() int                { return l.inner.Len() }
func (l *OTelLog) OnEmit(fn func(Event))   { l.inner.OnEmit(fn) }

// ByTraceID delegates to the inner MemLog.
func (l *OTelLog) ByTraceID(traceID string) []Event { return l.inner.ByTraceID(traceID) }

// NewOTelBusSet creates a BusSet where every bus pipes through OTel.
func NewOTelBusSet(ctx context.Context) (*BusSet, error) {
	control, err := NewOTelLog(ctx, "control")
	if err != nil {
		return nil, err
	}
	work, err := NewOTelLog(ctx, "work")
	if err != nil {
		return nil, err
	}
	status, err := NewOTelLog(ctx, "status")
	if err != nil {
		return nil, err
	}
	return &BusSet{
		Control: ControlLog{control},
		Work:    WorkLog{work},
		Status:  StatusLog{status},
	}, nil
}

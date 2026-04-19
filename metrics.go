package troupe

// MetricsExporter is the daemon metrics interface. Consumers plug
// Prometheus, OTEL, or any backend. Troupe emits; backend exports.
type MetricsExporter interface {
	CounterInc(name string, labels map[string]string)
	HistogramObserve(name string, value float64, labels map[string]string)
	GaugeSet(name string, value float64, labels map[string]string)
}

// NoopExporter discards all metrics. Default when no exporter is configured.
type NoopExporter struct{}

func (NoopExporter) CounterInc(_ string, _ map[string]string)                  {}
func (NoopExporter) HistogramObserve(_ string, _ float64, _ map[string]string) {}
func (NoopExporter) GaugeSet(_ string, _ float64, _ map[string]string)         {}

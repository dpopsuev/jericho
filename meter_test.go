package troupe_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/dpopsuev/troupe"
)

type tokenDetail struct{ In, Out int }

func (d tokenDetail) String() string { return fmt.Sprintf("tokens: in=%d out=%d", d.In, d.Out) }

type computeDetail struct{ GPUSec float64 }

func (d computeDetail) String() string { return fmt.Sprintf("gpu: %.1fs", d.GPUSec) }

func TestInMemoryMeter_RecordAndQuery(t *testing.T) {
	m := troupe.NewInMemoryMeter()
	m.Record(troupe.Usage{Actor: "a1", Step: "classify", Duration: time.Second})
	m.Record(troupe.Usage{Actor: "a2", Step: "review", Duration: 2 * time.Second})
	m.Record(troupe.Usage{Actor: "a1", Step: "summarize", Duration: 500 * time.Millisecond})

	a1 := m.Query("a1")
	if len(a1) != 2 {
		t.Fatalf("a1: got %d usages, want 2", len(a1))
	}
	if a1[0].Duration != time.Second {
		t.Errorf("a1[0] duration = %v, want 1s", a1[0].Duration)
	}

	a2 := m.Query("a2")
	if len(a2) != 1 {
		t.Fatalf("a2: got %d usages, want 1", len(a2))
	}
}

func TestInMemoryMeter_QueryEmpty(t *testing.T) {
	m := troupe.NewInMemoryMeter()
	result := m.Query("nonexistent")
	if len(result) != 0 {
		t.Errorf("expected empty, got %d", len(result))
	}
}

func TestMeter_ProviderAgnostic(t *testing.T) {
	m := troupe.NewInMemoryMeter()
	m.Record(troupe.Usage{Actor: "cloud", Detail: tokenDetail{In: 100, Out: 50}})
	m.Record(troupe.Usage{Actor: "onprem", Detail: computeDetail{GPUSec: 3.5}})

	cloud := m.Query("cloud")
	if cloud[0].Detail.String() != "tokens: in=100 out=50" {
		t.Errorf("cloud detail = %q", cloud[0].Detail.String())
	}

	onprem := m.Query("onprem")
	if onprem[0].Detail.String() != "gpu: 3.5s" {
		t.Errorf("onprem detail = %q", onprem[0].Detail.String())
	}
}

var _ troupe.Meter = (*troupe.InMemoryMeter)(nil)

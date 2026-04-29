package troupe_test

import (
	"testing"

	"github.com/dpopsuev/tangle"
)

func TestIntThreshold_BelowLimit(t *testing.T) {
	counter := 0
	th := troupe.IntThreshold(func() int { return counter }, 3)
	if th() {
		t.Fatal("should not fire when counter=0, limit=3")
	}
	counter = 2
	if th() {
		t.Fatal("should not fire when counter=2, limit=3")
	}
}

func TestIntThreshold_AtLimit(t *testing.T) {
	counter := 3
	th := troupe.IntThreshold(func() int { return counter }, 3)
	if !th() {
		t.Fatal("should fire when counter=3, limit=3")
	}
}

func TestIntThreshold_AboveLimit(t *testing.T) {
	counter := 5
	th := troupe.IntThreshold(func() int { return counter }, 3)
	if !th() {
		t.Fatal("should fire when counter=5, limit=3")
	}
}

func TestIntThreshold_ResetBelowLimit(t *testing.T) {
	counter := 5
	th := troupe.IntThreshold(func() int { return counter }, 3)
	if !th() {
		t.Fatal("should fire at 5")
	}
	counter = 1
	if th() {
		t.Fatal("should not fire after reset to 1")
	}
}

func TestFloatThreshold(t *testing.T) {
	usage := 0.0
	th := troupe.FloatThreshold(func() float64 { return usage }, 0.8)
	if th() {
		t.Fatal("should not fire at 0.0")
	}
	usage = 0.8
	if !th() {
		t.Fatal("should fire at 0.8")
	}
}

func TestDurationThreshold(t *testing.T) {
	elapsed := int64(0)
	th := troupe.DurationThreshold(func() int64 { return elapsed }, 1000)
	if th() {
		t.Fatal("should not fire at 0")
	}
	elapsed = 1000
	if !th() {
		t.Fatal("should fire at 1000")
	}
}

func TestThreshold_AsPlainFunc(t *testing.T) {
	counter := 0
	th := troupe.IntThreshold(func() int { return counter }, 3)

	var fn func() bool = th
	counter = 3
	if !fn() {
		t.Fatal("Threshold should be usable as func() bool")
	}
}

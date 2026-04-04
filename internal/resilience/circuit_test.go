package resilience

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedOnSuccess(t *testing.T) {
	cb := NewCircuitBreaker(CircuitConfig{Threshold: 3})

	err := cb.Call(func() error { return nil })
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if cb.State() != CircuitClosed {
		t.Fatalf("state = %v, want closed", cb.State())
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	var transitions []CircuitState
	cb := NewCircuitBreaker(CircuitConfig{
		Threshold: 3,
		Cooldown:  1 * time.Hour,
		OnChange:  func(_, to CircuitState) { transitions = append(transitions, to) },
	})

	fail := errors.New("fail")
	for range 3 {
		cb.Call(func() error { return fail })
	}

	if cb.State() != CircuitOpen {
		t.Fatalf("state = %v, want open", cb.State())
	}

	// Should reject immediately.
	err := cb.Call(func() error { return nil })
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("err = %v, want ErrCircuitOpen", err)
	}

	if len(transitions) != 1 || transitions[0] != CircuitOpen {
		t.Fatalf("transitions = %v", transitions)
	}
}

func TestCircuitBreaker_HalfOpenProbe(t *testing.T) {
	cb := NewCircuitBreaker(CircuitConfig{
		Threshold: 1,
		Cooldown:  1 * time.Millisecond,
	})

	// Trip the circuit.
	cb.Call(func() error { return errors.New("fail") })
	if cb.State() != CircuitOpen {
		t.Fatalf("state = %v, want open", cb.State())
	}

	// Wait for cooldown.
	time.Sleep(5 * time.Millisecond)

	// Next call should be half-open probe — success closes.
	err := cb.Call(func() error { return nil })
	if err != nil {
		t.Fatalf("half-open probe err = %v", err)
	}
	if cb.State() != CircuitClosed {
		t.Fatalf("state = %v, want closed after successful probe", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenFailReopens(t *testing.T) {
	cb := NewCircuitBreaker(CircuitConfig{
		Threshold: 1,
		Cooldown:  1 * time.Millisecond,
	})

	cb.Call(func() error { return errors.New("fail") })
	time.Sleep(5 * time.Millisecond)

	// Half-open probe fails → re-opens.
	cb.Call(func() error { return errors.New("still failing") })
	if cb.State() != CircuitOpen {
		t.Fatalf("state = %v, want open after failed probe", cb.State())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(CircuitConfig{Threshold: 1})
	cb.Call(func() error { return errors.New("fail") })

	cb.Reset()
	if cb.State() != CircuitClosed {
		t.Fatalf("state = %v after Reset", cb.State())
	}

	// Should accept calls again.
	err := cb.Call(func() error { return nil })
	if err != nil {
		t.Fatalf("err = %v after Reset", err)
	}
}

func TestCircuitBreaker_SuccessResetsFailureCount(t *testing.T) {
	cb := NewCircuitBreaker(CircuitConfig{Threshold: 3})

	// 2 failures, then a success.
	cb.Call(func() error { return errors.New("1") })
	cb.Call(func() error { return errors.New("2") })
	cb.Call(func() error { return nil }) // resets counter

	// 2 more failures should NOT trip (counter was reset).
	cb.Call(func() error { return errors.New("3") })
	cb.Call(func() error { return errors.New("4") })

	if cb.State() != CircuitClosed {
		t.Fatalf("state = %v, want closed (counter should have reset)", cb.State())
	}
}

func TestCircuitBreaker_DefaultConfig(t *testing.T) {
	cb := NewCircuitBreaker(CircuitConfig{})
	if cb.threshold != 5 {
		t.Fatalf("threshold = %d, want 5", cb.threshold)
	}
	if cb.cooldown != 30*time.Second {
		t.Fatalf("cooldown = %v, want 30s", cb.cooldown)
	}
}

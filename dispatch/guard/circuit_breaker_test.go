package guard

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	bd "github.com/dpopsuev/jericho/dispatch"
)

type failingDispatcher struct {
	failN int
	calls int
	mu    sync.Mutex
}

func (d *failingDispatcher) Dispatch(_ context.Context, _ bd.Context) ([]byte, error) { //nolint:gocritic
	d.mu.Lock()
	defer d.mu.Unlock()
	d.calls++
	if d.calls <= d.failN {
		return nil, errors.New("provider error")
	}
	return []byte(`{"ok":true}`), nil
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	inner := &failingDispatcher{failN: 100}
	var transitions []CircuitState
	cb := NewCircuitBreakerDispatcher(inner, CircuitBreakerConfig{
		Threshold: 3,
		Cooldown:  time.Hour,
		OnChange: func(_, to CircuitState) {
			transitions = append(transitions, to)
		},
	})

	ctx := bd.Context{Step: "test"}
	for i := 0; i < 3; i++ {
		_, _ = cb.Dispatch(context.Background(), ctx)
	}

	if cb.State() != CircuitOpen {
		t.Errorf("expected CircuitOpen after 3 failures, got %s", cb.State())
	}

	_, err := cb.Dispatch(context.Background(), ctx)
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}

	if len(transitions) != 1 || transitions[0] != CircuitOpen {
		t.Errorf("expected [open] transition, got %v", transitions)
	}
}

func TestCircuitBreaker_HalfOpenProbeSuccess(t *testing.T) {
	inner := &failingDispatcher{failN: 3}
	cb := NewCircuitBreakerDispatcher(inner, CircuitBreakerConfig{
		Threshold: 3,
		Cooldown:  1 * time.Millisecond,
	})

	ctx := bd.Context{Step: "test"}
	for i := 0; i < 3; i++ {
		_, _ = cb.Dispatch(context.Background(), ctx)
	}

	if cb.State() != CircuitOpen {
		t.Fatalf("expected CircuitOpen, got %s", cb.State())
	}

	time.Sleep(5 * time.Millisecond)

	data, err := cb.Dispatch(context.Background(), ctx)
	if err != nil {
		t.Fatalf("half-open probe should succeed: %v", err)
	}
	if string(data) != `{"ok":true}` {
		t.Errorf("unexpected data: %s", data)
	}
	if cb.State() != CircuitClosed {
		t.Errorf("expected CircuitClosed after successful probe, got %s", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenProbeFailure(t *testing.T) {
	inner := &failingDispatcher{failN: 100}
	cb := NewCircuitBreakerDispatcher(inner, CircuitBreakerConfig{
		Threshold: 2,
		Cooldown:  1 * time.Millisecond,
	})

	ctx := bd.Context{Step: "test"}
	for i := 0; i < 2; i++ {
		_, _ = cb.Dispatch(context.Background(), ctx)
	}

	time.Sleep(5 * time.Millisecond)

	_, err := cb.Dispatch(context.Background(), ctx)
	if err == nil {
		t.Fatal("expected failure on half-open probe")
	}
	if cb.State() != CircuitOpen {
		t.Errorf("expected CircuitOpen after failed probe, got %s", cb.State())
	}
}

func TestCircuitBreaker_ClosedOnSuccess(t *testing.T) {
	inner := &failingDispatcher{failN: 0}
	cb := NewCircuitBreakerDispatcher(inner, CircuitBreakerConfig{Threshold: 5})

	ctx := bd.Context{Step: "test"}
	data, err := cb.Dispatch(context.Background(), ctx)
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if string(data) != `{"ok":true}` {
		t.Errorf("unexpected data: %s", data)
	}
	if cb.State() != CircuitClosed {
		t.Errorf("expected CircuitClosed, got %s", cb.State())
	}
}

func TestCircuitBreaker_Unwrapper(t *testing.T) {
	inner := &failingDispatcher{}
	cb := NewCircuitBreakerDispatcher(inner, CircuitBreakerConfig{})
	if cb.Inner() != bd.Dispatcher(inner) {
		t.Error("Inner() should return the wrapped dispatcher")
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	inner := &failingDispatcher{failN: 100}
	cb := NewCircuitBreakerDispatcher(inner, CircuitBreakerConfig{
		Threshold: 3,
		Cooldown:  time.Hour,
	})

	ctx := bd.Context{Step: "test"}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = cb.Dispatch(context.Background(), ctx)
		}()
	}
	wg.Wait()

	state := cb.State()
	if state != CircuitOpen {
		t.Errorf("expected CircuitOpen after concurrent failures, got %s", state)
	}
}

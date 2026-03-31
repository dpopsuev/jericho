package guard

import (
	"context"

	bd "github.com/dpopsuev/jericho/dispatch"
	"github.com/dpopsuev/jericho/resilience"
)

// CircuitState is an alias for resilience.CircuitState.
type CircuitState = resilience.CircuitState

// Circuit state constants — re-exported from resilience.
const (
	CircuitClosed   = resilience.CircuitClosed
	CircuitOpen     = resilience.CircuitOpen
	CircuitHalfOpen = resilience.CircuitHalfOpen
)

// ErrCircuitOpen is re-exported from resilience.
var ErrCircuitOpen = resilience.ErrCircuitOpen

// CircuitBreakerHook is called on circuit state transitions.
type CircuitBreakerHook = func(from, to CircuitState)

// CircuitBreakerConfig configures a CircuitBreakerDispatcher.
type CircuitBreakerConfig = resilience.CircuitConfig

// CircuitBreakerDispatcher wraps a bd.Dispatcher with circuit breaker protection.
// Delegates to resilience.CircuitBreaker for the state machine.
type CircuitBreakerDispatcher struct {
	inner   bd.Dispatcher
	breaker *resilience.CircuitBreaker
}

// NewCircuitBreakerDispatcher wraps inner with circuit breaker protection.
func NewCircuitBreakerDispatcher(inner bd.Dispatcher, cfg CircuitBreakerConfig) *CircuitBreakerDispatcher {
	return &CircuitBreakerDispatcher{
		inner:   inner,
		breaker: resilience.NewCircuitBreaker(cfg),
	}
}

// State returns the current circuit state.
func (d *CircuitBreakerDispatcher) State() CircuitState {
	return d.breaker.State()
}

// Dispatch delegates to the inner dispatcher if the circuit allows it.
func (d *CircuitBreakerDispatcher) Dispatch(ctx context.Context, dc bd.Context) ([]byte, error) {
	var data []byte
	err := d.breaker.Call(func() error {
		var callErr error
		data, callErr = d.inner.Dispatch(ctx, dc)
		return callErr
	})
	return data, err
}

// Inner returns the wrapped dispatcher.
func (d *CircuitBreakerDispatcher) Inner() bd.Dispatcher { return d.inner }

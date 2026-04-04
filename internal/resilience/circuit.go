package resilience

import (
	"errors"
	"sync"
	"time"
)

// CircuitState represents the current state of a circuit breaker.
type CircuitState int

const (
	CircuitClosed   CircuitState = iota // normal operation
	CircuitOpen                         // failures exceeded threshold, rejecting
	CircuitHalfOpen                     // cooldown elapsed, probing with one call
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// ErrCircuitOpen is returned when the circuit breaker rejects a call.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// CircuitConfig configures a CircuitBreaker.
type CircuitConfig struct {
	Threshold int           // consecutive failures before opening (default 5)
	Cooldown  time.Duration // wait before half-open probe (default 30s)
	OnChange  func(from, to CircuitState)
}

// CircuitBreaker implements the circuit breaker pattern.
// After Threshold consecutive failures the circuit opens, rejecting all calls
// with ErrCircuitOpen. After Cooldown elapses, one probe call is allowed
// (half-open): success closes the circuit, failure re-opens it.
type CircuitBreaker struct {
	threshold int
	cooldown  time.Duration
	onChange  func(from, to CircuitState)

	mu       sync.Mutex
	state    CircuitState
	failures int
	openedAt time.Time
}

// NewCircuitBreaker creates a circuit breaker with the given config.
func NewCircuitBreaker(cfg CircuitConfig) *CircuitBreaker {
	threshold := cfg.Threshold
	if threshold <= 0 {
		threshold = 5
	}
	cooldown := cfg.Cooldown
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	return &CircuitBreaker{
		threshold: threshold,
		cooldown:  cooldown,
		onChange:  cfg.OnChange,
		state:     CircuitClosed,
	}
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Call executes fn if the circuit allows it. Returns ErrCircuitOpen if
// the circuit is open and cooldown hasn't elapsed.
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	if cb.state == CircuitOpen {
		if time.Since(cb.openedAt) >= cb.cooldown {
			cb.transition(CircuitHalfOpen)
		} else {
			cb.mu.Unlock()
			return ErrCircuitOpen
		}
	}
	cb.mu.Unlock()

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		if cb.state == CircuitHalfOpen {
			cb.transition(CircuitOpen)
			cb.openedAt = time.Now()
		} else if cb.failures >= cb.threshold {
			cb.transition(CircuitOpen)
			cb.openedAt = time.Now()
		}
		return err
	}

	if cb.state == CircuitHalfOpen || cb.failures > 0 {
		cb.transition(CircuitClosed)
	}
	cb.failures = 0
	return nil
}

// Reset forces the circuit back to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.transition(CircuitClosed)
	cb.failures = 0
}

func (cb *CircuitBreaker) transition(to CircuitState) {
	from := cb.state
	if from == to {
		return
	}
	cb.state = to
	if cb.onChange != nil {
		cb.onChange(from, to)
	}
}

package testkit

import (
	"context"
	"fmt"
	"sync"
)

// MockActor echoes prompts as responses. Thread-safe for concurrent use.
type MockActor struct {
	Name string

	mu       sync.Mutex
	prompts  []string
	ready    bool
	killed   bool
	failNext bool
}

func (a *MockActor) Perform(_ context.Context, prompt string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.killed {
		return "", fmt.Errorf("actor %s: killed", a.Name)
	}
	if a.failNext {
		a.failNext = false
		return "", fmt.Errorf("actor %s: simulated failure", a.Name)
	}

	a.prompts = append(a.prompts, prompt)
	return fmt.Sprintf("[%s] %s", a.Name, prompt), nil
}

func (a *MockActor) Ready() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return !a.killed
}

func (a *MockActor) Kill(_ context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.killed = true
	return nil
}

// Prompts returns the prompts this actor received.
func (a *MockActor) Prompts() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	cp := make([]string, len(a.prompts))
	copy(cp, a.prompts)
	return cp
}

// SetFailNext makes the next Perform call fail.
func (a *MockActor) SetFailNext() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.failNext = true
}

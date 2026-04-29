package testkit

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/dpopsuev/tangle/billing"
)

// EchoAgent returns the prompt as the response unchanged.
type EchoAgent struct {
	calls  atomic.Int64
	killed atomic.Bool
}

func (a *EchoAgent) Perform(_ context.Context, prompt string) (string, error) {
	if a.killed.Load() {
		return "", fmt.Errorf("echo agent: killed")
	}
	a.calls.Add(1)
	return prompt, nil
}

func (a *EchoAgent) Ready() bool                  { return !a.killed.Load() }
func (a *EchoAgent) Kill(_ context.Context) error { a.killed.Store(true); return nil }
func (a *EchoAgent) Calls() int64                 { return a.calls.Load() }

// SlowAgent sleeps for Delay before responding. Context cancellation
// preempts the sleep, making it useful for timeout and liveness tests.
type SlowAgent struct {
	Delay  time.Duration
	calls  atomic.Int64
	killed atomic.Bool
}

func (a *SlowAgent) Perform(ctx context.Context, prompt string) (string, error) {
	if a.killed.Load() {
		return "", fmt.Errorf("slow agent: killed")
	}
	select {
	case <-time.After(a.Delay):
		a.calls.Add(1)
		return prompt, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (a *SlowAgent) Ready() bool                  { return !a.killed.Load() }
func (a *SlowAgent) Kill(_ context.Context) error { a.killed.Store(true); return nil }
func (a *SlowAgent) Calls() int64                 { return a.calls.Load() }

// FailAgent returns an error every FailEvery calls (1 = always fail, 0 = never).
type FailAgent struct {
	FailEvery int
	calls     atomic.Int64
	killed    atomic.Bool
}

func (a *FailAgent) Perform(_ context.Context, prompt string) (string, error) {
	if a.killed.Load() {
		return "", fmt.Errorf("fail agent: killed")
	}
	n := a.calls.Add(1)
	if a.FailEvery > 0 && n%int64(a.FailEvery) == 0 {
		return "", fmt.Errorf("fail agent: simulated failure (call %d)", n)
	}
	return prompt, nil
}

func (a *FailAgent) Ready() bool                  { return !a.killed.Load() }
func (a *FailAgent) Kill(_ context.Context) error { a.killed.Store(true); return nil }
func (a *FailAgent) Calls() int64                 { return a.calls.Load() }

// BudgetAgent records TokensPerCall tokens to a billing.Tracker on each
// Perform call. Pair with billing.BudgetEnforcer to test spending limits.
type BudgetAgent struct {
	TokensPerCall int
	Tracker       billing.Tracker
	AgentID       string
	calls         atomic.Int64
	killed        atomic.Bool
}

func (a *BudgetAgent) Perform(_ context.Context, prompt string) (string, error) {
	if a.killed.Load() {
		return "", fmt.Errorf("budget agent: killed")
	}
	n := a.calls.Add(1)
	a.Tracker.Record(&billing.TokenRecord{
		Node:           a.AgentID,
		Step:           fmt.Sprintf("call-%d", n),
		PromptTokens:   a.TokensPerCall,
		ArtifactTokens: a.TokensPerCall,
		Timestamp:      time.Now(),
	})
	return prompt, nil
}

func (a *BudgetAgent) Ready() bool                  { return !a.killed.Load() }
func (a *BudgetAgent) Kill(_ context.Context) error { a.killed.Store(true); return nil }
func (a *BudgetAgent) Calls() int64                 { return a.calls.Load() }

package guard

import (
	"context"
	"os"
	"time"

	bd "github.com/dpopsuev/jericho/dispatch"

	"github.com/dpopsuev/jericho/billing"
)

// Hook is called after each dispatch with timing and error info.
type Hook func(provider, step string, duration time.Duration, err error)

// TokenTrackingDispatcher wraps any bd.Dispatcher and records token usage
// for each dispatch call. Optional Hooks receive timing/error data
// for bridging with metrics systems.
type TokenTrackingDispatcher struct {
	inner         bd.Dispatcher
	tracker       billing.Tracker
	provider      string
	dispatchHooks []Hook
}

// NewTokenTrackingDispatcher wraps a dispatcher with token tracking.
func NewTokenTrackingDispatcher(inner bd.Dispatcher, tracker billing.Tracker) *TokenTrackingDispatcher {
	return &TokenTrackingDispatcher{inner: inner, tracker: tracker}
}

// SetProvider sets the provider label used for dispatch hooks.
func (d *TokenTrackingDispatcher) SetProvider(name string) {
	d.provider = name
}

// OnDispatch registers a hook invoked after each Dispatch call.
func (d *TokenTrackingDispatcher) OnDispatch(hook Hook) {
	d.dispatchHooks = append(d.dispatchHooks, hook)
}

// Dispatch delegates to the inner dispatcher while recording token metrics.
func (d *TokenTrackingDispatcher) Dispatch(ctx context.Context, dc bd.Context) ([]byte, error) { //nolint:gocritic // value receiver for API compat
	promptBytes := 0
	if info, err := os.Stat(dc.PromptPath); err == nil {
		promptBytes = int(info.Size())
	}

	provider := d.provider
	if dc.Provider != "" {
		provider = dc.Provider
	}

	start := time.Now()
	data, err := d.inner.Dispatch(ctx, dc)
	elapsed := time.Since(start)

	for _, h := range d.dispatchHooks {
		h(provider, dc.Step, elapsed, err)
	}

	if err != nil {
		return data, err
	}

	artifactBytes := len(data)

	d.tracker.Record(&billing.TokenRecord{
		CaseID:         dc.CaseID,
		Step:           dc.Step,
		PromptBytes:    promptBytes,
		ArtifactBytes:  artifactBytes,
		PromptTokens:   billing.EstimateTokens(promptBytes),
		ArtifactTokens: billing.EstimateTokens(artifactBytes),
		Timestamp:      start,
		WallClockMs:    elapsed.Milliseconds(),
	})

	return data, nil
}

// Inner returns the wrapped dispatcher for type-specific operations.
func (d *TokenTrackingDispatcher) Inner() bd.Dispatcher {
	return d.inner
}

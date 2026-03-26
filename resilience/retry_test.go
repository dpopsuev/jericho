package resilience

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetry_SucceedsFirstAttempt(t *testing.T) {
	var calls atomic.Int32
	err := Retry(context.Background(), RetryConfig{MaxAttempts: 3}, func() error {
		calls.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
}

func TestRetry_SucceedsAfterTransientFailure(t *testing.T) {
	var calls atomic.Int32
	err := Retry(context.Background(), RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
	}, func() error {
		n := calls.Add(1)
		if n < 3 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3", calls.Load())
	}
}

func TestRetry_ExhaustsAttempts(t *testing.T) {
	permanent := errors.New("permanent")
	var calls atomic.Int32
	err := Retry(context.Background(), RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
	}, func() error {
		calls.Add(1)
		return permanent
	})
	if !errors.Is(err, permanent) {
		t.Fatalf("err = %v, want permanent", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3", calls.Load())
	}
}

func TestRetry_NonRetryableError(t *testing.T) {
	authErr := errors.New("auth failed")
	var calls atomic.Int32
	err := Retry(context.Background(), RetryConfig{
		MaxAttempts: 5,
		BaseDelay:   1 * time.Millisecond,
		Retryable:   func(e error) bool { return !errors.Is(e, authErr) },
	}, func() error {
		calls.Add(1)
		return authErr
	})
	if !errors.Is(err, authErr) {
		t.Fatalf("err = %v, want authErr", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1 (non-retryable should not retry)", calls.Load())
	}
}

func TestRetry_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	err := Retry(ctx, RetryConfig{
		MaxAttempts: 10,
		BaseDelay:   1 * time.Second,
	}, func() error {
		return errors.New("fail")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestRetry_DefaultConfig(t *testing.T) {
	cfg := RetryConfig{}.defaults()
	if cfg.MaxAttempts != 3 {
		t.Fatalf("MaxAttempts = %d", cfg.MaxAttempts)
	}
	if cfg.BaseDelay != 500*time.Millisecond {
		t.Fatalf("BaseDelay = %v", cfg.BaseDelay)
	}
	if cfg.MaxDelay != 30*time.Second {
		t.Fatalf("MaxDelay = %v", cfg.MaxDelay)
	}
}

func TestBackoff_ExponentialGrowth(t *testing.T) {
	base := 100 * time.Millisecond
	max := 10 * time.Second

	d0 := backoff(0, base, max, false) // 100ms
	d1 := backoff(1, base, max, false) // 200ms
	d2 := backoff(2, base, max, false) // 400ms

	if d0 != 100*time.Millisecond {
		t.Fatalf("d0 = %v", d0)
	}
	if d1 != 200*time.Millisecond {
		t.Fatalf("d1 = %v", d1)
	}
	if d2 != 400*time.Millisecond {
		t.Fatalf("d2 = %v", d2)
	}
}

func TestBackoff_CapsAtMax(t *testing.T) {
	d := backoff(20, 100*time.Millisecond, 5*time.Second, false)
	if d != 5*time.Second {
		t.Fatalf("d = %v, want 5s cap", d)
	}
}

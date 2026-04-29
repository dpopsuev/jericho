package providers_test

import (
	"testing"
	"time"

	"github.com/dpopsuev/tangle/providers"
)

// Compile-time contract: stubWorkItem implements WorkItem.
type stubWorkItem struct{}

func (s *stubWorkItem) ID() uint64            { return 1 }
func (s *stubWorkItem) Input() string         { return "test prompt" }
func (s *stubWorkItem) Timeout() time.Duration { return 0 }

var _ providers.WorkItem = (*stubWorkItem)(nil)

func TestWorkItem_InterfaceCompiles(t *testing.T) {
	var item providers.WorkItem = &stubWorkItem{}
	if item.ID() != 1 {
		t.Fatal("expected ID 1")
	}
	if item.Input() != "test prompt" {
		t.Fatal("expected input 'test prompt'")
	}
	if item.Timeout() != 0 {
		t.Fatal("expected zero timeout")
	}
}

func TestWorkerHints_ZeroValue(t *testing.T) {
	hints := providers.WorkerHints{}
	if hints.Stickiness != 0 {
		t.Fatal("zero value stickiness should be 0")
	}
	if hints.PreferredTag != "" {
		t.Fatal("zero value preferred tag should be empty")
	}
	if hints.ConsecutiveMisses != 0 {
		t.Fatal("zero value consecutive misses should be 0")
	}
}

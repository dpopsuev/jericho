package providers_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	tangle "github.com/dpopsuev/tangle"
	"github.com/dpopsuev/tangle/providers"
)

// testWorkItem implements WorkItem for pool tests.
type testWorkItem struct {
	id    uint64
	input string
}

func (w *testWorkItem) ID() uint64            { return w.id }
func (w *testWorkItem) Input() string         { return w.input }
func (w *testWorkItem) Timeout() time.Duration { return 0 }

// memQueue is a simple in-memory Queue for testing.
type memQueue struct {
	mu      sync.Mutex
	items   []providers.WorkItem
	results map[uint64][]byte
	active  int

	// itemCh signals that a new item is available.
	itemCh chan struct{}
	// resultCh delivers results per work item ID.
	resultCh map[uint64]chan []byte
}

func newMemQueue() *memQueue {
	return &memQueue{
		results:  make(map[uint64][]byte),
		itemCh:   make(chan struct{}, 64),
		resultCh: make(map[uint64]chan []byte),
	}
}

func (q *memQueue) Enqueue(_ context.Context, item providers.WorkItem) error {
	q.mu.Lock()
	q.items = append(q.items, item)
	q.resultCh[item.ID()] = make(chan []byte, 1)
	q.mu.Unlock()

	q.itemCh <- struct{}{}
	return nil
}

func (q *memQueue) Pull(ctx context.Context) (providers.WorkItem, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-q.itemCh:
			q.mu.Lock()
			if len(q.items) == 0 {
				q.mu.Unlock()
				continue
			}
			item := q.items[0]
			q.items = q.items[1:]
			q.active++
			q.mu.Unlock()
			return item, nil
		}
	}
}

func (q *memQueue) PullWithHints(ctx context.Context, _ providers.WorkerHints) (providers.WorkItem, error) {
	return q.Pull(ctx)
}

func (q *memQueue) Submit(_ context.Context, id uint64, result []byte) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.results[id] = result
	q.active--

	if ch, ok := q.resultCh[id]; ok {
		ch <- result
	}
	return nil
}

func (q *memQueue) ActiveCount() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.active
}

// awaitResult blocks until the result for the given ID is submitted.
func (q *memQueue) awaitResult(ctx context.Context, id uint64) ([]byte, error) {
	q.mu.Lock()
	ch, ok := q.resultCh[id]
	q.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("no result channel for id %d", id)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		return res, nil
	}
}

func TestPool_ProcessesAllItems(t *testing.T) {
	q := newMemQueue()

	// Echo actor: returns input prefixed with "echo:".
	echoActor := tangle.CompleteFunc(func(_ context.Context, params tangle.CompletionParams) (*tangle.Completion, error) {
		return &tangle.Completion{Content: "echo:" + params.Prompt}, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Enqueue 3 work items.
	for i := range 3 {
		id := uint64(i + 1)
		err := q.Enqueue(ctx, &testWorkItem{id: id, input: fmt.Sprintf("task-%d", id)})
		if err != nil {
			t.Fatalf("enqueue item %d: %v", id, err)
		}
	}

	// Start pool with 2 workers.
	pool := providers.NewPool(q, echoActor, 2)
	pool.Start(ctx)

	// Wait for all 3 results.
	for i := range 3 {
		id := uint64(i + 1)
		result, err := q.awaitResult(ctx, id)
		if err != nil {
			t.Fatalf("await result %d: %v", id, err)
		}
		expected := fmt.Sprintf("echo:task-%d", id)
		if string(result) != expected {
			t.Errorf("item %d: got %q, want %q", id, string(result), expected)
		}
	}

	// Cancel and drain.
	cancel()
	pool.Drain()

	// Verify no items remain active.
	if q.ActiveCount() != 0 {
		t.Errorf("expected 0 active items, got %d", q.ActiveCount())
	}
}

func TestPool_ErrorSubmission(t *testing.T) {
	q := newMemQueue()

	failActor := tangle.CompleteFunc(func(_ context.Context, _ tangle.CompletionParams) (*tangle.Completion, error) {
		return nil, fmt.Errorf("boom")
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := q.Enqueue(ctx, &testWorkItem{id: 1, input: "fail-me"})
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	pool := providers.NewPool(q, failActor, 1)
	pool.Start(ctx)

	result, err := q.awaitResult(ctx, 1)
	if err != nil {
		t.Fatalf("await result: %v", err)
	}

	if string(result) != "error: boom" {
		t.Errorf("got %q, want %q", string(result), "error: boom")
	}

	cancel()
	pool.Drain()
}

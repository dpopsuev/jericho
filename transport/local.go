package transport

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// LocalTransport is an in-process, channel-based A2A transport.
// Handlers are registered by agent ID and invoked asynchronously
// when SendMessage is called. Suitable for same-process agent
// coordination (Papercup pattern).
type LocalTransport struct {
	mu       sync.RWMutex
	handlers map[string]Handler
	tasks    map[string]*taskEntry
	nextID   uint64
	closed   bool
}

type taskEntry struct {
	task *Task
	subs []chan Event
	mu   sync.Mutex
}

// NewLocalTransport creates a new in-process transport.
func NewLocalTransport() *LocalTransport {
	return &LocalTransport{
		handlers: make(map[string]Handler),
		tasks:    make(map[string]*taskEntry),
	}
}

// Register associates a Handler with the given agent ID.
// Subsequent SendMessage calls to this agent will invoke the handler.
func (t *LocalTransport) Register(agentID string, handler Handler) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.handlers[agentID] = handler
}

// Unregister removes the handler for the given agent ID.
func (t *LocalTransport) Unregister(agentID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.handlers, agentID)
}

// SendMessage dispatches a message to the agent identified by `to`.
// The handler runs in a goroutine; the returned Task starts in TaskSubmitted
// state and transitions to TaskWorking, then TaskCompleted or TaskFailed.
func (t *LocalTransport) SendMessage(ctx context.Context, to string, msg Message) (*Task, error) { //nolint:gocritic // interface conformance requires value param
	t.mu.RLock()
	handler, ok := t.handlers[to]
	closed := t.closed
	t.mu.RUnlock()

	if closed {
		return nil, fmt.Errorf("transport: closed")
	}
	if !ok {
		return nil, fmt.Errorf("transport: agent %q not registered", to)
	}

	taskID := fmt.Sprintf("task-%d", atomic.AddUint64(&t.nextID, 1))
	task := &Task{
		ID:    taskID,
		State: TaskSubmitted,
	}
	entry := &taskEntry{task: task}

	t.mu.Lock()
	t.tasks[taskID] = entry
	t.mu.Unlock()

	t.notify(entry, Event{TaskID: taskID, State: TaskSubmitted})

	go t.execute(ctx, handler, entry, msg)

	return task, nil
}

func (t *LocalTransport) execute(ctx context.Context, handler Handler, entry *taskEntry, msg Message) { //nolint:gocritic // called once from SendMessage, value copy acceptable
	entry.mu.Lock()
	entry.task.State = TaskWorking
	taskID := entry.task.ID
	entry.mu.Unlock()

	t.notify(entry, Event{TaskID: taskID, State: TaskWorking})

	result, err := handler(ctx, msg)

	entry.mu.Lock()
	defer entry.mu.Unlock()

	if err != nil {
		entry.task.State = TaskFailed
		entry.task.Error = err.Error()
		t.notifyLocked(entry, Event{TaskID: taskID, State: TaskFailed})
	} else {
		entry.task.State = TaskCompleted
		entry.task.Result = &result
		t.notifyLocked(entry, Event{TaskID: taskID, State: TaskCompleted, Data: &result})
	}

	// Close all subscriber channels — no more events for this task.
	for _, ch := range entry.subs {
		close(ch)
	}
	entry.subs = nil
}

// Subscribe returns a buffered channel that receives Events for state
// transitions of the given task. Returns an error if the task ID is unknown.
func (t *LocalTransport) Subscribe(_ context.Context, taskID string) (<-chan Event, error) {
	t.mu.RLock()
	entry, ok := t.tasks[taskID]
	t.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("transport: task %q not found", taskID)
	}

	ch := make(chan Event, 8)

	entry.mu.Lock()
	defer entry.mu.Unlock()

	// If task is already terminal, send the final state and close.
	if entry.task.State == TaskCompleted || entry.task.State == TaskFailed || entry.task.State == TaskCanceled {
		ch <- Event{TaskID: taskID, State: entry.task.State, Data: entry.task.Result}
		close(ch)
		return ch, nil
	}

	entry.subs = append(entry.subs, ch)
	return ch, nil
}

// Close unregisters all handlers and marks the transport as closed.
func (t *LocalTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.handlers = make(map[string]Handler)
	t.closed = true
	return nil
}

// notify sends an event to all subscribers of the task entry.
// Acquires entry.mu internally.
func (t *LocalTransport) notify(entry *taskEntry, ev Event) {
	entry.mu.Lock()
	defer entry.mu.Unlock()
	t.notifyLocked(entry, ev)
}

// notifyLocked sends an event to all subscribers. Caller must hold entry.mu.
func (*LocalTransport) notifyLocked(entry *taskEntry, ev Event) {
	for _, ch := range entry.subs {
		select {
		case ch <- ev:
		default:
			// subscriber channel full — drop event to avoid blocking
		}
	}
}

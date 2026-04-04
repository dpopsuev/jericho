package transport

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

// Sentinel errors for transport operations.
var (
	ErrTransportClosed   = errors.New("transport: closed")
	ErrAgentNotFound     = errors.New("transport: agent not registered")
	ErrTaskNotFound      = errors.New("transport: task not found")
	ErrTaskChanClosed    = errors.New("transport: task channel closed without terminal state")
	ErrTaskFailed        = errors.New("transport: task failed")
	ErrNoAgentsForRole   = errors.New("transport: no agents for role")
	ErrAlreadyRegistered = errors.New("transport: agent already registered")
)

// LocalTransport is an in-process, channel-based A2A transport.
// MsgHandlers are registered by agent ID and invoked asynchronously
// when SendMessage is called. Suitable for same-process agent
// coordination (Papercup pattern).
type LocalTransport struct {
	mu          sync.RWMutex
	handlers    map[AgentID]MsgHandler
	tasks       map[string]*taskEntry
	nextID      uint64
	closed      bool
	roles       *RoleRegistry
	roleCounter map[string]int
}

type taskEntry struct {
	task *Task
	subs []chan Event
	mu   sync.Mutex
}

// NewLocalTransport creates a new in-process transport.
func NewLocalTransport() *LocalTransport {
	return &LocalTransport{
		handlers:    make(map[AgentID]MsgHandler),
		tasks:       make(map[string]*taskEntry),
		roles:       NewRoleRegistry(),
		roleCounter: make(map[string]int),
	}
}

// Roles returns the transport's RoleRegistry for role-based routing.
func (t *LocalTransport) Roles() *RoleRegistry {
	return t.roles
}

// Register associates a MsgHandler with the given agent ID.
// Returns ErrAlreadyRegistered if the agent ID is already registered.
// Use Unregister first to replace an existing handler.
func (t *LocalTransport) Register(agentID AgentID, handler MsgHandler) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, exists := t.handlers[agentID]; exists {
		return fmt.Errorf("%w: %q", ErrAlreadyRegistered, agentID)
	}
	t.handlers[agentID] = handler
	return nil
}

// Unregister removes the handler for the given agent ID.
func (t *LocalTransport) Unregister(agentID AgentID) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.handlers, agentID)
}

// SendMessage dispatches a message to the agent identified by `to`.
// The handler runs in a goroutine; the returned Task starts in TaskSubmitted
// state and transitions to TaskWorking, then TaskCompleted or TaskFailed.
func (t *LocalTransport) SendMessage(ctx context.Context, to AgentID, msg Message) (*Task, error) { //nolint:gocritic // interface conformance requires value param
	t.mu.RLock()
	handler, ok := t.handlers[to]
	closed := t.closed
	t.mu.RUnlock()

	if closed {
		return nil, ErrTransportClosed
	}
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrAgentNotFound, to)
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

func (t *LocalTransport) execute(ctx context.Context, handler MsgHandler, entry *taskEntry, msg Message) { //nolint:gocritic // called once from SendMessage, value copy acceptable
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
		return nil, fmt.Errorf("%w: %q", ErrTaskNotFound, taskID)
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
	t.handlers = make(map[AgentID]MsgHandler)
	t.closed = true
	return nil
}

// Ask sends a message to the named agent and blocks until the handler
// responds or the context is canceled. Returns the response message
// on success, or an error if the handler failed or the context expired.
func (t *LocalTransport) Ask(ctx context.Context, to AgentID, msg Message) (Message, error) {
	task, err := t.SendMessage(ctx, to, msg)
	if err != nil {
		return Message{}, err
	}

	ch, err := t.Subscribe(ctx, task.ID)
	if err != nil {
		return Message{}, err
	}

	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return Message{}, fmt.Errorf("%w: %s", ErrTaskChanClosed, task.ID)
			}
			switch ev.State {
			case TaskCompleted:
				if ev.Data != nil {
					return *ev.Data, nil
				}
				return Message{}, nil
			case TaskFailed:
				return Message{}, fmt.Errorf("%w: %s", ErrTaskFailed, task.ID)
			}
		case <-ctx.Done():
			return Message{}, ctx.Err()
		}
	}
}

// SendToRole sends a message to one agent with the given role, selected
// via round-robin. Returns the Task for the chosen agent.
func (t *LocalTransport) SendToRole(ctx context.Context, role string, msg Message) (*Task, error) {
	agents := t.roles.AgentsForRole(role)
	if len(agents) == 0 {
		return nil, fmt.Errorf("%w: %q", ErrNoAgentsForRole, role)
	}

	t.mu.Lock()
	idx := t.roleCounter[role]
	t.roleCounter[role] = idx + 1
	t.mu.Unlock()

	target := AgentID(agents[idx%len(agents)])
	return t.SendMessage(ctx, target, msg)
}

// AskRole sends a message to one agent with the given role (round-robin)
// and blocks until the handler responds or the context is canceled.
func (t *LocalTransport) AskRole(ctx context.Context, role string, msg Message) (Message, error) {
	agents := t.roles.AgentsForRole(role)
	if len(agents) == 0 {
		return Message{}, fmt.Errorf("%w: %q", ErrNoAgentsForRole, role)
	}

	t.mu.Lock()
	idx := t.roleCounter[role]
	t.roleCounter[role] = idx + 1
	t.mu.Unlock()

	target := AgentID(agents[idx%len(agents)])
	return t.Ask(ctx, target, msg)
}

// Broadcast sends a message to ALL agents with the given role.
// Returns a Task per agent. Returns an error if no agents have the role.
func (t *LocalTransport) Broadcast(ctx context.Context, role string, msg Message) ([]*Task, error) {
	agents := t.roles.AgentsForRole(role)
	if len(agents) == 0 {
		return nil, fmt.Errorf("%w: %q", ErrNoAgentsForRole, role)
	}

	tasks := make([]*Task, 0, len(agents))
	for _, aid := range agents {
		task, err := t.SendMessage(ctx, AgentID(aid), msg)
		if err != nil {
			return tasks, fmt.Errorf("transport: broadcast to %s: %w", aid, err)
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
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

package orchestrate

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/dpopsuev/jericho/bugle"
)

// Log key constants for sloglint compliance.
const (
	logKeySession = "session"
	logKeyCount   = "count"
)

// Sentinel errors.
var (
	ErrAlreadyRunning = errors.New("workers are already running; call stop first")
	ErrNotRunning     = errors.New("no workers running")
	ErrStepFailed     = errors.New("pull failed")
)

// ResponderFactory creates a Responder for a worker. Called once per worker
// goroutine. The consumer controls how agents are spawned:
//   - Origami: ACP launcher + facade.AgentHandle
//   - Djinn: driver-based agent
//   - K8s: Pod exec client
//   - Test: StaticResponder
//
// The returned cleanup function is called when the worker exits.
type ResponderFactory func(ctx context.Context, workerID string) (bugle.Responder, func(), error)

// Manager manages a pool of agent workers that connect to an MCP endpoint.
type Manager struct {
	mu               sync.Mutex
	endpoint         string
	cancel           context.CancelFunc
	running          bool
	count            int
	session          string
	cfg              WorkerConfig
	completed        atomic.Int64
	errored          atomic.Int64
	responderFactory ResponderFactory
}

// NewManager creates a manager that spawns workers connecting to the given endpoint.
// The factory creates a Responder for each worker goroutine.
func NewManager(endpoint string, factory ResponderFactory, cfg WorkerConfig) *Manager {
	cfg.defaults()
	return &Manager{
		endpoint:         endpoint,
		cfg:              cfg,
		responderFactory: factory,
	}
}

// Start spawns N worker goroutines.
func (m *Manager) Start(ctx context.Context, session string, count int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return ErrAlreadyRunning
	}

	if count < 1 {
		count = 4
	}

	m.session = session
	m.count = count
	m.running = true
	m.completed.Store(0)
	m.errored.Store(0)

	workerCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	var wg sync.WaitGroup
	for i := range count {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			workerID := fmt.Sprintf("worker-%d", id+1)

			responder, cleanup, err := m.responderFactory(workerCtx, workerID)
			if err != nil {
				m.errored.Add(1)
				slog.ErrorContext(workerCtx, "responder factory failed",
					slog.String(logKeyWorker, workerID),
					slog.Any(logKeyError, err))
				return
			}
			defer cleanup()

			mcpSession, err := ConnectEndpoint(workerCtx, m.endpoint, workerID)
			if err != nil {
				m.errored.Add(1)
				slog.ErrorContext(workerCtx, "connect failed",
					slog.String(logKeyWorker, workerID),
					slog.Any(logKeyError, err))
				return
			}
			defer mcpSession.Close()

			if err := RunWorker(workerCtx, mcpSession, responder, session, workerID, m.cfg); err != nil {
				m.errored.Add(1)
				slog.ErrorContext(workerCtx, "worker failed",
					slog.String(logKeyWorker, workerID),
					slog.Any(logKeyError, err))
			} else {
				m.completed.Add(1)
				slog.InfoContext(workerCtx, "worker done",
					slog.String(logKeyWorker, workerID))
			}
		}(i)
	}

	go func() {
		wg.Wait()
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()
	}()

	slog.InfoContext(ctx, "workers started",
		slog.String(logKeySession, session),
		slog.Int(logKeyCount, count))

	return nil
}

// Stop kills all running workers.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return ErrNotRunning
	}
	m.cancel()
	return nil
}

// Health returns worker status.
func (m *Manager) Health() map[string]any {
	m.mu.Lock()
	running := m.running
	m.mu.Unlock()

	return map[string]any{
		"running":   running,
		"session":   m.session,
		"count":     m.count,
		"completed": m.completed.Load(),
		"errored":   m.errored.Load(),
	}
}

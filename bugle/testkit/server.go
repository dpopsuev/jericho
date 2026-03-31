package testkit

import (
	"context"
	"sync"

	"github.com/dpopsuev/jericho/bugle"
)

// MockServer implements bugle.Server with configurable handlers and call tracking.
type MockServer struct {
	mu sync.Mutex

	startFn  func(bugle.StartRequest) (bugle.StartResponse, error)
	pullFn   func(bugle.PullRequest) (bugle.PullResponse, error)
	pushFn   func(bugle.PushRequest) (bugle.PushResponse, error)
	cancelFn func(bugle.CancelRequest) (bugle.CancelResponse, error)
	statusFn func(bugle.StatusRequest) (bugle.StatusResponse, error)

	pushes  []bugle.PushRequest
	pulls   int
	starts  int
	cancels int
}

// NewMockServer creates a mock with default handlers that return zero values.
func NewMockServer() *MockServer {
	return &MockServer{}
}

// OnStart sets the handler for start requests.
func (s *MockServer) OnStart(fn func(bugle.StartRequest) (bugle.StartResponse, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.startFn = fn
}

// OnPull sets the handler for pull requests.
func (s *MockServer) OnPull(fn func(bugle.PullRequest) (bugle.PullResponse, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pullFn = fn
}

// OnPush sets the handler for push requests.
func (s *MockServer) OnPush(fn func(bugle.PushRequest) (bugle.PushResponse, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pushFn = fn
}

// OnCancel sets the handler for cancel requests.
func (s *MockServer) OnCancel(fn func(bugle.CancelRequest) (bugle.CancelResponse, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancelFn = fn
}

// OnStatus sets the handler for status requests.
func (s *MockServer) OnStatus(fn func(bugle.StatusRequest) (bugle.StatusResponse, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statusFn = fn
}

// Start implements bugle.Server.
func (s *MockServer) Start(_ context.Context, req bugle.StartRequest) (bugle.StartResponse, error) {
	s.mu.Lock()
	s.starts++
	fn := s.startFn
	s.mu.Unlock()
	if fn != nil {
		return fn(req)
	}
	return bugle.StartResponse{SessionID: "mock-session", TotalItems: 0, Status: "running"}, nil
}

// Pull implements bugle.Server.
func (s *MockServer) Pull(_ context.Context, req bugle.PullRequest) (bugle.PullResponse, error) {
	s.mu.Lock()
	s.pulls++
	fn := s.pullFn
	s.mu.Unlock()
	if fn != nil {
		return fn(req)
	}
	return bugle.PullResponse{Done: true}, nil
}

// Push implements bugle.Server.
func (s *MockServer) Push(_ context.Context, req bugle.PushRequest) (bugle.PushResponse, error) {
	s.mu.Lock()
	s.pushes = append(s.pushes, req)
	fn := s.pushFn
	s.mu.Unlock()
	if fn != nil {
		return fn(req)
	}
	return bugle.PushResponse{OK: true}, nil
}

// Cancel implements bugle.Server.
func (s *MockServer) Cancel(_ context.Context, req bugle.CancelRequest) (bugle.CancelResponse, error) {
	s.mu.Lock()
	s.cancels++
	fn := s.cancelFn
	s.mu.Unlock()
	if fn != nil {
		return fn(req)
	}
	return bugle.CancelResponse{OK: true, Canceled: 1}, nil
}

// Status implements bugle.Server.
func (s *MockServer) Status(_ context.Context, req bugle.StatusRequest) (bugle.StatusResponse, error) {
	s.mu.Lock()
	fn := s.statusFn
	s.mu.Unlock()
	if fn != nil {
		return fn(req)
	}
	return bugle.StatusResponse{SessionID: req.SessionID, Progress: bugle.Progress{}}, nil
}

// --- Inspection methods ---

// PushCount returns the number of push calls received.
func (s *MockServer) PushCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.pushes)
}

// Pushes returns all push requests received.
func (s *MockServer) Pushes() []bugle.PushRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]bugle.PushRequest, len(s.pushes))
	copy(cp, s.pushes)
	return cp
}

// LastPush returns the most recent push request. Panics if none.
func (s *MockServer) LastPush() bugle.PushRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pushes[len(s.pushes)-1]
}

// PullCount returns the number of pull calls received.
func (s *MockServer) PullCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pulls
}

// StartCount returns the number of start calls received.
func (s *MockServer) StartCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.starts
}

// CancelCount returns the number of cancel calls received.
func (s *MockServer) CancelCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cancels
}

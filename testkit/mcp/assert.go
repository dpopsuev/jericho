package mcp

import (
	"testing"

	"github.com/dpopsuev/troupe/internal/protocol"
	"github.com/dpopsuev/troupe/signal"
)

// AssertPushCount verifies the number of push calls.
func AssertPushCount(t *testing.T, server *MockServer, want int) {
	t.Helper()
	if got := server.PushCount(); got != want {
		t.Errorf("PushCount() = %d, want %d", got, want)
	}
}

// AssertPushStatus verifies the last push has the given status.
func AssertPushStatus(t *testing.T, server *MockServer, want protocol.SubmitStatus) {
	t.Helper()
	if server.PushCount() == 0 {
		t.Fatal("no pushes received")
	}
	got := server.LastPush().Status
	if got != want {
		t.Errorf("last push status = %q, want %q", got, want)
	}
}

// AssertAndonLevel verifies the last push has the given andon level.
func AssertAndonLevel(t *testing.T, server *MockServer, want signal.AndonLevel) {
	t.Helper()
	if server.PushCount() == 0 {
		t.Fatal("no pushes received")
	}
	push := server.LastPush()
	if push.Andon == nil {
		t.Fatal("last push has no andon")
	}
	if push.Andon.Level != want {
		t.Errorf("last push andon level = %q, want %q", push.Andon.Level, want)
	}
}

// AssertBudgetReported verifies the last push has budget_actual set.
func AssertBudgetReported(t *testing.T, server *MockServer) {
	t.Helper()
	if server.PushCount() == 0 {
		t.Fatal("no pushes received")
	}
	if server.LastPush().Budget == nil {
		t.Error("last push has no budget_actual")
	}
}

// AssertNoPushes verifies no push calls were made (e.g., after abort).
func AssertNoPushes(t *testing.T, server *MockServer) {
	t.Helper()
	if got := server.PushCount(); got != 0 {
		t.Errorf("expected no pushes, got %d", got)
	}
}

// AssertWorkerID verifies all pushes used the expected worker_id.
func AssertWorkerID(t *testing.T, server *MockServer, want string) {
	t.Helper()
	pushes := server.Pushes()
	for i := range pushes {
		if pushes[i].WorkerID != want {
			t.Errorf("push[%d].WorkerID = %q, want %q", i, pushes[i].WorkerID, want)
		}
	}
}

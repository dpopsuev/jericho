package mcp

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/dpopsuev/jericho/internal/protocol"
)

// runLoop simulates the protocol loop without MCP transport.
// This is the testable core of orchestrate.RunWorker.
//
//nolint:unparam // sessionID varies in production, fixed in tests for simplicity
func runLoop(ctx context.Context, server protocol.Server, responder protocol.Responder, sessionID, workerID string, andonFn func() *protocol.Andon, budgetFn func() *protocol.BudgetActual) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		pullResp, err := server.Pull(ctx, protocol.PullRequest{
			Action:    protocol.ActionPull,
			SessionID: sessionID,
			WorkerID:  workerID,
		})
		if err != nil {
			return err
		}

		if pullResp.Andon == protocol.AndonDead {
			return nil
		}
		if pullResp.Done {
			return nil
		}
		if !pullResp.Available {
			continue
		}

		response, err := responder.RespondTo(ctx, pullResp.PromptContent)

		pushReq := protocol.PushRequest{
			Action:     protocol.ActionPush,
			SessionID:  sessionID,
			WorkerID:   workerID,
			DispatchID: pullResp.DispatchID,
			Item:       pullResp.Item,
		}

		if err != nil {
			pushReq.Status = protocol.StatusBlocked
			pushReq.Fields = []byte(`{"reason":"` + err.Error() + `"}`)
		} else {
			pushReq.Status = protocol.StatusOk
			pushReq.Fields = []byte(response)
		}

		if andonFn != nil {
			pushReq.Andon = andonFn()
		}
		if budgetFn != nil {
			pushReq.Budget = budgetFn()
		}

		if _, pushErr := server.Push(ctx, pushReq); pushErr != nil {
			return pushErr
		}
	}
}

func TestProtocol_WorkerAbortsOnAndonDead(t *testing.T) {
	server := NewMockServer()
	server.OnPull(func(_ protocol.PullRequest) (protocol.PullResponse, error) {
		return protocol.PullResponse{Andon: protocol.AndonDead}, nil
	})

	err := runLoop(context.Background(), server, &StaticResponder{Response: "x"}, "s1", "w1", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	AssertNoPushes(t, server)
}

func TestProtocol_WorkerPushesBlockedOnResponderFailure(t *testing.T) {
	var pullCount atomic.Int32
	server := NewMockServer()
	server.OnPull(func(_ protocol.PullRequest) (protocol.PullResponse, error) {
		n := pullCount.Add(1)
		if n == 1 {
			return protocol.PullResponse{Available: true, Item: "F0", DispatchID: 1, PromptContent: "test"}, nil
		}
		return protocol.PullResponse{Done: true}, nil
	})

	responder := &FailingResponder{Err: errors.New("agent crashed")}

	err := runLoop(context.Background(), server, responder, "s1", "w1", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	AssertPushCount(t, server, 1)
	AssertPushStatus(t, server, protocol.StatusBlocked)
}

func TestProtocol_BudgetActualIncludedWhenFuncSet(t *testing.T) {
	var pullCount atomic.Int32
	server := NewMockServer()
	server.OnPull(func(_ protocol.PullRequest) (protocol.PullResponse, error) {
		n := pullCount.Add(1)
		if n == 1 {
			return protocol.PullResponse{Available: true, Item: "F0", DispatchID: 1, PromptContent: "test"}, nil
		}
		return protocol.PullResponse{Done: true}, nil
	})

	budgetFn := func() *protocol.BudgetActual {
		return &protocol.BudgetActual{TokensIn: 500, TokensOut: 300}
	}

	err := runLoop(context.Background(), server, &StaticResponder{Response: `{"ok":true}`}, "s1", "w1", nil, budgetFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	AssertPushCount(t, server, 1)
	AssertBudgetReported(t, server)
}

func TestProtocol_AndonIncludedWhenFuncSet(t *testing.T) {
	var pullCount atomic.Int32
	server := NewMockServer()
	server.OnPull(func(_ protocol.PullRequest) (protocol.PullResponse, error) {
		n := pullCount.Add(1)
		if n == 1 {
			return protocol.PullResponse{Available: true, Item: "F0", DispatchID: 1, PromptContent: "test"}, nil
		}
		return protocol.PullResponse{Done: true}, nil
	})

	andonFn := func() *protocol.Andon {
		return &protocol.Andon{Level: protocol.AndonDegraded, Priority: protocol.PriorityDegraded, Message: "82% tokens"}
	}

	err := runLoop(context.Background(), server, &StaticResponder{Response: `{"ok":true}`}, "s1", "w1", andonFn, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	AssertPushCount(t, server, 1)
	AssertAndonLevel(t, server, protocol.AndonDegraded)
}

func TestProtocol_WorkerIDSentOnEveryPush(t *testing.T) {
	var pullCount atomic.Int32
	server := NewMockServer()
	server.OnPull(func(_ protocol.PullRequest) (protocol.PullResponse, error) {
		n := pullCount.Add(1)
		if n <= 3 {
			return protocol.PullResponse{Available: true, Item: "F0", DispatchID: int64(n), PromptContent: "test"}, nil
		}
		return protocol.PullResponse{Done: true}, nil
	})

	err := runLoop(context.Background(), server, &StaticResponder{Response: `{}`}, "s1", "[Azure·Cerulean|Analyst]", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	AssertPushCount(t, server, 3)
	AssertWorkerID(t, server, "[Azure·Cerulean|Analyst]")
}

func TestProtocol_MultipleItemsProcessedSequentially(t *testing.T) {
	var pullCount atomic.Int32
	server := NewMockServer()
	server.OnPull(func(_ protocol.PullRequest) (protocol.PullResponse, error) {
		n := pullCount.Add(1)
		if n <= 5 {
			return protocol.PullResponse{Available: true, Item: "item", DispatchID: int64(n), PromptContent: "test"}, nil
		}
		return protocol.PullResponse{Done: true}, nil
	})

	scripted := NewScriptedResponder("r1", "r2", "r3", "r4", "r5")
	err := runLoop(context.Background(), server, scripted, "s1", "w1", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	AssertPushCount(t, server, 5)
	if scripted.CallCount() != 5 {
		t.Errorf("ScriptedResponder calls = %d, want 5", scripted.CallCount())
	}
}

func TestProtocol_DoneSignalStopsLoop(t *testing.T) {
	server := NewMockServer()
	server.OnPull(func(_ protocol.PullRequest) (protocol.PullResponse, error) {
		return protocol.PullResponse{Done: true}, nil
	})

	err := runLoop(context.Background(), server, &StaticResponder{Response: "x"}, "s1", "w1", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	AssertNoPushes(t, server)
}

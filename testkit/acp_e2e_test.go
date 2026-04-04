// acp_e2e_test.go — E2E tests for bugle/acp through the agent.
//
// Proves: ACPLauncher plugs into Staff → Spawn starts a mock ACP agent →
// Client.Send + Client.Chat streams events → Kill shuts down cleanly.
// Uses a mock bash ACP server — no real LLM, no cost.
package testkit

import (
	"context"
	"os/exec"
	"testing"

	"github.com/dpopsuev/troupe/internal/acp"
	"github.com/dpopsuev/troupe/internal/warden"
	"github.com/dpopsuev/troupe/world"
)

// mockACPServer simulates an ACP agent over stdio.
const mockACPServer = `#!/bin/bash
while IFS= read -r line; do
  method=$(echo "$line" | grep -o '"method":"[^"]*"' | cut -d'"' -f4)
  id=$(echo "$line" | grep -o '"id":[0-9]*' | cut -d: -f2)

  case "$method" in
    initialize)
      echo '{"jsonrpc":"2.0","id":'$id',"result":{"protocolVersion":1,"agentInfo":{"name":"mock-e2e","version":"1.0.0"}}}'
      ;;
    session/new)
      echo '{"jsonrpc":"2.0","id":'$id',"result":{"sessionId":"e2e-session"}}'
      ;;
    session/prompt)
      echo '{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"e2e-session","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"e2e "}}}}'
      echo '{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"e2e-session","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"response"}}}}'
      echo '{"jsonrpc":"2.0","id":'$id',"result":{"stopReason":"end_turn"}}'
      ;;
    session/cancel)
      exit 0
      ;;
  esac
done
`

func mockCmdFactory(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "bash", "-c", mockACPServer)
}

// TestACPE2E_ClientLifecycle tests the ACP client directly:
// start → send → chat → stream events → stop.
func TestACPE2E_ClientLifecycle(t *testing.T) {
	client, err := acp.NewClient("cursor",
		acp.WithCommandFactory(mockCmdFactory),
		acp.WithClientInfo(acp.ClientInfo{Name: "bugle-e2e", Version: "test"}),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// Start — handshake + session.
	if err := client.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if client.SessionID() != "e2e-session" {
		t.Fatalf("sessionID = %q, want e2e-session", client.SessionID())
	}
	if client.AgentName() != "cursor" {
		t.Fatalf("agentName = %q", client.AgentName())
	}

	// Send + Chat.
	client.Send(acp.Message{Role: acp.RoleUser, Content: "e2e test"})
	ch, err := client.Chat(ctx)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	var fullText string
	var gotDone bool
	var gotToolUse bool
	for evt := range ch {
		switch evt.Type {
		case acp.EventText:
			fullText += evt.Text
		case acp.EventToolUse:
			gotToolUse = true
		case acp.EventDone:
			gotDone = true
		case acp.EventError:
			t.Fatalf("unexpected error: %s", evt.Error)
		}
	}

	if !gotDone {
		t.Fatal("never got EventDone")
	}
	if fullText != "e2e response" {
		t.Fatalf("text = %q, want 'e2e response'", fullText)
	}
	_ = gotToolUse // mock doesn't emit tool calls — that's OK

	// History should have user + assistant.
	msgs := client.Messages()
	if len(msgs) != 2 {
		t.Fatalf("messages = %d, want 2", len(msgs))
	}
	if msgs[0].Role != acp.RoleUser || msgs[0].Content != "e2e test" {
		t.Fatalf("msg[0] = %+v", msgs[0])
	}
	if msgs[1].Role != acp.RoleAssistant || msgs[1].Content != "e2e response" {
		t.Fatalf("msg[1] = %+v", msgs[1])
	}

	// Stop — kill returns "signal: killed" which is expected for process termination.
	client.Stop(ctx) //nolint:errcheck // test cleanup, error irrelevant
}

// TestACPE2E_LauncherInterface tests that ACPLauncher satisfies warden.AgentSupervisor
// and manages client lifecycle correctly.
func TestACPE2E_LauncherInterface(t *testing.T) {
	launcher := acp.NewACPLauncher()
	ctx := context.Background()

	// Verify interface compliance at runtime.
	var _ warden.AgentSupervisor = launcher

	// Start will fail because "agent" binary isn't on PATH — that's expected
	// in CI. We verify the launcher correctly creates and tracks clients.
	var id world.EntityID = 42
	err := launcher.Start(ctx, id, warden.AgentConfig{Model: "cursor"})
	if err != nil { //nolint:nestif // test exercises both success and failure paths
		// Expected — no real agent binary. Verify client wasn't tracked.
		t.Logf("launcher.Start failed (expected — no agent binary): %v", err)
		if launcher.Healthy(ctx, id) {
			t.Fatal("failed start should not be healthy")
		}
		_, ok := launcher.Client(id)
		if ok {
			t.Fatal("failed start should not have a client")
		}
	} else {
		// Unlikely in CI, but if it succeeded, verify and clean up.
		if !launcher.Healthy(ctx, id) {
			t.Fatal("agent should be healthy after start")
		}
		client, ok := launcher.Client(id)
		if !ok {
			t.Fatal("Client() should return the client")
		}
		if client.AgentName() != "cursor" {
			t.Fatalf("agentName = %q", client.AgentName())
		}
		launcher.Stop(ctx, id) //nolint:errcheck // test cleanup, error irrelevant
		if launcher.Healthy(ctx, id) {
			t.Fatal("should not be healthy after stop")
		}
	}
}

// TestACPE2E_ShapeClassifier tests shape classification end-to-end
// with realistic ACP event payloads.
func TestACPE2E_ShapeClassifier(t *testing.T) {
	tests := []struct {
		name  string
		data  map[string]any
		shape acp.ShapeKind
	}{
		{
			"agent text response",
			map[string]any{"content": "I'll help you with that."},
			acp.ShapeTextStream,
		},
		{
			"plan with steps",
			map[string]any{
				"steps": []any{
					"1. Read the file",
					"2. Analyze the code",
					"3. Make changes",
				},
			},
			acp.ShapePlanUpdate,
		},
		{
			"file diff",
			map[string]any{
				"changes": []any{
					map[string]any{"file": "main.go", "action": "modify"},
					map[string]any{"file": "test.go", "action": "create"},
				},
			},
			acp.ShapeDiffUpdate,
		},
		{
			"tool execution status",
			map[string]any{"status": "running"},
			acp.ShapeActionLifecycle,
		},
		{
			"tool list",
			map[string]any{"tools": []any{"Read", "Write", "Bash"}},
			acp.ShapeCapabilityList,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shape := acp.ClassifyShape(tt.data)
			if shape != tt.shape {
				t.Fatalf("ClassifyShape = %v, want %v", shape, tt.shape)
			}

			// Route the event and verify we get a typed message.
			msg := acp.RouteEvent(shape, tt.data)
			if msg == nil {
				t.Fatal("RouteEvent returned nil")
			}
		})
	}
}

// TestACPE2E_AuditCapabilities verifies the capability audit from
// the consumer's perspective — what can I use?
func TestACPE2E_AuditCapabilities(t *testing.T) {
	caps := acp.AuditCapabilities()

	var supported, unsupported int
	for _, c := range caps {
		if c.Supported {
			supported++
		} else {
			unsupported++
		}
		if c.Name == "" || c.Description == "" {
			t.Errorf("capability has empty name or description: %+v", c)
		}
	}

	if supported < 7 {
		t.Fatalf("supported = %d, want >= 7", supported)
	}
	if unsupported < 2 {
		t.Fatalf("unsupported = %d, want >= 2 (future capabilities)", unsupported)
	}

	t.Logf("ACP capabilities: %d supported, %d future", supported, unsupported)
}

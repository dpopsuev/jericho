//go:build e2e

// acceptance_test.go — E2E acceptance tests with real AI agents.
// Gated by build tag + binary availability. Run with:
//
//	go test ./testkit/ -tags=e2e -run TestAcceptance -v -timeout 120s
//	JERICHO_TEST_AGENT=claude go test ./testkit/ -tags=e2e -v -timeout 120s
//
// Cost: ~$0.05 per run (real API calls).
package testkit

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/dpopsuev/jericho/acp"
)

// testAgent returns the CLI agent to test. Default: cursor.
// Override with JERICHO_TEST_AGENT env var.
func testAgent(t *testing.T) string {
	t.Helper()
	agent := os.Getenv("JERICHO_TEST_AGENT")
	if agent == "" {
		agent = "cursor"
	}
	return agent
}

// requireAgent skips the test if the agent binary is not in PATH.
func requireAgent(t *testing.T, name string) string {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping real agent test in -short mode")
	}

	// Map agent name to binary.
	binaries := map[string]string{
		"cursor": "agent",
		"claude": "claude",
		"gemini": "gemini",
		"codex":  "codex",
	}
	binary, ok := binaries[name]
	if !ok {
		binary = name
	}

	path, err := exec.LookPath(binary)
	if err != nil {
		t.Skipf("%s (%s) not found in PATH — skipping", name, binary)
	}
	return path
}

// --- Smoke Tests ---

func TestAcceptance_Smoke_AgentExists(t *testing.T) {
	agent := testAgent(t)
	path := requireAgent(t, agent)
	t.Logf("%s found at %s", agent, path)
}

// --- Real Agent Tests ---

func TestAcceptance_RealAgent_RespondTo(t *testing.T) {
	agent := testAgent(t)
	requireAgent(t, agent)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := acp.NewClient(agent,
		acp.WithHandshakeTimeout(15*time.Second),
		acp.WithSessionTimeout(15*time.Second),
	)
	if err != nil {
		t.Fatalf("NewClient(%s): %v", agent, err)
	}

	if err := client.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		client.Stop(stopCtx) //nolint:errcheck // test cleanup
	}()

	// Send a simple prompt.
	client.Send(acp.Message{
		Role:    acp.RoleUser,
		Content: "Respond with exactly: JERICHO_E2E_OK",
	})

	ch, err := client.Chat(ctx)
	if err != nil {
		// Auth failures are expected when API key is not set.
		t.Skipf("Chat failed (likely auth): %v", err)
	}

	var gotText bool
	var gotError bool
	var totalIn, totalOut int
	for evt := range ch {
		switch evt.Type {
		case acp.EventText:
			gotText = true
		case acp.EventError:
			gotError = true
		case acp.EventDone:
			if evt.Usage != nil {
				totalIn += evt.Usage.InputTokens
				totalOut += evt.Usage.OutputTokens
			}
		}
	}

	if gotError && !gotText {
		t.Skipf("agent returned error (likely auth) — set API key for %s", agent)
	}

	if !gotText {
		t.Fatal("no text response received from real agent")
	}

	t.Logf("agent=%s tokens_in=%d tokens_out=%d", agent, totalIn, totalOut)

	// Budget ceiling — fail if test is unreasonably expensive.
	const maxTokens = 5000
	if totalIn+totalOut > maxTokens {
		t.Errorf("token budget exceeded: %d > %d", totalIn+totalOut, maxTokens)
	}
}

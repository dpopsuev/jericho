// gate.go — Gatekeeper interface for collective boundary control.
//
// One contract, two roles: Ingress (bouncer) and Egress (reviewer).
// Both can be LLM agents or code functions. The collective is a Room
// with two doors — the operator never sees inside.
package collective

import (
	"context"
	"strings"

	"github.com/dpopsuev/tangle"
)

// Gatekeeper decides if content should pass through a collective boundary.
// Returns (allowed, reason, error). Reason explains why content was
// rejected or provides audit context on pass.
type Gatekeeper interface {
	Pass(ctx context.Context, content string) (bool, string, error)
}

// AgentGatekeeper wraps an Solo as a Gatekeeper. The agent's system prompt
// defines its gating policy. The agent is asked to evaluate content
// and respond with PASS or REJECT.
type AgentGatekeeper struct {
	Agent troupe.Agent
}

// Pass asks the agent to evaluate the content. The agent should respond
// with PASS or REJECT followed by a reason. Defaults to PASS on
// ambiguous responses (fail-open).
func (g *AgentGatekeeper) Pass(ctx context.Context, content string) (allowed bool, reason string, err error) {
	prompt := "Evaluate the following content. Respond with exactly PASS or REJECT followed by your reason.\n\n" + content
	resp, err := g.Agent.Perform(ctx, prompt)
	if err != nil {
		// Fail-open on error — don't block the room because the gate is broken.
		return true, "", err
	}

	upper := strings.ToUpper(strings.TrimSpace(resp))
	if strings.HasPrefix(upper, "REJECT") {
		return false, resp, nil
	}
	return true, resp, nil
}

// GateKeeper wraps a troupe.Gate as a Gatekeeper. The Gate receives
// the content string as its subject.
type GateKeeper struct {
	gate troupe.Gate
}

// NewGateKeeper creates a Gatekeeper backed by a Gate predicate.
func NewGateKeeper(g troupe.Gate) *GateKeeper {
	return &GateKeeper{gate: g}
}

// Pass delegates to the underlying Gate with content as subject.
func (g *GateKeeper) Pass(ctx context.Context, content string) (bool, string, error) {
	return g.gate(ctx, content)
}

// Compile-time checks.
var (
	_ Gatekeeper = (*AgentGatekeeper)(nil)
	_ Gatekeeper = (*GateKeeper)(nil)
)

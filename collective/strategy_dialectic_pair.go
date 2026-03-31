// strategy_dialectic_pair.go — DialecticPair: debate then pair-execute.
//
// Two-phase strategy inspired by XP pair programming + Hegelian dialectic:
// Phase 1: Dialectic convergence on a plan (thesis/antithesis)
// Phase 2: Pair execution — Driver writes, Navigator gates
package collective

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dpopsuev/jericho/agent"
)

// Sentinel errors for dialectic pair.
var (
	ErrDialecticPairRequiresTwo = errors.New("dialectic pair requires exactly 2 agents")
	ErrNavigatorRejected        = errors.New("navigator rejected driver output")
)

// DialecticPair chains two phases with the same two agents:
// Phase 1: Dialectic — stress-test approach until convergence
// Phase 2: PairExecution — Driver implements, Navigator reviews
type DialecticPair struct {
	MaxRounds int // dialectic convergence limit (default 5)
}

// Orchestrate runs dialectic convergence then pair execution.
func (dp *DialecticPair) Orchestrate(ctx context.Context, prompt string, agents []*agent.Solo) (string, error) {
	if len(agents) != 2 { //nolint:mnd // exactly 2 agents required by design
		return "", fmt.Errorf("%w, got %d", ErrDialecticPairRequiresTwo, len(agents))
	}

	maxRounds := dp.MaxRounds
	if maxRounds <= 0 {
		maxRounds = 5
	}

	driver, navigator := agents[0], agents[1]

	// Phase 1: Dialectic convergence on a plan.
	dialectic := &Dialectic{MaxRounds: maxRounds}
	plan, err := dialectic.Orchestrate(ctx, prompt, agents)
	if err != nil {
		return "", fmt.Errorf("dialectic pair phase 1 (plan): %w", err)
	}

	// Phase 2: Driver executes the plan, Navigator reviews.
	execPrompt := fmt.Sprintf(
		"Agreed plan:\n%s\n\nExecute this plan. Produce the final output.",
		plan,
	)
	driverResp, err := driver.Ask(ctx, execPrompt)
	if err != nil {
		return plan, nil // driver failed, return the plan at least
	}

	// Navigator reviews driver output.
	reviewPrompt := fmt.Sprintf(
		"Original request:\n%s\n\nAgreed plan:\n%s\n\nDriver output:\n%s\n\n"+
			"Review this output against the plan. If it correctly implements the plan, "+
			"respond with exactly APPROVED. Otherwise, explain what's wrong.",
		prompt, plan, driverResp,
	)
	navResp, err := navigator.Ask(ctx, reviewPrompt)
	if err != nil {
		return driverResp, nil // navigator error, return driver output
	}

	if strings.Contains(strings.ToUpper(navResp), "APPROVED") {
		return driverResp, nil
	}

	// Navigator rejected — driver gets one revision attempt.
	revisePrompt := fmt.Sprintf(
		"Your output was reviewed and rejected.\n\nFeedback:\n%s\n\nRevise your output to address the feedback.",
		navResp,
	)
	revised, err := driver.Ask(ctx, revisePrompt)
	if err != nil {
		return driverResp, nil // revision failed, return original
	}

	return revised, nil
}

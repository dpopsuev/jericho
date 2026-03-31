// strategy_arbiter.go — Arbiter: thesis + antithesis + judge.
//
// Three-agent debate where a judge has final authority. The judge
// decides after each round: AFFIRM (thesis wins), AMEND (revise),
// or REMAND (try again). Reduces debate rounds by having authority.
package collective

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dpopsuev/jericho/facade"
)

// ErrTooFewAgentsArbiter is returned when arbiter has fewer than 3 agents.
var ErrTooFewAgentsArbiter = errors.New("arbiter requires at least 3 agents")

// ArbiterDecision is the judge's ruling.
type ArbiterDecision string

const (
	DecisionAffirm ArbiterDecision = "AFFIRM" // thesis wins as-is
	DecisionAmend  ArbiterDecision = "AMEND"  // thesis needs revision
	DecisionRemand ArbiterDecision = "REMAND" // start over
)

// Arbiter is a CollectiveStrategy with a judge agent.
// Requires at least 3 agents: thesis (0), antithesis (1), judge (2).
type Arbiter struct {
	MaxRounds int // default 3
}

func (a *Arbiter) defaults() int {
	if a.MaxRounds <= 0 {
		return 3
	}
	return a.MaxRounds
}

// Orchestrate runs the arbiter debate. Judge decides after each round.
func (a *Arbiter) Orchestrate(ctx context.Context, prompt string, agents []*facade.AgentHandle) (string, error) {
	if len(agents) < 3 {
		return "", fmt.Errorf("%w, got %d", ErrTooFewAgentsArbiter, len(agents))
	}

	maxRounds := a.defaults()
	thesis, anti, judge := agents[0], agents[1], agents[2]

	// Thesis drafts.
	thesisResp, err := thesis.Ask(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("thesis draft: %w", err)
	}

	for round := range maxRounds {
		// Antithesis challenges.
		antiPrompt := fmt.Sprintf(
			"Original request:\n%s\n\nThesis (round %d):\n%s\n\n"+
				"Challenge this response. Identify flaws and alternatives.",
			prompt, round+1, thesisResp,
		)
		antiResp, err := anti.Ask(ctx, antiPrompt)
		if err != nil {
			return thesisResp, nil
		}

		// Judge decides.
		judgePrompt := fmt.Sprintf(
			"Original request:\n%s\n\nThesis:\n%s\n\nAntithesis:\n%s\n\n"+
				"You are the arbiter. Evaluate both positions and decide:\n"+
				"- Reply AFFIRM if the thesis is adequate\n"+
				"- Reply AMEND with specific revision instructions if the thesis needs improvement\n"+
				"- Reply REMAND if both positions are inadequate and a fresh start is needed\n\n"+
				"Start your response with exactly one of: AFFIRM, AMEND, or REMAND.",
			prompt, thesisResp, antiResp,
		)
		judgeResp, err := judge.Ask(ctx, judgePrompt)
		if err != nil {
			return thesisResp, nil
		}

		decision := parseDecision(judgeResp)

		switch decision {
		case DecisionAffirm:
			return thesisResp, nil

		case DecisionAmend:
			revisePrompt := fmt.Sprintf(
				"Original request:\n%s\n\nYour previous response:\n%s\n\n"+
					"Critique:\n%s\n\nJudge's instructions:\n%s\n\n"+
					"Revise your response following the judge's instructions.",
				prompt, thesisResp, antiResp, judgeResp,
			)
			revised, err := thesis.Ask(ctx, revisePrompt)
			if err != nil {
				return thesisResp, nil
			}
			thesisResp = revised

		case DecisionRemand:
			// Fresh start with judge's feedback.
			remandPrompt := fmt.Sprintf(
				"Original request:\n%s\n\nPrevious attempt was remanded by the judge:\n%s\n\n"+
					"Start fresh with a new approach.",
				prompt, judgeResp,
			)
			fresh, err := thesis.Ask(ctx, remandPrompt)
			if err != nil {
				return thesisResp, nil
			}
			thesisResp = fresh
		}
	}

	return thesisResp, nil
}

func parseDecision(response string) ArbiterDecision {
	upper := strings.ToUpper(strings.TrimSpace(response))
	if strings.HasPrefix(upper, string(DecisionAffirm)) {
		return DecisionAffirm
	}
	if strings.HasPrefix(upper, string(DecisionRemand)) {
		return DecisionRemand
	}
	return DecisionAmend // default to amend if unclear
}

// strategy_dialectic.go — Dialectic: thesis-antithesis ping-pong convergence.
//
// The default CollectiveStrategy. Thesis drafts, antithesis challenges,
// thesis revises. Repeat until antithesis says CONVERGED or max rounds
// exhausted. Thesis has last word — it naturally produces the synthesis.
package collective

import (
	"context"
	"fmt"
	"strings"

	"github.com/dpopsuev/bugle/facade"
)

// Dialectic is the default CollectiveStrategy: thesis-antithesis ping-pong.
type Dialectic struct {
	MaxRounds       int    // default 3
	ConvergenceWord string // default "CONVERGED"
}

func (d *Dialectic) defaults() (int, string) {
	maxRounds := d.MaxRounds
	if maxRounds <= 0 {
		maxRounds = 3
	}
	word := d.ConvergenceWord
	if word == "" {
		word = "CONVERGED"
	}
	return maxRounds, word
}

// Orchestrate runs the dialectic debate between agents[0] (thesis) and
// agents[1] (antithesis). Returns the thesis's last response as synthesis.
func (d *Dialectic) Orchestrate(ctx context.Context, prompt string, agents []*facade.AgentHandle) (string, error) {
	if len(agents) < 2 {
		return "", fmt.Errorf("dialectic requires at least 2 agents, got %d", len(agents))
	}

	maxRounds, convergenceWord := d.defaults()
	thesis, anti := agents[0], agents[1]

	// Round 1: thesis drafts.
	thesisResp, err := thesis.Ask(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("thesis initial draft: %w", err)
	}

	for round := range maxRounds {
		// Antithesis challenges.
		antiPrompt := fmt.Sprintf(
			"Original request:\n%s\n\nThesis response (round %d):\n%s\n\n"+
				"Challenge this response. Identify flaws, missing considerations, "+
				"and alternatives. If the response adequately addresses all concerns, "+
				"respond with exactly %s.",
			prompt, round+1, thesisResp, convergenceWord,
		)
		antiResp, err := anti.Ask(ctx, antiPrompt)
		if err != nil {
			return thesisResp, nil // antithesis error — return best thesis
		}

		// Check convergence.
		if strings.Contains(antiResp, convergenceWord) {
			return thesisResp, nil
		}

		// Thesis revises.
		revisePrompt := fmt.Sprintf(
			"Original request:\n%s\n\nYour previous response:\n%s\n\n"+
				"Critique received:\n%s\n\n"+
				"Revise your response to address valid points while defending correct positions.",
			prompt, thesisResp, antiResp,
		)
		revised, err := thesis.Ask(ctx, revisePrompt)
		if err != nil {
			return thesisResp, nil // revision error — return last good thesis
		}
		thesisResp = revised
	}

	// Max rounds exhausted — thesis wins.
	return thesisResp, nil
}

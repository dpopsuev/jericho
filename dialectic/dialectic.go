// Package dialectic defines the adversarial dialectic types:
// thesis-antithesis-synthesis debate with evidence gap tracking.
//
// Edge factory code (BuildEdgeFactory, HD1-HD13 evaluators) remains in
// Origami's dialectic package because it depends on circuit.Edge and
// engine.EdgeFactory.
package dialectic

import "time"

// Config controls the adversarial dialectic circuit activation and limits.
// When the Thesis path's confidence falls below the contradiction threshold,
// the adversarial path activates for thesis-antithesis-synthesis debate.
type Config struct {
	Enabled                bool                                `json:"enabled"`
	TTL                    time.Duration                       `json:"ttl"`
	MaxTurns               int                                 `json:"max_turns"`
	MaxNegations           int                                 `json:"max_negations"`
	ContradictionThreshold float64                             `json:"contradiction_threshold"`
	GapClosureThreshold    float64                             `json:"gap_closure_threshold"`
	CMRREnabled            bool                                `json:"cmrr_enabled"`
	ContextGuard           func(map[string]any) map[string]any `json:"-"`
}

// DefaultConfig returns conservative defaults for the dialectic circuit.
func DefaultConfig() Config {
	return Config{
		Enabled:                false,
		TTL:                    10 * time.Minute,
		MaxTurns:               6,
		MaxNegations:           2,
		ContradictionThreshold: 0.85,
		GapClosureThreshold:    0.15,
	}
}

// NeedsAntithesis returns true when a Thesis path confidence falls in the
// uncertain range that triggers adversarial dialectic review.
func (c Config) NeedsAntithesis(confidence float64) bool {
	if !c.Enabled {
		return false
	}
	return confidence >= 0.50 && confidence < c.ContradictionThreshold
}

// SynthesisDecision represents the outcome of the adversarial dialectic.
type SynthesisDecision string

const (
	SynthesisAffirm     SynthesisDecision = "affirm"
	SynthesisAmend      SynthesisDecision = "amend"
	SynthesisAcquit     SynthesisDecision = "acquit"
	SynthesisRemand     SynthesisDecision = "remand"
	SynthesisUnresolved SynthesisDecision = "unresolved"
)

// EvidenceItem is a single piece of evidence with an assigned weight.
type EvidenceItem struct {
	Description string  `json:"description"`
	Source      string  `json:"source"`
	Weight      float64 `json:"weight"`
}

// ThesisChallenge is the D0 thesis-holder artifact: charged defect type with
// itemized evidence and a thesis-holder narrative.
type ThesisChallenge struct {
	ChargedDefectType string         `json:"charged_defect_type"`
	ThesisNarrative   string         `json:"thesis_narrative"`
	Evidence          []EvidenceItem `json:"evidence"`
	ConfidenceScore   float64        `json:"confidence"`
}

// Type returns the artifact type identifier.
func (t *ThesisChallenge) Type() string { return "thesis_challenge" }

// Confidence returns the thesis confidence score.
func (t *ThesisChallenge) Confidence() float64 { return t.ConfidenceScore }

// Raw returns the artifact as an untyped value.
func (t *ThesisChallenge) Raw() any { return t }

// EvidenceChallenge captures a specific challenge to an evidence item.
type EvidenceChallenge struct {
	EvidenceIndex int    `json:"evidence_index"`
	Challenge     string `json:"challenge"`
	Severity      string `json:"severity"`
}

// AntithesisResponse is the D2 antithesis-holder artifact: challenges to evidence,
// alternative hypothesis, and concession flag.
type AntithesisResponse struct {
	Challenges            []EvidenceChallenge `json:"challenges"`
	AlternativeHypothesis string              `json:"alternative_hypothesis,omitempty"`
	Concession            bool                `json:"concession"`
	ConfidenceScore       float64             `json:"confidence"`
}

// Type returns the artifact type identifier.
func (a *AntithesisResponse) Type() string { return "antithesis_response" }

// Confidence returns the antithesis confidence score.
func (a *AntithesisResponse) Confidence() float64 { return a.ConfidenceScore }

// Raw returns the artifact as an untyped value.
func (a *AntithesisResponse) Raw() any { return a }

// Round captures one round of thesis argument, antithesis rebuttal, and
// arbiter notes.
type Round struct {
	Round              int    `json:"round"`
	ThesisArgument     string `json:"thesis_argument"`
	AntithesisRebuttal string `json:"antithesis_rebuttal"`
	ArbiterNotes       string `json:"arbiter_notes"`
}

// Record is the D3 dialectic artifact: rounds of structured debate.
type Record struct {
	Rounds     []Round `json:"rounds"`
	MaxRounds  int     `json:"max_rounds"`
	Converged  bool    `json:"converged"`
	GapClosure float64 `json:"gap_closure"`
}

// Type returns the artifact type identifier.
func (d *Record) Type() string { return "dialectic_record" }

// Confidence returns zero (records carry no confidence).
func (d *Record) Confidence() float64 { return 0 }

// Raw returns the artifact as an untyped value.
func (d *Record) Raw() any { return d }

// Synthesis is the D4 final decision artifact.
type Synthesis struct {
	Decision            SynthesisDecision `json:"decision"`
	FinalClassification string            `json:"final_classification"`
	ConfidenceScore     float64           `json:"confidence"`
	Reasoning           string            `json:"reasoning"`
	NegationFeedback    *NegationFeedback `json:"negation_feedback,omitempty"`
}

// Type returns the artifact type identifier.
func (s *Synthesis) Type() string { return "synthesis" }

// Confidence returns the synthesis confidence score.
func (s *Synthesis) Confidence() float64 { return s.ConfidenceScore }

// Raw returns the artifact as an untyped value.
func (s *Synthesis) Raw() any { return s }

// NegationFeedback provides structured feedback when a case is remanded
// back to the Thesis path for reinvestigation.
type NegationFeedback struct {
	ChallengedEvidence []int    `json:"challenged_evidence"`
	AlternativeHyp     string   `json:"alternative_hypothesis"`
	SpecificQuestions  []string `json:"specific_questions"`
}

// DialecticEvidenceGap extends EvidenceGap with dialectic-specific context.
type DialecticEvidenceGap struct { //nolint:revive // kept for Origami alias compat
	EvidenceGap
	DialecticPhase string `json:"dialectic_phase,omitempty"`
}

// CMRRCheck captures shared-assumption detection results between thesis and antithesis.
// When both sides share unexamined premises, the debate may converge on a wrong answer.
// CMRR (Common-Mode Rejection Ratio) flags this: high SuspicionScore means
// shared assumptions need independent verification.
type CMRRCheck struct {
	SharedPremises []string `json:"shared_premises"`
	SuspicionScore float64  `json:"suspicion_score"`
}

// Type returns the artifact type identifier.
func (c *CMRRCheck) Type() string { return "cmrr_check" }

// Confidence returns 1.0 minus the suspicion score.
func (c *CMRRCheck) Confidence() float64 { return 1.0 - c.SuspicionScore }

// Raw returns the artifact as an untyped value.
func (c *CMRRCheck) Raw() any { return c }

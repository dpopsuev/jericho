package collective

import "testing"

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Enabled {
		t.Error("default dialectic should be disabled")
	}
	if cfg.MaxNegations != 2 {
		t.Errorf("MaxNegations = %d, want 2", cfg.MaxNegations)
	}
	if cfg.MaxTurns != 6 {
		t.Errorf("MaxTurns = %d, want 6", cfg.MaxTurns)
	}
	if cfg.ContradictionThreshold != 0.85 {
		t.Errorf("ContradictionThreshold = %f, want 0.85", cfg.ContradictionThreshold)
	}
	if cfg.GapClosureThreshold != 0.15 {
		t.Errorf("GapClosureThreshold = %f, want 0.15", cfg.GapClosureThreshold)
	}
}

func TestConfig_NeedsAntithesis(t *testing.T) {
	cfg := Config{Enabled: true, ContradictionThreshold: 0.85}

	cases := []struct {
		confidence float64
		want       bool
	}{
		{0.90, false},
		{0.85, false},
		{0.84, true},
		{0.65, true},
		{0.50, true},
		{0.49, false},
		{0.30, false},
		{1.00, false},
	}
	for _, tc := range cases {
		got := cfg.NeedsAntithesis(tc.confidence)
		if got != tc.want {
			t.Errorf("NeedsAntithesis(%f) = %v, want %v", tc.confidence, got, tc.want)
		}
	}
}

func TestConfig_NeedsAntithesis_Disabled(t *testing.T) {
	cfg := Config{Enabled: false, ContradictionThreshold: 0.85}
	if cfg.NeedsAntithesis(0.65) {
		t.Error("disabled dialectic should never activate")
	}
}

func TestThesisChallenge_ArtifactInterface(t *testing.T) {
	tc := &ThesisChallenge{
		ChargedDefectType: "product_bug",
		ConfidenceScore:   0.8,
		Evidence:          []EvidenceItem{{Description: "test", Source: "log", Weight: 0.9}},
	}
	if tc.Type() != "thesis_challenge" {
		t.Errorf("Type() = %q, want %q", tc.Type(), "thesis_challenge")
	}
	if tc.Raw() != tc {
		t.Error("Raw() should return self")
	}
}

func TestAntithesisResponse_ArtifactInterface(t *testing.T) {
	ar := &AntithesisResponse{
		Challenges:      []EvidenceChallenge{{EvidenceIndex: 0, Challenge: "weak", Severity: "high"}},
		Concession:      false,
		ConfidenceScore: 0.7,
	}
	if ar.Type() != "antithesis_response" {
		t.Errorf("Type() = %q, want %q", ar.Type(), "antithesis_response")
	}
}

func TestRecord_ArtifactInterface(t *testing.T) {
	record := &Record{
		Rounds:    []Round{{Round: 1, ThesisArgument: "t", AntithesisRebuttal: "a", ArbiterNotes: "n"}},
		MaxRounds: 3,
		Converged: false,
	}
	if record.Type() != "dialectic_record" {
		t.Errorf("Type() = %q, want %q", record.Type(), "dialectic_record")
	}
}

func TestSynthesis_ArtifactInterface(t *testing.T) {
	s := &Synthesis{
		Decision:            SynthesisAffirm,
		FinalClassification: "product_bug",
		ConfidenceScore:     0.9,
		Reasoning:           "confirmed",
	}
	if s.Type() != "synthesis" {
		t.Errorf("Type() = %q, want %q", s.Type(), "synthesis")
	}
}

func TestSynthesis_Remand(t *testing.T) {
	s := &Synthesis{
		Decision: SynthesisRemand,
		NegationFeedback: &NegationFeedback{
			ChallengedEvidence: []int{0, 2},
			AlternativeHyp:     "could be flaky",
			SpecificQuestions:  []string{"Was network stable?"},
		},
	}
	if s.Decision != SynthesisRemand {
		t.Errorf("Decision = %q, want remand", s.Decision)
	}
	if s.NegationFeedback == nil {
		t.Fatal("NegationFeedback should not be nil for remand")
	}
	if len(s.NegationFeedback.ChallengedEvidence) != 2 {
		t.Errorf("ChallengedEvidence count = %d, want 2", len(s.NegationFeedback.ChallengedEvidence))
	}
}

func TestSynthesisDecision_Constants(t *testing.T) {
	decisions := []SynthesisDecision{SynthesisAffirm, SynthesisAmend, SynthesisAcquit, SynthesisRemand, SynthesisUnresolved}
	if len(decisions) != 5 {
		t.Errorf("expected 5 synthesis decisions, got %d", len(decisions))
	}
	seen := make(map[SynthesisDecision]bool)
	for _, d := range decisions {
		if seen[d] {
			t.Errorf("duplicate decision: %s", d)
		}
		seen[d] = true
	}
}

func TestDialecticEvidenceGap(t *testing.T) {
	gap := DialecticEvidenceGap{
		EvidenceGap: EvidenceGap{
			Description:     "missing network metrics during failure window",
			Source:          "infrastructure_telemetry",
			Severity:        GapSeverityHigh,
			SuggestedAction: "collect pod-level network stats from prometheus",
		},
		DialecticPhase: "D3",
	}
	if gap.Description == "" {
		t.Error("Description should not be empty")
	}
	if gap.SuggestedAction == "" {
		t.Error("SuggestedAction should not be empty")
	}
	if gap.DialecticPhase != "D3" {
		t.Errorf("DialecticPhase = %q, want D3", gap.DialecticPhase)
	}
}

func TestCMRRCheck_ArtifactInterface(t *testing.T) {
	c := &CMRRCheck{SharedPremises: []string{"p1"}, SuspicionScore: 0.3}
	if c.Type() != "cmrr_check" {
		t.Errorf("Type() = %q, want cmrr_check", c.Type())
	}
	if c.Confidence() != 0.7 {
		t.Errorf("Confidence() = %f, want 0.7", c.Confidence())
	}
}

package collective

import "testing"

func TestGapSeverityConstants(t *testing.T) {
	severities := []GapSeverity{GapSeverityCritical, GapSeverityHigh, GapSeverityMedium, GapSeverityLow}
	if len(severities) != 4 {
		t.Errorf("expected 4 severities, got %d", len(severities))
	}
	seen := make(map[GapSeverity]bool)
	for _, s := range severities {
		if seen[s] {
			t.Errorf("duplicate severity: %s", s)
		}
		seen[s] = true
	}
}

func TestEvidenceGapBrief_ArtifactInterface(t *testing.T) {
	brief := &EvidenceGapBrief{
		CaseID:          "C01",
		FinalConfidence: 0.35,
		Gaps: []EvidenceGap{
			{Description: "no repo logs", Source: "git", Severity: GapSeverityHigh},
			{Description: "missing CI artifacts", Source: "jenkins", Severity: GapSeverityMedium},
		},
		Summary: "Insufficient evidence for confident classification",
	}
	if brief.Type() != "evidence_gap_brief" {
		t.Errorf("Type() = %q, want evidence_gap_brief", brief.Type())
	}
	if brief.Confidence() != 0.35 {
		t.Errorf("Confidence() = %f, want 0.35", brief.Confidence())
	}
	if brief.Raw() != brief {
		t.Error("Raw() should return self")
	}
	if len(brief.Gaps) != 2 {
		t.Errorf("len(Gaps) = %d, want 2", len(brief.Gaps))
	}
}

func TestDefaultGapBriefThreshold(t *testing.T) {
	th := DefaultGapBriefThreshold()
	if th.MinConfidence != 0.50 {
		t.Errorf("MinConfidence = %f, want 0.50", th.MinConfidence)
	}
	if th.MaxGaps != 10 {
		t.Errorf("MaxGaps = %d, want 10", th.MaxGaps)
	}
}

func TestGapBriefThreshold_ShouldProduceGapBrief(t *testing.T) {
	th := GapBriefThreshold{MinConfidence: 0.50}
	cases := []struct {
		confidence float64
		want       bool
	}{
		{0.49, true},
		{0.50, false},
		{0.30, true},
		{0.80, false},
		{0.00, true},
		{1.00, false},
	}
	for _, tc := range cases {
		got := th.ShouldProduceGapBrief(tc.confidence)
		if got != tc.want {
			t.Errorf("ShouldProduceGapBrief(%f) = %v, want %v", tc.confidence, got, tc.want)
		}
	}
}

func TestDialecticEvidenceGap_EmbedsEvidenceGap(t *testing.T) {
	gap := DialecticEvidenceGap{
		EvidenceGap: EvidenceGap{
			Description:     "thesis evidence was circumstantial",
			Source:          "dialectic_hearing",
			Severity:        GapSeverityHigh,
			SuggestedAction: "gather direct evidence",
		},
		DialecticPhase: "D3",
	}
	if gap.Description != "thesis evidence was circumstantial" {
		t.Errorf("embedded Description not accessible: %q", gap.Description)
	}
	if gap.DialecticPhase != "D3" {
		t.Errorf("DialecticPhase = %q, want D3", gap.DialecticPhase)
	}
}

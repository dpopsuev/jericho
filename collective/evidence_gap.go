package collective

// GapSeverity classifies the impact of a missing evidence gap.
type GapSeverity string

const (
	GapSeverityCritical GapSeverity = "critical"
	GapSeverityHigh     GapSeverity = "high"
	GapSeverityMedium   GapSeverity = "medium"
	GapSeverityLow      GapSeverity = "low"
)

// EvidenceGap represents a single missing piece of evidence that prevented
// the circuit from reaching high confidence. Shared between the Thesis path
// (evidence-gap-brief) and the Antithesis path (dialectic unresolved contradiction).
type EvidenceGap struct {
	Description     string      `json:"description"`
	Source          string      `json:"source"`
	Severity        GapSeverity `json:"severity"`
	SuggestedAction string      `json:"suggested_action,omitempty"`
}

// EvidenceGapBrief aggregates gaps from a circuit run that ended with
// low or inconclusive confidence. It answers: "I don't know because X."
type EvidenceGapBrief struct {
	CaseID          string        `json:"case_id"`
	FinalConfidence float64       `json:"final_confidence"`
	Gaps            []EvidenceGap `json:"gaps"`
	Summary         string        `json:"summary"`
}

// Type returns the artifact type identifier.
func (b *EvidenceGapBrief) Type() string { return "evidence_gap_brief" }

// Confidence returns the final confidence score.
func (b *EvidenceGapBrief) Confidence() float64 { return b.FinalConfidence }

// Raw returns the artifact as an untyped value.
func (b *EvidenceGapBrief) Raw() any { return b }

// GapBriefThreshold defines when the circuit should produce an EvidenceGapBrief
// instead of a confident classification.
type GapBriefThreshold struct {
	MinConfidence float64 `json:"min_confidence"`
	MaxGaps       int     `json:"max_gaps"`
}

// DefaultGapBriefThreshold returns conservative defaults: produce a gap brief
// when confidence is below 0.50, allowing up to 10 gaps per brief.
func DefaultGapBriefThreshold() GapBriefThreshold {
	return GapBriefThreshold{
		MinConfidence: 0.50,
		MaxGaps:       10,
	}
}

// ShouldProduceGapBrief returns true when the circuit's final confidence
// is below the threshold, indicating an evidence gap brief should be emitted.
func (th GapBriefThreshold) ShouldProduceGapBrief(confidence float64) bool {
	return confidence < th.MinConfidence
}

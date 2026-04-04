package protocol

// AndonLevel identifies a health state. Reserved levels have standard meaning.
// Custom levels use any name with a declared priority and color.
type AndonLevel string

// Reserved levels — fixed priority, fixed color, standard meaning.
const (
	AndonNominal  AndonLevel = "nominal"  // green #00FF00 — what are you doing?
	AndonDegraded AndonLevel = "degraded" // amber #FFBF00 — what limit is approaching?
	AndonFailure  AndonLevel = "failure"  // red #FF0000 — what broke?
	AndonBlocked  AndonLevel = "blocked"  // blue #0000FF — what do you need to unblock?
	AndonDead     AndonLevel = "dead"     // black #000000 — what killed you?
)

// Reserved priorities — evenly spaced, immutable.
const (
	PriorityNominal  uint8 = 0
	PriorityDegraded uint8 = 64
	PriorityFailure  uint8 = 128
	PriorityBlocked  uint8 = 160
	PriorityDead     uint8 = 255
)

// ReservedLevels maps reserved level names to their fixed priority.
var ReservedLevels = map[AndonLevel]uint8{
	AndonNominal:  PriorityNominal,
	AndonDegraded: PriorityDegraded,
	AndonFailure:  PriorityFailure,
	AndonBlocked:  PriorityBlocked,
	AndonDead:     PriorityDead,
}

// ReservedColors maps reserved level names to their IEC 60073 hex color.
var ReservedColors = map[AndonLevel]string{
	AndonNominal:  "#00FF00",
	AndonDegraded: "#FFBF00",
	AndonFailure:  "#FF0000",
	AndonBlocked:  "#0000FF",
	AndonDead:     "#000000",
}

// AndonCategory classifies what triggered the signal.
type AndonCategory string

const (
	CategoryBudget    AndonCategory = "budget"
	CategoryDeadlock  AndonCategory = "deadlock"
	CategoryLifecycle AndonCategory = "lifecycle"
	CategoryQuality   AndonCategory = "quality"
	CategorySecurity  AndonCategory = "security"
	CategoryDrift     AndonCategory = "drift"
)

// Andon is a health signal — the stack light.
type Andon struct {
	Level    AndonLevel    `json:"level"`
	Priority uint8         `json:"priority"`
	Category AndonCategory `json:"category,omitempty"`
	Message  string        `json:"message"`
	Detail   string        `json:"detail,omitempty"`
}

// AndonLevelDef declares a level in the server's vocabulary.
// Returned in capabilities.andon_levels on start response.
type AndonLevelDef struct {
	Name     AndonLevel `json:"name"`
	Priority uint8      `json:"priority"`
	Color    string     `json:"color"`
}

// DefaultVocabulary returns the 5 reserved andon level definitions.
func DefaultVocabulary() []AndonLevelDef {
	return []AndonLevelDef{
		{AndonNominal, PriorityNominal, "#00FF00"},
		{AndonDegraded, PriorityDegraded, "#FFBF00"},
		{AndonFailure, PriorityFailure, "#FF0000"},
		{AndonBlocked, PriorityBlocked, "#0000FF"},
		{AndonDead, PriorityDead, "#000000"},
	}
}

// Worse returns true if a has higher priority (more severe) than b.
func Worse(a, b uint8) bool {
	return a > b
}

// WorstPriority returns the highest priority from a set.
func WorstPriority(priorities ...uint8) uint8 {
	var worst uint8
	for _, p := range priorities {
		if p > worst {
			worst = p
		}
	}
	return worst
}

// PriorityOf returns the priority for a level. For reserved levels,
// returns the fixed priority. For custom levels, the caller must
// provide the priority from the vocabulary.
func PriorityOf(level AndonLevel) (uint8, bool) {
	p, ok := ReservedLevels[level]
	return p, ok
}

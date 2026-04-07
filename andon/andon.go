// Package andon provides health signal types for the agent stack.
// Five reserved levels (nominal/degraded/failure/blocked/dead) with
// IEC 60073 colors and fixed priorities. Workers report andon level
// on every push. AndonDead aborts the worker.
package andon

// AndonLevel identifies a health state. Reserved levels have standard meaning.
// Custom levels use any name with a declared priority and color.
type AndonLevel string

// Reserved levels — fixed priority, fixed color, standard meaning.
const (
	Nominal  AndonLevel = "nominal"  // green #00FF00 — what are you doing?
	Degraded AndonLevel = "degraded" // amber #FFBF00 — what limit is approaching?
	Failure  AndonLevel = "failure"  // red #FF0000 — what broke?
	Blocked  AndonLevel = "blocked"  // blue #0000FF — what do you need to unblock?
	Dead     AndonLevel = "dead"     // black #000000 — what killed you?
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
	Nominal:  PriorityNominal,
	Degraded: PriorityDegraded,
	Failure:  PriorityFailure,
	Blocked:  PriorityBlocked,
	Dead:     PriorityDead,
}

// ReservedColors maps reserved level names to their IEC 60073 hex color.
var ReservedColors = map[AndonLevel]string{
	Nominal:  "#00FF00",
	Degraded: "#FFBF00",
	Failure:  "#FF0000",
	Blocked:  "#0000FF",
	Dead:     "#000000",
}

// Category classifies what triggered the signal.
type Category string

const (
	CategoryBudget    Category = "budget"
	CategoryDeadlock  Category = "deadlock"
	CategoryLifecycle Category = "lifecycle"
	CategoryQuality   Category = "quality"
	CategorySecurity  Category = "security"
	CategoryDrift     Category = "drift"
)

// Andon is a health signal — the stack light.
type Andon struct {
	Level    AndonLevel `json:"level"`
	Priority uint8      `json:"priority"`
	Category Category   `json:"category,omitempty"`
	Message  string     `json:"message"`
	Detail   string     `json:"detail,omitempty"`
}

// LevelDef declares a level in the server's vocabulary.
// Returned in capabilities.andon_levels on start response.
type LevelDef struct {
	Name     AndonLevel `json:"name"`
	Priority uint8      `json:"priority"`
	Color    string     `json:"color"`
}

// DefaultVocabulary returns the 5 reserved andon level definitions.
func DefaultVocabulary() []LevelDef {
	return []LevelDef{
		{Nominal, PriorityNominal, "#00FF00"},
		{Degraded, PriorityDegraded, "#FFBF00"},
		{Failure, PriorityFailure, "#FF0000"},
		{Blocked, PriorityBlocked, "#0000FF"},
		{Dead, PriorityDead, "#000000"},
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

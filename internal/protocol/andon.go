package protocol

import "github.com/dpopsuev/troupe/andon"

// Type aliases — internal consumers continue to work unchanged.
// New code should import github.com/dpopsuev/troupe/andon directly.
type (
	AndonLevel    = andon.AndonLevel
	AndonCategory = andon.Category
	Andon         = andon.Andon
	AndonLevelDef = andon.LevelDef
)

// Re-export constants.
const (
	AndonNominal  = andon.Nominal
	AndonDegraded = andon.Degraded
	AndonFailure  = andon.Failure
	AndonBlocked  = andon.Blocked
	AndonDead     = andon.Dead

	PriorityNominal  = andon.PriorityNominal
	PriorityDegraded = andon.PriorityDegraded
	PriorityFailure  = andon.PriorityFailure
	PriorityBlocked  = andon.PriorityBlocked
	PriorityDead     = andon.PriorityDead

	CategoryBudget    = andon.CategoryBudget
	CategoryDeadlock  = andon.CategoryDeadlock
	CategoryLifecycle = andon.CategoryLifecycle
	CategoryQuality   = andon.CategoryQuality
	CategorySecurity  = andon.CategorySecurity
	CategoryDrift     = andon.CategoryDrift
)

// Re-export functions.
var (
	DefaultVocabulary = andon.DefaultVocabulary
	Worse             = andon.Worse
	WorstPriority     = andon.WorstPriority
	PriorityOf        = andon.PriorityOf
	ReservedLevels    = andon.ReservedLevels
	ReservedColors    = andon.ReservedColors
)

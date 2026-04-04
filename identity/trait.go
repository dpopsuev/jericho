// Package trait provides behavioral trait primitives for agents.
// Traits are quantified behavioral preferences (0.0-1.0) backed by
// real LLM benchmarks. Consumers attach trait.Set as an ECS component
// and map values to operational config (budget, timeout, loop limits).
//
// Stubs in v0.2.0 — full inference pipeline deferred to Arsenal (JRC-NED-2).
package identity

import "github.com/dpopsuev/troupe/world"

// TraitType is the ComponentType for a trait Set.
const TraitType world.ComponentType = "trait"

// Trait is a single behavioral trait with a value between 0.0 and 1.0.
type Trait struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

// Set is a collection of traits, attached as an ECS component.
type Set []Trait

// ComponentType implements world.Component.
func (Set) ComponentType() world.ComponentType { return TraitType }

// Get returns the value for a named trait, or 0.0 if not present.
func (s Set) Get(name string) float64 {
	for _, t := range s {
		if t.Name == name {
			return t.Value
		}
	}
	return 0.0
}

// Has returns true if the named trait is present.
func (s Set) Has(name string) bool {
	for _, t := range s {
		if t.Name == name {
			return true
		}
	}
	return false
}

// Default trait names — the 8-trait vocabulary from JRC-SPC-4.
const (
	Speed      = "speed"      // fast scan vs slow analysis
	Reasoning  = "reasoning"  // multi-step logical chains
	Rigor      = "rigor"      // demands evidence, rejects uncertainty
	Coding     = "coding"     // read, write, debug code
	Discipline = "discipline" // follows instructions exactly
	ToolUse    = "tooluse"    // chains tool calls autonomously
	Discourse  = "discourse"  // pushes back, challenges, brainstorms
	Visual     = "visual"     // reads and creates visual/spatial content
)

// DefaultVocabulary returns the 8 default trait names.
func DefaultVocabulary() []string {
	return []string{Speed, Reasoning, Rigor, Coding, Discipline, ToolUse, Discourse, Visual}
}

// TraitVector holds normalized trait scores (0.0-1.0) for the 8-trait vocabulary.
type TraitVector struct {
	Speed      float64 `yaml:"speed"      json:"speed"`
	Reasoning  float64 `yaml:"reasoning"  json:"reasoning"`
	Rigor      float64 `yaml:"rigor"      json:"rigor"`
	Coding     float64 `yaml:"coding"     json:"coding"`
	Discipline float64 `yaml:"discipline" json:"discipline"`
	ToolUse    float64 `yaml:"tooluse"    json:"tooluse"`
	Discourse  float64 `yaml:"discourse"  json:"discourse"`
	Visual     float64 `yaml:"visual"     json:"visual"`
}

// Score returns the dot product of this vector with a weight vector.
func (v TraitVector) Score(w TraitVector) float64 {
	return v.Speed*w.Speed +
		v.Reasoning*w.Reasoning +
		v.Rigor*w.Rigor +
		v.Coding*w.Coding +
		v.Discipline*w.Discipline +
		v.ToolUse*w.ToolUse +
		v.Discourse*w.Discourse +
		v.Visual*w.Visual
}

// MeetsMinimum returns true if every non-zero field in floor is <= the
// corresponding field in v.
func (v TraitVector) MeetsMinimum(floor TraitVector) bool {
	return (floor.Speed == 0 || v.Speed >= floor.Speed) &&
		(floor.Reasoning == 0 || v.Reasoning >= floor.Reasoning) &&
		(floor.Rigor == 0 || v.Rigor >= floor.Rigor) &&
		(floor.Coding == 0 || v.Coding >= floor.Coding) &&
		(floor.Discipline == 0 || v.Discipline >= floor.Discipline) &&
		(floor.ToolUse == 0 || v.ToolUse >= floor.ToolUse) &&
		(floor.Discourse == 0 || v.Discourse >= floor.Discourse) &&
		(floor.Visual == 0 || v.Visual >= floor.Visual)
}

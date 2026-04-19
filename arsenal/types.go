// Package arsenal provides an embedded YAML catalog of agent models with
// trait-scored selection. Three concepts: Source (access path), Provider
// (model creator), Model (specific model). Consumer preferences are weight
// vectors over normalized traits. Declarative selection resolves down to
// imperative map lookup.
//
// Zero Jericho imports — stdlib + yaml.v3 only.
package arsenal

import "errors"

// Sentinel errors.
var (
	ErrNotFound     = errors.New("arsenal: model+source not found")
	ErrNoCandidate  = errors.New("arsenal: no model matches preferences")
	ErrBadPin       = errors.New("arsenal: unknown snapshot pin")
	ErrEmptyCatalog = errors.New("arsenal: no snapshots found in embedded catalog")
)

// CostEntry holds per-million-token pricing.
type CostEntry struct {
	InputPerM  float64 `yaml:"input_per_m"  json:"input_per_m"`
	OutputPerM float64 `yaml:"output_per_m" json:"output_per_m"`
}

// ModelEntry is a single model in a provider manifest.
type ModelEntry struct {
	ID         string             `yaml:"id"`
	Provider   string             `yaml:"provider"`
	Context    int                `yaml:"context"`
	Benchmarks map[string]float64 `yaml:"benchmarks"`
	Cost       CostEntry          `yaml:"cost"`
	Traits     TraitVector        // computed at load time, not in YAML
	Available  bool               // live discovery result (true = API confirmed, false = unavailable/unprobed)
}

// SourceModifiers describe how a source affects effective performance.
type SourceModifiers struct {
	ContextCap    int     `yaml:"context_cap,omitempty"`
	TokenOverhead float64 `yaml:"token_overhead,omitempty"`
	Pipeline      string  `yaml:"pipeline,omitempty"` // "direct" or "multi-model"
}

// SourceEntry is a single source in the catalog.
type SourceEntry struct {
	Source   string          `yaml:"source"`
	Kind     string          `yaml:"kind"` // cli, api
	Binary   string          `yaml:"binary,omitempty"`
	EnvKey   string          `yaml:"env_key,omitempty"`
	Provider string          `yaml:"provider,omitempty"`
	Mods     SourceModifiers `yaml:"modifiers,omitempty"`
	Models   []ModelEntry    `yaml:"models,omitempty"` // source's own models
	Access   []string        `yaml:"access,omitempty"` // provider models it can reach
}

// ResolvedAgent is the output of Pick() and Select() — a model resolved
// through a specific source with effective modifiers applied.
type ResolvedAgent struct {
	Model      string
	Provider   string
	Source     string
	Traits     TraitVector
	EffContext int
	Overhead   float64
	Pipeline   string
	Cost       CostEntry
}

// Preferences is the consumer's weight vector + filters.
type Preferences struct {
	Weights   TraitVector
	MinTraits TraitVector
	Sources   Filter
	Providers Filter
	Models    Filter
	MaxCost   float64 // max input_per_m, 0 = no limit
}

// Filter is an allow/block list for string matching.
type Filter struct {
	Allow []string
	Block []string
}

// matches returns true if the value passes the filter.
func (f Filter) matches(value string) bool {
	if len(f.Block) > 0 {
		for _, b := range f.Block {
			if b == value {
				return false
			}
		}
	}
	if len(f.Allow) > 0 {
		for _, a := range f.Allow {
			if a == value {
				return true
			}
		}
		return false // allow list is non-empty but value not in it
	}
	return true // no allow list = allow all
}

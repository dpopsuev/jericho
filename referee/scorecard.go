package referee

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// ErrNameRequired is returned when a scorecard has no name.
var ErrNameRequired = errors.New("scorecard: name is required")

// Scorecard defines how to score an agent run.
type Scorecard struct {
	Name               string          `yaml:"name"`
	Threshold          int             `yaml:"threshold"`            // score >= threshold = pass
	DefaultWeight      int             `yaml:"default_weight"`       // weight for listed events without explicit weight
	UnknownEventWeight int             `yaml:"unknown_event_weight"` // weight for unlisted events
	Rules              []ScorecardRule `yaml:"rules"`
}

// ScorecardRule is a single scoring rule.
type ScorecardRule struct {
	On        string `yaml:"on"`                  // event kind to match
	Weight    int    `yaml:"weight"`              // score delta when matched
	Condition string `yaml:"condition,omitempty"` // optional condition (future: expressions)
}

// ParseScorecard parses a scorecard from YAML bytes.
func ParseScorecard(data []byte) (Scorecard, error) {
	var sc Scorecard
	if err := yaml.Unmarshal(data, &sc); err != nil {
		return sc, fmt.Errorf("parse scorecard: %w", err)
	}
	if sc.Name == "" {
		return sc, fmt.Errorf("%w", ErrNameRequired)
	}
	return sc, nil
}

// ruleIndex builds a lookup map from event kind to rule.
func (sc *Scorecard) ruleIndex() map[string]*ScorecardRule {
	idx := make(map[string]*ScorecardRule, len(sc.Rules))
	for i := range sc.Rules {
		idx[sc.Rules[i].On] = &sc.Rules[i]
	}
	return idx
}

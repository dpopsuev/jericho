package arsenal

import "fmt"

const unknownValue = "unknown"

// ModelIdentity records which foundation LLM model ("ghost") is behind
// a backend ("shell"). Wrapper records the hosting environment (e.g.
// Cursor, Azure) between the caller and the foundation model.
type ModelIdentity struct { //nolint:revive // kept for Origami alias compat
	ModelName string `json:"model_name"`
	Provider  string `json:"provider"`
	Version   string `json:"version,omitempty"`
	Wrapper   string `json:"wrapper,omitempty"`
	Raw       string `json:"raw,omitempty"`
}

// String returns "model@version/provider (via wrapper)".
func (m ModelIdentity) String() string {
	name := m.ModelName
	if name == "" {
		name = unknownValue
	}
	prov := m.Provider
	if prov == "" {
		prov = unknownValue
	}

	var s string
	if m.Version != "" {
		s = fmt.Sprintf("%s@%s/%s", name, m.Version, prov)
	} else {
		s = fmt.Sprintf("%s/%s", name, prov)
	}

	if m.Wrapper != "" {
		s += fmt.Sprintf(" (via %s)", m.Wrapper)
	}
	return s
}

// Tag returns a bracket-wrapped model name for log lines.
func (m ModelIdentity) Tag() string { //nolint:gocritic // value receiver for Origami compat
	name := m.ModelName
	if name == "" {
		name = unknownValue
	}
	if len(name) > 20 {
		name = name[:20]
	}
	return fmt.Sprintf("[%s]", name)
}

// CostProfile describes the resource cost of using an agent.
type CostProfile struct {
	TokensPerStep int     `json:"tokens_per_step"`
	LatencyMs     int     `json:"latency_ms"`
	CostPerToken  float64 `json:"cost_per_token"`
}

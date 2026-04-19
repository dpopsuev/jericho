package signal

// Signal represents a single event on the agent message bus.
type Signal struct {
	Timestamp string            `json:"ts"`
	Event     string            `json:"event"`
	Agent     string            `json:"agent"`
	CaseID    string            `json:"case_id,omitempty"`
	Step      string            `json:"step,omitempty"`
	Meta      map[string]string `json:"meta,omitempty"`
}

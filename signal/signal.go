package signal

// Performative classifies the intent of a signal in agent-to-agent
// communication, following FIPA-ACL speech-act semantics.
type Performative string

const (
	// Inform notifies peers of a fact or observation.
	Inform Performative = "inform"
	// Request asks a peer to perform an action.
	Request Performative = "request"
	// Confirm acknowledges a previous request was fulfilled.
	Confirm Performative = "confirm"
	// Refuse declines a previous request.
	Refuse Performative = "refuse"
	// Handoff transfers responsibility to another agent.
	Handoff Performative = "handoff"
	// Directive issues a command from a supervisor.
	Directive Performative = "directive"
)

// Signal represents a single event on the agent message bus.
type Signal struct {
	Timestamp    string            `json:"ts"`
	Event        string            `json:"event"`
	Agent        string            `json:"agent"`
	CaseID       string            `json:"case_id,omitempty"`
	Step         string            `json:"step,omitempty"`
	Meta         map[string]string `json:"meta,omitempty"`
	Performative Performative      `json:"performative,omitempty"`
}

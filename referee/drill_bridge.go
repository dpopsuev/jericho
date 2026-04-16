package referee

import "github.com/dpopsuev/troupe/signal"

// EmitWorkspaceResult emits events for a workspace check result.
// Call after a drill workspace check completes. The Referee will
// score these events like any other.
func EmitWorkspaceResult(log signal.EventLog, pass bool, score float64, errors []string) {
	kind := "workspace.check.pass"
	if !pass {
		kind = "workspace.check.fail"
	}
	log.Emit(signal.Event{
		Source: "referee",
		Kind:   kind,
		Data: map[string]any{
			"pass":   pass,
			"score":  score,
			"errors": errors,
		},
	})
}

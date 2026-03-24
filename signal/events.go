package signal

// Signal event name constants for cross-package bus.Emit calls.
// Using constants prevents typo-induced mismatches between emitters
// and listeners (e.g., Supervisor.Process).
const (
	EventWorkerStarted  = "worker_started"
	EventWorkerStopped  = "worker_stopped"
	EventWorkerStart    = "start"
	EventWorkerDone     = "done"
	EventWorkerError    = "error"
	EventShouldStop     = "should_stop"
	EventBudgetUpdate   = "budget_update"
	EventZoneShift      = "zone_shift"
	EventDispatchRouted = "dispatch_routed"
	EventHookExecuted   = "hook_executed"
	EventVetoApplied    = "veto_applied"
)

// Signal meta key constants used in bus.Emit meta maps and read by
// Supervisor.Process and other signal consumers.
const (
	MetaKeyWorkerID       = "worker_id"
	MetaKeyError          = "error"
	MetaKeyUsed           = "used"
	MetaKeyFromZone       = "from_zone"
	MetaKeyToZone         = "to_zone"
	MetaKeyMode           = "mode"
	MetaKeyBytes          = "bytes"
	MetaKeyInFlight       = "in_flight"
	MetaKeyVia            = "via"
	MetaKeyPromptPath     = "prompt_path"
	MetaKeyDispatchReason = "dispatch_reason"
	MetaKeyQueueDepth     = "queue_depth"
	MetaKeyHookName       = "hook_name"
	MetaKeyHookPhase      = "hook_phase"
)

// Agent name constants used as the agent field in Signal.
const (
	AgentWorker     = "worker"
	AgentSupervisor = "supervisor"
	AgentServer     = "server"
	AgentMediator   = "mediator"
)

package protocol

// SubmitStatus is the outcome of a submitted work item. Closed set.
type SubmitStatus string

const (
	StatusOk       SubmitStatus = "ok"       // Completed successfully (default).
	StatusBlocked  SubmitStatus = "blocked"  // Worker cannot complete, needs human.
	StatusResolved SubmitStatus = "resolved" // Human resolved a blocked item.
	StatusCanceled SubmitStatus = "canceled"
	StatusError    SubmitStatus = "error" // Unrecoverable worker error.
)

// ValidStatuses contains all recognized submit status values.
var ValidStatuses = map[SubmitStatus]bool{
	StatusOk:       true,
	StatusBlocked:  true,
	StatusResolved: true,
	StatusCanceled: true,
	StatusError:    true,
}

// exit.go — exit status types for agent process supervision.
package pool

import (
	"time"

	"github.com/dpopsuev/bugle/world"
)

// ExitCode describes why an agent terminated.
type ExitCode int

const (
	ExitSuccess ExitCode = 0 // completed normally
	ExitError   ExitCode = 1 // runtime error
	ExitBudget  ExitCode = 2 // budget ceiling exceeded
	ExitTimeout ExitCode = 3 // context deadline exceeded
)

// ExitStatus is returned by Wait/WaitAny when a zombie agent is reaped.
type ExitStatus struct {
	AgentID  world.EntityID
	ParentID world.EntityID
	Role     string
	Code     ExitCode
	Error    string
	Duration time.Duration
}

// TreeNode represents an agent and its children in the process tree.
type TreeNode struct {
	ID       world.EntityID
	ParentID world.EntityID
	Role     string
	State    string // "running", "zombie", "idle"
	Children []*TreeNode
}

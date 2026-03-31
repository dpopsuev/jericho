package bugle

import "encoding/json"

// --- Start ---

// StartRequest initiates a new session.
type StartRequest struct {
	Action    Action         `json:"action"`
	SessionID string         `json:"session_id,omitempty"`
	Parallel  int            `json:"parallel,omitempty"`
	Force     bool           `json:"force,omitempty"`
	Extra     map[string]any `json:"extra,omitempty"`
	Agent     string         `json:"agent,omitempty"`
	Workers   int            `json:"workers,omitempty"`
	Auth      *AuthToken     `json:"auth,omitempty"`
}

// StartResponse is returned after session creation.
type StartResponse struct {
	SessionID    string        `json:"session_id"`
	TotalItems   int           `json:"total_items"`
	Status       string        `json:"status"`
	Capabilities *Capabilities `json:"capabilities,omitempty"`
}

// --- Pull ---

// PullRequest pulls the next available work item.
type PullRequest struct {
	Action    Action     `json:"action"`
	SessionID string     `json:"session_id"`
	WorkerID  string     `json:"worker_id,omitempty"`
	TimeoutMs int        `json:"timeout_ms,omitempty"`
	Role      string     `json:"role,omitempty"` // "reviewer" for HITL blocked items
	Auth      *AuthToken `json:"auth,omitempty"`
}

// PullResponse returns a work item or signals completion.
type PullResponse struct {
	Done            bool             `json:"done"`
	Available       bool             `json:"available"`
	Item            string           `json:"item,omitempty"`
	PromptContent   string           `json:"prompt_content,omitempty"`
	DispatchID      int64            `json:"dispatch_id,omitempty"`
	Andon           AndonLevel       `json:"andon,omitempty"` // session-level severity
	BudgetRemaining *BudgetRemaining `json:"budget_remaining,omitempty"`
}

// --- Push ---

// PushRequest returns the result for a dispatched work item.
type PushRequest struct {
	Action     Action          `json:"action"`
	SessionID  string          `json:"session_id"`
	WorkerID   string          `json:"worker_id,omitempty"`
	DispatchID int64           `json:"dispatch_id"`
	Item       string          `json:"item"`
	Fields     json.RawMessage `json:"fields"`
	Status     SubmitStatus    `json:"status,omitempty"` // default: ok
	Andon      *Andon          `json:"andon,omitempty"`
	Budget     *BudgetActual   `json:"budget_actual,omitempty"`
	Auth       *AuthToken      `json:"auth,omitempty"`
}

// PushResponse acknowledges a submission.
type PushResponse struct {
	OK bool `json:"ok"`
}

// --- Cancel ---

// CancelRequest cancels a session or individual dispatch.
type CancelRequest struct {
	Action     Action     `json:"action"`
	SessionID  string     `json:"session_id"`
	DispatchID int64      `json:"dispatch_id,omitempty"` // 0 = cancel entire session
	Reason     string     `json:"reason,omitempty"`
	Auth       *AuthToken `json:"auth,omitempty"`
}

// CancelResponse acknowledges cancellation.
type CancelResponse struct {
	OK       bool `json:"ok"`
	Canceled int  `json:"canceled"`
}

// --- Status ---

// StatusRequest queries session state.
type StatusRequest struct {
	Action    Action     `json:"action"`
	SessionID string     `json:"session_id"`
	Auth      *AuthToken `json:"auth,omitempty"`
}

// StatusResponse returns aggregated session state.
type StatusResponse struct {
	SessionID string         `json:"session_id"`
	Progress  Progress       `json:"progress"`
	Andon     *AndonSummary  `json:"andon,omitempty"`
	Budget    *BudgetSummary `json:"budget,omitempty"`
	Cordons   []Cordon       `json:"cordons,omitempty"`
}

// AndonSummary is the aggregated andon in a status response.
type AndonSummary struct {
	Level         AndonLevel            `json:"level"`
	Priority      uint8                 `json:"priority"`
	WorstCategory AndonCategory         `json:"worst_category,omitempty"`
	PerWorker     map[string]AndonLevel `json:"per_worker,omitempty"`
}

// --- Cordon ---

// CordonRequest blocks work matching a scope pattern.
type CordonRequest struct {
	Action    Action     `json:"action"`
	SessionID string     `json:"session_id"`
	Scope     []string   `json:"scope"`
	Reason    string     `json:"reason,omitempty"`
	Auth      *AuthToken `json:"auth,omitempty"`
}

// Cordon represents an active scope block.
type Cordon struct {
	Scope  []string `json:"scope"`
	Reason string   `json:"reason,omitempty"`
}

// --- Auth ---

// AuthToken is the protocol-level identity token.
type AuthToken struct {
	Token string `json:"token"`
}

// --- Pull Meta (for worker callbacks) ---

// PullMeta carries protocol metadata from a pull response for worker callbacks.
type PullMeta struct {
	Andon           AndonLevel       `json:"andon,omitempty"`
	BudgetRemaining *BudgetRemaining `json:"budget_remaining,omitempty"`
}

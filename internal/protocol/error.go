package protocol

import "errors"

// ErrorCode is a protocol-level error identifier. Snake_case strings.
type ErrorCode string

const (
	ErrCodeNoActiveSession  ErrorCode = "no_active_session"
	ErrCodeInvalidAction    ErrorCode = "invalid_action"
	ErrCodeDispatchNotFound ErrorCode = "dispatch_not_found"
	ErrCodeSessionConflict  ErrorCode = "session_conflict"
	ErrCodeInvalidStatus    ErrorCode = "invalid_status"
	ErrCodeBudgetExceeded   ErrorCode = "budget_exceeded"
)

// ProtocolError is a structured error returned by bugle servers.
type ProtocolError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

func (e *ProtocolError) Error() string { return string(e.Code) + ": " + e.Message }

// Sentinel errors for common protocol failures.
var (
	ErrNoActiveSession  = errors.New("bugle: no active session")
	ErrInvalidAction    = errors.New("bugle: invalid action")
	ErrDispatchNotFound = errors.New("bugle: dispatch not found")
	ErrSessionConflict  = errors.New("bugle: session already exists")
	ErrInvalidStatus    = errors.New("bugle: invalid submit status")
	ErrBudgetExceeded   = errors.New("bugle: budget exceeded")
)

package protocol

import (
	"context"
	"log/slog"
)

// Audit log key constants.
const (
	auditKeyAction  = "action"
	auditKeySubject = "subject"
	auditKeyResult  = "result"
	auditKeySession = "session_id"
	auditKeyWorker  = "worker_id"
)

// AuditLog emits a structured audit log entry for a protocol operation.
func AuditLog(ctx context.Context, action Action, identity Identity, result string) {
	slog.InfoContext(ctx, "bugle.audit",
		slog.String(auditKeyAction, string(action)),
		slog.String(auditKeySubject, identity.Subject),
		slog.String(auditKeyResult, result),
	)
}

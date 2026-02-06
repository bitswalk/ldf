package common

import (
	"github.com/bitswalk/ldf/src/common/logs"
	"github.com/gin-gonic/gin"
)

var auditLogger = logs.NewDefault()

// SetAuditLogger sets the logger used for audit events.
func SetAuditLogger(l *logs.Logger) {
	if l != nil {
		auditLogger = l
	}
}

// AuditEvent represents a security-relevant event for audit logging.
type AuditEvent struct {
	// Action identifies the operation (e.g., "auth.login", "distribution.create").
	Action string
	// UserID is the authenticated user's ID (empty for unauthenticated requests).
	UserID string
	// UserName is the authenticated user's name.
	UserName string
	// Resource identifies the target (e.g., "distribution:abc-123", "setting:server.port").
	Resource string
	// ClientIP is the client's IP address.
	ClientIP string
	// Detail provides optional extra context.
	Detail string
	// Success indicates whether the operation succeeded.
	Success bool
}

// AuditLog emits a structured audit log entry from a gin request context.
// Log entries include audit=true for easy filtering.
func AuditLog(c *gin.Context, event AuditEvent) {
	status := "success"
	if !event.Success {
		status = "failure"
	}

	clientIP := event.ClientIP
	if clientIP == "" && c != nil {
		clientIP = c.ClientIP()
	}

	args := []any{
		"audit", true,
		"action", event.Action,
		"status", status,
		"client_ip", clientIP,
	}

	if event.UserID != "" {
		args = append(args, "user_id", event.UserID, "user_name", event.UserName)
	}
	if event.Resource != "" {
		args = append(args, "resource", event.Resource)
	}
	if event.Detail != "" {
		args = append(args, "detail", event.Detail)
	}

	auditLogger.Info("audit", args...)
}

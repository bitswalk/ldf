package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuditLog_DoesNotPanic(t *testing.T) {
	// Ensure AuditLog doesn't panic with a valid gin context
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	c.Request.RemoteAddr = "192.168.1.1:12345"

	// Should not panic
	AuditLog(c, AuditEvent{
		Action:   "auth.login",
		UserID:   "user-123",
		UserName: "testuser",
		Resource: "auth:login",
		Success:  true,
	})

	AuditLog(c, AuditEvent{
		Action:  "auth.login",
		Detail:  "invalid password",
		Success: false,
	})
}

func TestAuditLog_NilContext(t *testing.T) {
	// Should not panic with nil context when ClientIP is provided
	AuditLog(nil, AuditEvent{
		Action:   "auth.login",
		ClientIP: "10.0.0.1",
		Success:  true,
	})
}

func TestAuditEvent_Fields(t *testing.T) {
	event := AuditEvent{
		Action:   "distribution.create",
		UserID:   "user-456",
		UserName: "admin",
		Resource: "distribution:abc-123",
		ClientIP: "10.0.0.1",
		Detail:   "test detail",
		Success:  true,
	}

	if event.Action != "distribution.create" {
		t.Fatalf("Action = %q, want %q", event.Action, "distribution.create")
	}
	if event.UserID != "user-456" {
		t.Fatalf("UserID = %q, want %q", event.UserID, "user-456")
	}
	if !event.Success {
		t.Fatal("Success should be true")
	}
}

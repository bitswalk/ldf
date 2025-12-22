package tests

import (
	stderrors "errors"
	"net/http"
	"testing"

	"github.com/bitswalk/ldf/src/common/errors"
)

// =============================================================================
// Error Creation Tests
// =============================================================================

func TestError_New(t *testing.T) {
	err := errors.New(errors.DomainAuth, "test_code", http.StatusUnauthorized, "test message")

	if err.Domain != errors.DomainAuth {
		t.Fatalf("expected domain %s, got %s", errors.DomainAuth, err.Domain)
	}
	if err.Code != "test_code" {
		t.Fatalf("expected code test_code, got %s", err.Code)
	}
	if err.HTTPStatus != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, err.HTTPStatus)
	}
	if err.Message != "test message" {
		t.Fatalf("expected message 'test message', got %s", err.Message)
	}
}

func TestError_Wrap(t *testing.T) {
	cause := stderrors.New("underlying error")
	err := errors.Wrap(cause, errors.DomainDatabase, "query_failed", http.StatusInternalServerError, "query failed")

	if err.Unwrap() != cause {
		t.Fatal("expected wrapped error to be returned by Unwrap")
	}

	// Check error string includes cause
	errStr := err.Error()
	if errStr != "database.query_failed: query failed: underlying error" {
		t.Fatalf("unexpected error string: %s", errStr)
	}
}

// =============================================================================
// Error Methods Tests
// =============================================================================

func TestError_WithCause(t *testing.T) {
	original := errors.ErrUserNotFound
	cause := stderrors.New("db connection failed")

	wrapped := original.WithCause(cause)

	// Original should be unchanged
	if original.Unwrap() != nil {
		t.Fatal("original error should not have cause")
	}

	// Wrapped should have cause
	if wrapped.Unwrap() != cause {
		t.Fatal("wrapped error should have cause")
	}

	// Should maintain same domain/code
	if wrapped.Domain != original.Domain || wrapped.Code != original.Code {
		t.Fatal("wrapped error should maintain domain and code")
	}
}

func TestError_WithMessage(t *testing.T) {
	original := errors.ErrUserNotFound
	custom := original.WithMessage("User john not found")

	if custom.Message != "User john not found" {
		t.Fatalf("expected custom message, got %s", custom.Message)
	}

	// Original should be unchanged
	if original.Message == custom.Message {
		t.Fatal("original message should not be changed")
	}
}

func TestError_WithMessagef(t *testing.T) {
	original := errors.ErrUserNotFound
	custom := original.WithMessagef("User %s not found in %s", "john", "database")

	expected := "User john not found in database"
	if custom.Message != expected {
		t.Fatalf("expected message '%s', got '%s'", expected, custom.Message)
	}
}

// =============================================================================
// Error Interface Tests
// =============================================================================

func TestError_ErrorInterface(t *testing.T) {
	err := errors.ErrUserNotFound

	// Should implement error interface
	var _ error = err

	// Error string should include domain and code
	errStr := err.Error()
	if errStr == "" {
		t.Fatal("error string should not be empty")
	}
}

func TestError_Is(t *testing.T) {
	// Same error should match
	if !errors.Is(errors.ErrUserNotFound, errors.ErrUserNotFound) {
		t.Fatal("same error should match with Is")
	}

	// Wrapped error should match original
	wrapped := errors.ErrUserNotFound.WithCause(stderrors.New("cause"))
	if !errors.Is(wrapped, errors.ErrUserNotFound) {
		t.Fatal("wrapped error should match original with Is")
	}

	// Different errors should not match
	if errors.Is(errors.ErrUserNotFound, errors.ErrRoleNotFound) {
		t.Fatal("different errors should not match")
	}
}

func TestError_As(t *testing.T) {
	err := errors.ErrUserNotFound.WithCause(stderrors.New("cause"))

	var target *errors.Error
	if !errors.As(err, &target) {
		t.Fatal("As should find *Error in chain")
	}

	if target.Code != errors.ErrUserNotFound.Code {
		t.Fatal("As should extract correct error")
	}
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestGetHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{"user not found", errors.ErrUserNotFound, http.StatusNotFound},
		{"unauthorized", errors.ErrInvalidCredentials, http.StatusUnauthorized},
		{"forbidden", errors.ErrSystemRoleModification, http.StatusForbidden},
		{"conflict", errors.ErrUserAlreadyExists, http.StatusConflict},
		{"internal", errors.ErrInternal, http.StatusInternalServerError},
		{"standard error", stderrors.New("standard"), http.StatusInternalServerError},
		{"nil error", nil, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := errors.GetHTTPStatus(tt.err)
			if status != tt.expected {
				t.Fatalf("expected status %d, got %d", tt.expected, status)
			}
		})
	}
}

func TestGetCode(t *testing.T) {
	code := errors.GetCode(errors.ErrUserNotFound)
	if code != errors.CodeNotFound {
		t.Fatalf("expected code %s, got %s", errors.CodeNotFound, code)
	}

	// Standard error should return empty code
	code = errors.GetCode(stderrors.New("standard"))
	if code != "" {
		t.Fatalf("expected empty code for standard error, got %s", code)
	}
}

func TestGetDomain(t *testing.T) {
	domain := errors.GetDomain(errors.ErrUserNotFound)
	if domain != errors.DomainUser {
		t.Fatalf("expected domain %s, got %s", errors.DomainUser, domain)
	}

	// Standard error should return empty domain
	domain = errors.GetDomain(stderrors.New("standard"))
	if domain != "" {
		t.Fatalf("expected empty domain for standard error, got %s", domain)
	}
}

// =============================================================================
// HTTP Response Tests
// =============================================================================

func TestError_ToResponse(t *testing.T) {
	resp := errors.ErrUserNotFound.ToResponse()

	expectedError := "user.not_found"
	if resp.Error != expectedError {
		t.Fatalf("expected error '%s', got '%s'", expectedError, resp.Error)
	}

	if resp.Message == "" {
		t.Fatal("response message should not be empty")
	}
}

func TestError_ToResponseWithDetails(t *testing.T) {
	details := map[string]interface{}{
		"user_id": "123",
		"reason":  "deleted",
	}
	resp := errors.ErrUserNotFound.ToResponseWithDetails(details)

	if resp.Details == nil {
		t.Fatal("response details should not be nil")
	}
	if resp.Details["user_id"] != "123" {
		t.Fatal("response details should contain user_id")
	}
}

func TestNewResponse(t *testing.T) {
	// With *Error
	resp := errors.NewResponse(errors.ErrUserNotFound)
	if resp.Error != "user.not_found" {
		t.Fatalf("expected error 'user.not_found', got '%s'", resp.Error)
	}

	// With standard error
	resp = errors.NewResponse(stderrors.New("standard error"))
	if resp.Error != "internal.internal_error" {
		t.Fatalf("expected error 'internal.internal_error', got '%s'", resp.Error)
	}
}

func TestNewResponseWithMessage(t *testing.T) {
	resp := errors.NewResponseWithMessage(errors.ErrUserNotFound, "Custom message")
	if resp.Message != "Custom message" {
		t.Fatalf("expected message 'Custom message', got '%s'", resp.Message)
	}
}

func TestNewValidationResponse(t *testing.T) {
	fields := []errors.ValidationError{
		errors.NewValidationField("email", "invalid format"),
		errors.NewValidationField("password", "too short"),
	}
	resp := errors.NewValidationResponse(fields)

	if resp.Error != "validation.validation_failed" {
		t.Fatalf("expected error 'validation.validation_failed', got '%s'", resp.Error)
	}
	if len(resp.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(resp.Fields))
	}
	if resp.Fields[0].Field != "email" {
		t.Fatalf("expected field 'email', got '%s'", resp.Fields[0].Field)
	}
}

// =============================================================================
// Predefined Errors Tests
// =============================================================================

func TestPredefinedErrors_Auth(t *testing.T) {
	authErrors := []*errors.Error{
		errors.ErrInvalidCredentials,
		errors.ErrTokenExpired,
		errors.ErrTokenInvalid,
		errors.ErrTokenRevoked,
		errors.ErrNoToken,
		errors.ErrInsufficientPermissions,
	}

	for _, err := range authErrors {
		if err.Domain != errors.DomainAuth {
			t.Fatalf("expected domain %s for %s, got %s", errors.DomainAuth, err.Code, err.Domain)
		}
		if err.HTTPStatus == 0 {
			t.Fatalf("HTTP status should be set for %s", err.Code)
		}
	}
}

func TestPredefinedErrors_User(t *testing.T) {
	userErrors := []*errors.Error{
		errors.ErrUserNotFound,
		errors.ErrUserAlreadyExists,
		errors.ErrEmailAlreadyExists,
		errors.ErrRootUserExists,
	}

	for _, err := range userErrors {
		if err.Domain != errors.DomainUser {
			t.Fatalf("expected domain %s for %s, got %s", errors.DomainUser, err.Code, err.Domain)
		}
	}
}

func TestPredefinedErrors_Role(t *testing.T) {
	roleErrors := []*errors.Error{
		errors.ErrRoleNotFound,
		errors.ErrRoleAlreadyExists,
		errors.ErrSystemRoleModification,
		errors.ErrSystemRoleDeletion,
	}

	for _, err := range roleErrors {
		if err.Domain != errors.DomainRole {
			t.Fatalf("expected domain %s for %s, got %s", errors.DomainRole, err.Code, err.Domain)
		}
	}
}

func TestPredefinedErrors_Storage(t *testing.T) {
	storageErrors := []*errors.Error{
		errors.ErrStorageNotFound,
		errors.ErrStorageUploadFailed,
		errors.ErrStorageDownloadFailed,
		errors.ErrStorageDeleteFailed,
		errors.ErrStorageUnavailable,
	}

	for _, err := range storageErrors {
		if err.Domain != errors.DomainStorage {
			t.Fatalf("expected domain %s for %s, got %s", errors.DomainStorage, err.Code, err.Domain)
		}
	}
}

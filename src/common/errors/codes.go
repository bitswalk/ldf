package errors

import "net/http"

// Common error codes used across domains
const (
	CodeNotFound       Code = "not_found"
	CodeAlreadyExists  Code = "already_exists"
	CodeInvalidRequest Code = "invalid_request"
	CodeUnauthorized   Code = "unauthorized"
	CodeForbidden      Code = "forbidden"
	CodeConflict       Code = "conflict"
	CodeInternal       Code = "internal_error"
	CodeUnavailable    Code = "unavailable"
	CodeTimeout        Code = "timeout"
	CodeRateLimited    Code = "rate_limited"
)

// ============================================================================
// Authentication Errors
// ============================================================================

var (
	// ErrInvalidCredentials is returned when authentication fails due to invalid credentials
	ErrInvalidCredentials = New(DomainAuth, "invalid_credentials", http.StatusUnauthorized,
		"Invalid credentials")

	// ErrTokenExpired is returned when a JWT token has expired
	ErrTokenExpired = New(DomainAuth, "token_expired", http.StatusUnauthorized,
		"Token has expired")

	// ErrTokenInvalid is returned when a JWT token is malformed or invalid
	ErrTokenInvalid = New(DomainAuth, "token_invalid", http.StatusUnauthorized,
		"Invalid token")

	// ErrTokenRevoked is returned when a JWT token has been revoked
	ErrTokenRevoked = New(DomainAuth, "token_revoked", http.StatusUnauthorized,
		"Token has been revoked")

	// ErrNoToken is returned when no authentication token is provided
	ErrNoToken = New(DomainAuth, "no_token", http.StatusUnauthorized,
		"No authentication token provided")

	// ErrInsufficientPermissions is returned when user lacks required permissions
	ErrInsufficientPermissions = New(DomainAuth, "insufficient_permissions", http.StatusForbidden,
		"Insufficient permissions")
)

// ============================================================================
// User Errors
// ============================================================================

var (
	// ErrUserNotFound is returned when a user cannot be found
	ErrUserNotFound = New(DomainUser, CodeNotFound, http.StatusNotFound,
		"User not found")

	// ErrUserAlreadyExists is returned when trying to create a user that already exists
	ErrUserAlreadyExists = New(DomainUser, CodeAlreadyExists, http.StatusConflict,
		"User already exists")

	// ErrEmailAlreadyExists is returned when the email is already registered
	ErrEmailAlreadyExists = New(DomainUser, "email_exists", http.StatusConflict,
		"Email already exists")

	// ErrRootUserExists is returned when trying to create a root user but one already exists
	ErrRootUserExists = New(DomainUser, "root_exists", http.StatusConflict,
		"Root user already exists")

	// ErrInvalidUserData is returned when user data fails validation
	ErrInvalidUserData = New(DomainUser, CodeInvalidRequest, http.StatusBadRequest,
		"Invalid user data")
)

// ============================================================================
// Role Errors
// ============================================================================

var (
	// ErrRoleNotFound is returned when a role cannot be found
	ErrRoleNotFound = New(DomainRole, CodeNotFound, http.StatusNotFound,
		"Role not found")

	// ErrRoleAlreadyExists is returned when trying to create a role that already exists
	ErrRoleAlreadyExists = New(DomainRole, CodeAlreadyExists, http.StatusConflict,
		"Role already exists")

	// ErrSystemRoleModification is returned when trying to modify a system role
	ErrSystemRoleModification = New(DomainRole, CodeForbidden, http.StatusForbidden,
		"Cannot modify system role")

	// ErrSystemRoleDeletion is returned when trying to delete a system role
	ErrSystemRoleDeletion = New(DomainRole, CodeForbidden, http.StatusForbidden,
		"Cannot delete system role")

	// ErrInvalidRoleData is returned when role data fails validation
	ErrInvalidRoleData = New(DomainRole, CodeInvalidRequest, http.StatusBadRequest,
		"Invalid role data")
)

// ============================================================================
// Distribution Errors
// ============================================================================

var (
	// ErrDistributionNotFound is returned when a distribution cannot be found
	ErrDistributionNotFound = New(DomainDistribution, CodeNotFound, http.StatusNotFound,
		"Distribution not found")

	// ErrDistributionAlreadyExists is returned when trying to create a distribution that already exists
	ErrDistributionAlreadyExists = New(DomainDistribution, CodeAlreadyExists, http.StatusConflict,
		"Distribution already exists")

	// ErrDistributionAccessDenied is returned when user cannot access a distribution
	ErrDistributionAccessDenied = New(DomainDistribution, CodeForbidden, http.StatusForbidden,
		"Access denied to distribution")

	// ErrInvalidDistributionData is returned when distribution data fails validation
	ErrInvalidDistributionData = New(DomainDistribution, CodeInvalidRequest, http.StatusBadRequest,
		"Invalid distribution data")
)

// ============================================================================
// Storage Errors
// ============================================================================

var (
	// ErrStorageNotFound is returned when a storage object cannot be found
	ErrStorageNotFound = New(DomainStorage, CodeNotFound, http.StatusNotFound,
		"Object not found in storage")

	// ErrStorageUploadFailed is returned when a storage upload fails
	ErrStorageUploadFailed = New(DomainStorage, "upload_failed", http.StatusInternalServerError,
		"Failed to upload object to storage")

	// ErrStorageDownloadFailed is returned when a storage download fails
	ErrStorageDownloadFailed = New(DomainStorage, "download_failed", http.StatusInternalServerError,
		"Failed to download object from storage")

	// ErrStorageDeleteFailed is returned when a storage delete fails
	ErrStorageDeleteFailed = New(DomainStorage, "delete_failed", http.StatusInternalServerError,
		"Failed to delete object from storage")

	// ErrStorageUnavailable is returned when the storage backend is unavailable
	ErrStorageUnavailable = New(DomainStorage, CodeUnavailable, http.StatusServiceUnavailable,
		"Storage backend unavailable")
)

// ============================================================================
// Database Errors
// ============================================================================

var (
	// ErrDatabaseConnection is returned when database connection fails
	ErrDatabaseConnection = New(DomainDatabase, "connection_failed", http.StatusServiceUnavailable,
		"Database connection failed")

	// ErrDatabaseQuery is returned when a database query fails
	ErrDatabaseQuery = New(DomainDatabase, "query_failed", http.StatusInternalServerError,
		"Database query failed")

	// ErrDatabaseTransaction is returned when a database transaction fails
	ErrDatabaseTransaction = New(DomainDatabase, "transaction_failed", http.StatusInternalServerError,
		"Database transaction failed")
)

// ============================================================================
// Validation Errors
// ============================================================================

var (
	// ErrValidationFailed is returned when request validation fails
	ErrValidationFailed = New(DomainValidation, "validation_failed", http.StatusBadRequest,
		"Validation failed")

	// ErrMissingRequiredField is returned when a required field is missing
	ErrMissingRequiredField = New(DomainValidation, "missing_field", http.StatusBadRequest,
		"Missing required field")

	// ErrInvalidFieldValue is returned when a field value is invalid
	ErrInvalidFieldValue = New(DomainValidation, "invalid_value", http.StatusBadRequest,
		"Invalid field value")

	// ErrInvalidJSON is returned when JSON parsing fails
	ErrInvalidJSON = New(DomainValidation, "invalid_json", http.StatusBadRequest,
		"Invalid JSON")
)

// ============================================================================
// Internal Errors
// ============================================================================

var (
	// ErrInternal is a generic internal server error
	ErrInternal = New(DomainInternal, CodeInternal, http.StatusInternalServerError,
		"Internal server error")

	// ErrNotImplemented is returned when a feature is not implemented
	ErrNotImplemented = New(DomainInternal, "not_implemented", http.StatusNotImplemented,
		"Not implemented")
)

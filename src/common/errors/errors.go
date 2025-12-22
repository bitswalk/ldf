// Package errors provides a structured error system for the LDF platform.
// It supports error codes, HTTP status mapping, error wrapping, and consistent
// error responses across all LDF components (ldfd, ldfctl).
package errors

import (
	"errors"
	"fmt"
)

// Code represents a unique error code within a domain
type Code string

// Domain represents an error domain (e.g., "auth", "distribution", "storage")
type Domain string

// Common error domains
const (
	DomainAuth         Domain = "auth"
	DomainUser         Domain = "user"
	DomainRole         Domain = "role"
	DomainDistribution Domain = "distribution"
	DomainStorage      Domain = "storage"
	DomainDatabase     Domain = "database"
	DomainValidation   Domain = "validation"
	DomainInternal     Domain = "internal"
)

// Error represents a structured error with domain, code, and HTTP status
type Error struct {
	// Domain categorizes the error (e.g., "auth", "storage")
	Domain Domain `json:"domain"`

	// Code is a unique identifier within the domain (e.g., "not_found", "unauthorized")
	Code Code `json:"code"`

	// Message is a human-readable error message
	Message string `json:"message"`

	// HTTPStatus is the corresponding HTTP status code
	HTTPStatus int `json:"-"`

	// cause is the underlying error if this error wraps another
	cause error
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s.%s: %s: %v", e.Domain, e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s.%s: %s", e.Domain, e.Code, e.Message)
}

// Unwrap returns the underlying error for errors.Is and errors.As support
func (e *Error) Unwrap() error {
	return e.cause
}

// Is implements error comparison for errors.Is
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Domain == t.Domain && e.Code == t.Code
}

// WithCause returns a new error with the underlying cause attached
func (e *Error) WithCause(cause error) *Error {
	return &Error{
		Domain:     e.Domain,
		Code:       e.Code,
		Message:    e.Message,
		HTTPStatus: e.HTTPStatus,
		cause:      cause,
	}
}

// WithMessage returns a new error with a custom message
func (e *Error) WithMessage(message string) *Error {
	return &Error{
		Domain:     e.Domain,
		Code:       e.Code,
		Message:    message,
		HTTPStatus: e.HTTPStatus,
		cause:      e.cause,
	}
}

// WithMessagef returns a new error with a formatted custom message
func (e *Error) WithMessagef(format string, args ...interface{}) *Error {
	return &Error{
		Domain:     e.Domain,
		Code:       e.Code,
		Message:    fmt.Sprintf(format, args...),
		HTTPStatus: e.HTTPStatus,
		cause:      e.cause,
	}
}

// New creates a new Error with the given parameters
func New(domain Domain, code Code, httpStatus int, message string) *Error {
	return &Error{
		Domain:     domain,
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// Wrap wraps an existing error with an Error
func Wrap(err error, domain Domain, code Code, httpStatus int, message string) *Error {
	return &Error{
		Domain:     domain,
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		cause:      err,
	}
}

// GetHTTPStatus returns the HTTP status code for an error.
// If the error is not an *Error, it returns 500 (Internal Server Error).
func GetHTTPStatus(err error) int {
	var e *Error
	if errors.As(err, &e) {
		return e.HTTPStatus
	}
	return 500
}

// GetCode returns the error code if the error is an *Error, otherwise empty string
func GetCode(err error) Code {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

// GetDomain returns the error domain if the error is an *Error, otherwise empty string
func GetDomain(err error) Domain {
	var e *Error
	if errors.As(err, &e) {
		return e.Domain
	}
	return ""
}

// Is checks if an error matches a target error (delegates to errors.Is)
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target (delegates to errors.As)
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

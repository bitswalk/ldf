package errors

// Response represents a standard error response for HTTP APIs
type Response struct {
	// Error contains the error code (domain.code format)
	Error string `json:"error"`

	// Message contains a human-readable error message
	Message string `json:"message"`

	// Details contains optional additional error details
	Details map[string]interface{} `json:"details,omitempty"`
}

// ToResponse converts an Error to an HTTP response structure
func (e *Error) ToResponse() Response {
	return Response{
		Error:   string(e.Domain) + "." + string(e.Code),
		Message: e.Message,
	}
}

// ToResponseWithDetails converts an Error to an HTTP response with additional details
func (e *Error) ToResponseWithDetails(details map[string]interface{}) Response {
	return Response{
		Error:   string(e.Domain) + "." + string(e.Code),
		Message: e.Message,
		Details: details,
	}
}

// NewResponse creates a new error response from an error.
// If the error is an *Error, it uses its domain and code.
// Otherwise, it creates a generic internal error response.
func NewResponse(err error) Response {
	if e, ok := err.(*Error); ok {
		return e.ToResponse()
	}

	// For non-Error types, return a generic internal error
	return Response{
		Error:   string(DomainInternal) + "." + string(CodeInternal),
		Message: "Internal server error",
	}
}

// NewResponseWithMessage creates a new error response with a custom message
func NewResponseWithMessage(err error, message string) Response {
	if e, ok := err.(*Error); ok {
		return Response{
			Error:   string(e.Domain) + "." + string(e.Code),
			Message: message,
		}
	}

	return Response{
		Error:   string(DomainInternal) + "." + string(CodeInternal),
		Message: message,
	}
}

// ValidationError represents a field-level validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationResponse represents a validation error response with field-level details
type ValidationResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Fields  []ValidationError `json:"fields,omitempty"`
}

// NewValidationResponse creates a validation error response with field details
func NewValidationResponse(fields []ValidationError) ValidationResponse {
	return ValidationResponse{
		Error:   string(DomainValidation) + ".validation_failed",
		Message: "Validation failed",
		Fields:  fields,
	}
}

// NewValidationField creates a new ValidationError for a specific field
func NewValidationField(field, message string) ValidationError {
	return ValidationError{
		Field:   field,
		Message: message,
	}
}

package api

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error" example:"Not found"`
	Code    int    `json:"code" example:"404"`
	Message string `json:"message" example:"The requested resource was not found"`
}

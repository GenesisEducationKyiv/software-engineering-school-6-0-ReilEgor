package dto

// ErrorResponse is the standard shape of an error returned by the API.
type ErrorResponse struct {
	Error string `json:"error" example:"invalid request body"` // Human-readable error message.
}

// ValidationErrorResponse is returned when a request fails multiple validation checks.
type ValidationErrorResponse struct {
	Errors []string `json:"errors" example:"invalid email format,invalid repository format (expected 'owner/repo')"` // List of validation failure messages.
}

// MessageResponse is a generic success payload carrying a human-readable message.
type MessageResponse struct {
	Message string `json:"message" example:"subscription confirmed successfully"` // Human-readable status message.
}

package models

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// SuccessResponse represents a generic success response.
type SuccessResponse struct {
	Message string `json:"message"`
}
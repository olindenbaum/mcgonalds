package model

// ErrorResponse represents a standardized error response for the API
type ErrorResponse struct {
	Status  int    `json:"status" example:"400"`
	Message string `json:"message" example:"Bad Request"`
	Error   string `json:"error,omitempty" example:"Invalid input data"`
}

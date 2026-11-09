package errors

type ErrorResponse struct {
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}

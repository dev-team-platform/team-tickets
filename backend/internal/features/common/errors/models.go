// Package errors defines expected application errors for API handlers.
package errors

import "net/http"

// Error is an expected request failure with optional details for the client.
type Error struct {
	StatusCode int
	Message    string
	Details    map[string]any
	Cause      error
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Cause }

func NewUnauthorized(message string, details map[string]any) *Error {
	return &Error{StatusCode: http.StatusUnauthorized, Message: message, Details: details}
}

func NewForbidden(message string, details map[string]any) *Error {
	return &Error{StatusCode: http.StatusForbidden, Message: message, Details: details}
}

func NewNotFound(message string, details map[string]any) *Error {
	return &Error{StatusCode: http.StatusNotFound, Message: message, Details: details}
}

func NewConflict(message string, details map[string]any) *Error {
	return &Error{StatusCode: http.StatusConflict, Message: message, Details: details}
}

func NewUnprocessableEntity(message string, details map[string]any) *Error {
	return &Error{StatusCode: http.StatusUnprocessableEntity, Message: message, Details: details}
}

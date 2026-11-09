package middlewares

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	appErrors "github.com/trannhanlv2004/team-tickets/internal/features/common/errors"
)

func TestErrorHandlerMapsReturnedErrors(t *testing.T) {
	details := map[string]any{"field": "name"}
	tests := []struct {
		name   string
		err    error
		status int
	}{
		{"unauthorized", appErrors.NewUnauthorized("unauthorized", details), http.StatusUnauthorized},
		{"forbidden", appErrors.NewForbidden("forbidden", details), http.StatusForbidden},
		{"not found", appErrors.NewNotFound("not found", details), http.StatusNotFound},
		{"conflict", appErrors.NewConflict("conflict", details), http.StatusConflict},
		{"unprocessable", appErrors.NewUnprocessableEntity("unprocessable", details), http.StatusUnprocessableEntity},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := ErrorHandler(Handle(func(w http.ResponseWriter, r *http.Request) error { return fmt.Errorf("wrapped: %w", tt.err) }))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
			var body appErrors.ErrorResponse
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if response.Code != tt.status || response.Header().Get("Content-Type") != "application/json" || body.Message != tt.err.Error() || body.Details["field"] != "name" {
				t.Fatalf("unexpected response: status %d, body %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestErrorHandlerHidesUnexpectedErrorsAndPanics(t *testing.T) {
	for _, value := range []any{errors.New("private error"), "private panic", map[string]string{"secret": "value"}} {
		t.Run(fmt.Sprintf("%T", value), func(t *testing.T) {
			handler := ErrorHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(value) }))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
			var body appErrors.ErrorResponse
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusInternalServerError || body.Message != "Something went wrong." || body.Details != nil {
				t.Fatalf("unexpected response: status %d, body %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestErrorHandlerPassesSuccessfulResponse(t *testing.T) {
	handler := ErrorHandler(Handle(func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
		return nil
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusCreated || response.Body.String() != "ok" {
		t.Fatalf("unexpected response: %d %s", response.Code, response.Body.String())
	}
}

func TestErrorHandlerRepanicsAfterResponseStarts(t *testing.T) {
	handler := ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("partial")); panic("late failure") }))
	response := httptest.NewRecorder()
	defer func() {
		if recovered := recover(); recovered != "late failure" {
			t.Fatalf("unexpected panic: %v", recovered)
		}
		if response.Code != http.StatusOK || response.Body.String() != "partial" {
			t.Fatalf("unexpected response: %d %s", response.Code, response.Body.String())
		}
	}()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
}

func TestErrorHandlerRepanicsAfterResponseIsFlushed(t *testing.T) {
	handler := ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Fatal(err)
		}
		panic("late failure")
	}))
	response := httptest.NewRecorder()
	defer func() {
		if recovered := recover(); recovered != "late failure" {
			t.Fatalf("unexpected panic: %v", recovered)
		}
		if !response.Flushed || response.Code != http.StatusOK {
			t.Fatalf("unexpected response: %d", response.Code)
		}
	}()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
}

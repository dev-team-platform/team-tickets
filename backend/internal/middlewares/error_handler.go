package middlewares

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	appErrors "github.com/trannhanlv2004/team-tickets/internal/features/common/errors"
)

// ErrorResponse is the common JSON response for failed requests.

// ErrorHandlerFunc is an HTTP handler that can return an error to ErrorHandler.
type ErrorHandlerFunc func(http.ResponseWriter, *http.Request) error

// Handle adapts an error-returning handler for use behind ErrorHandler.
func Handle(handler ErrorHandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			panic(err)
		}
	})
}

// ErrorHandler maps expected errors and unexpected panics to JSON responses.
// If a response has already started, it re-panics so the server can abort it.
func ErrorHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := &responseWriter{ResponseWriter: w}
		defer func() {
			if recovered := recover(); recovered != nil {
				if recovered == http.ErrAbortHandler || response.started {
					panic(recovered)
				}
				status, body := errorResponse(recovered)
				if status >= 500 {
					slog.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", recovered)
				} else {
					slog.Warn("request failed", "method", r.Method, "path", r.URL.Path, "status", status, "error", recovered)
				}
				for key := range w.Header() {
					w.Header().Del(key)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				_ = json.NewEncoder(w).Encode(body)
			}
		}()
		next.ServeHTTP(response, r)
	})
}

func errorResponse(value any) (int, appErrors.ErrorResponse) {
	if err, ok := value.(error); ok {
		var expected *appErrors.Error
		if errors.As(err, &expected) && expected != nil {
			switch expected.StatusCode {
			case http.StatusUnauthorized,
				http.StatusForbidden,
				http.StatusNotFound,
				http.StatusConflict,
				http.StatusUnprocessableEntity:
				return expected.StatusCode, appErrors.ErrorResponse{Message: expected.Message, Details: expected.Details}
			}
		}
	}
	return http.StatusInternalServerError, appErrors.ErrorResponse{Message: "Something went wrong."}
}

type responseWriter struct {
	http.ResponseWriter
	started bool
}

func (w *responseWriter) WriteHeader(status int) {
	if status >= 200 {
		w.started = true
	}
	w.ResponseWriter.WriteHeader(status)
}
func (w *responseWriter) Write(body []byte) (int, error) {
	w.started = true
	return w.ResponseWriter.Write(body)
}
func (w *responseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// FlushError keeps ResponseController flushes visible to the recovery handler.
func (w *responseWriter) FlushError() error {
	err := http.NewResponseController(w.ResponseWriter).Flush()
	if err == nil {
		w.started = true
	}
	return err
}

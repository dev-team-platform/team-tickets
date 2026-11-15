package service

import (
	"context"
	"errors"
)

// MessagingConsumer processes a delivery. Wrap invalid messages in
// NonRetryable to route them directly to the dead-letter queue.
type MessagingConsumer interface {
	Handle(context.Context, []byte) error
}

type permanentError struct{ err error }

func (e permanentError) Error() string { return e.err.Error() }
func (e permanentError) Unwrap() error { return e.err }
func NonRetryable(err error) error {
	if err == nil {
		return nil
	}
	return permanentError{
		err: err,
	}
}
func isNonRetryable(err error) bool { var target permanentError; return errors.As(err, &target) }

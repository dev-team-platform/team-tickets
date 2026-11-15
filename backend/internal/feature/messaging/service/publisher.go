package service

import (
	"context"

	"github.com/trannhanlv2004/team-tickets/internal/feature/messaging/model"
)

type MessagingPublisher[T any] interface {
	Handle(context.Context, model.OutboxEvent[T]) error
}

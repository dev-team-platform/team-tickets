package model

type OutboxEvent[T any] struct {
	RoutingKey     string
	Type           string
	Data           T
	ThrowOnFailure bool
}

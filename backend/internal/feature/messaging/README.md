# RabbitMQ messaging

The runtime declares the tickets topic exchange and direct retry, retry-return,
and dead-letter exchanges. It reconnects and redeclares topology after a broker
connection fails. Configure connection and topology with the `RABBITMQ_*`
variables in `env/.env.example`.

Construct `service.OutboxEventPublisher[T]` with the runtime's `Publisher` and
call `Handle` with a routing key, type and data. Each call creates a UUIDv7 event
ID and an `{id, source, type, data}` envelope with source `team-tickets`, then
publishes it as a persistent mandatory message and waits for broker confirmation.
`ThrowOnFailure` controls whether publish failures are returned; cancellation is
always returned. A cancelled or disconnected caller can receive an error after
the broker has accepted a message, so consumers should deduplicate by message ID.

Register consumers by passing `service.ConsumerConfig` values to `Bootstrap`.
Each consumer has its own durable queue and bindings. Failures are republished to
TTL retry queues in configured order and then dead-lettered; use
`service.NonRetryable` for errors that should immediately go to the dead-letter
queue. The original delivery is acknowledged only after a confirmed retry or
dead-letter publish. Redelivery is possible after a connection failure.

The current tickets backend does not have a consumer for the organization's
`user.logged-in.v1` event: its user model has no last-login field. No consumers
are registered by default. Future ticket services should receive the shared
runtime publisher and register their handlers at bootstrap.

This matches the organization service's in-memory publisher and event envelope.
Despite the `OutboxEvent` name, neither implementation stores events in a
transactional database outbox; a process failure can lose pending messages. No
outbox table or database migration is included.

To run the integration test against a disposable local broker with guest access,
set `RABBITMQ_INTEGRATION_PORT` and run `go test ./internal/feature/messaging`.

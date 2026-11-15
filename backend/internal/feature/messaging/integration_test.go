package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/trannhanlv2004/team-tickets/internal/config"
	"github.com/trannhanlv2004/team-tickets/internal/feature/messaging/model"
	"github.com/trannhanlv2004/team-tickets/internal/feature/messaging/service"
)

type integrationConsumer struct {
	attempts atomic.Int32
	received chan string
}

func (h *integrationConsumer) Handle(_ context.Context, body []byte) error {
	var envelope struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return service.NonRetryable(err)
	}
	if envelope.Data == "bad" {
		return service.NonRetryable(errors.New("invalid event"))
	}
	if h.attempts.Add(1) == 1 {
		return errors.New("retry once")
	}
	h.received <- envelope.Data
	return nil
}

// Set RABBITMQ_INTEGRATION_PORT to run against a disposable local broker.
func TestRabbitMQPublishRetryAndDeadLetterIntegration(t *testing.T) {
	portStr := os.Getenv("RABBITMQ_INTEGRATION_PORT")
	if portStr == "" {
		t.Skip("RABBITMQ_INTEGRATION_PORT is not set")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}
	var inspect *amqp.Connection
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		inspect, err = amqp.Dial(fmt.Sprintf("amqp://guest:guest@127.0.0.1:%d/", port))
		if err == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("broker not ready: %v", err)
	}
	defer inspect.Close()
	inspectCh, err := inspect.Channel()
	if err != nil {
		t.Fatal(err)
	}
	defer inspectCh.Close()
	prefix := "integration." + uuid.NewString()
	cfg := config.RabbitMQConfig{
		Host:               "127.0.0.1",
		Port:               port,
		User:               "guest",
		Password:           "guest",
		VHost:              "/",
		PublisherExchange:  prefix + ".events",
		RetryExchange:      prefix + ".retry",
		ReturnExchange:     prefix + ".return",
		DeadLetterExchange: prefix + ".dlx",
		AttemptHeader:      "x-retry-attempt",
		RetryDelays:        "fast:50",
		QueueCapacity:      10,
		ReconnectSeconds:   1,
		ShutdownSeconds:    5,
	}
	handler := &integrationConsumer{
		received: make(chan string, 1),
	}
	queueName := prefix + ".q"
	runtime, err := Bootstrap(cfg, []service.ConsumerConfig{
		{
			Name:     "test",
			Queue:    queueName,
			Prefetch: 1,
			Handler:  handler,
			Bindings: []service.Binding{
				{
					Exchange:     cfg.PublisherExchange,
					ExchangeType: "topic",
					RoutingKeys:  []string{"test.event"},
				},
			},
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := runtime.Shutdown(ctx); err != nil {
			t.Errorf("shutdown: %v", err)
		}
	}()
	publisher := service.OutboxEventPublisher[string]{
		Publisher: runtime.Publisher,
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := publisher.Handle(ctx, model.OutboxEvent[string]{
		RoutingKey:     "test.event",
		Type:           "test.event",
		Data:           "good",
		ThrowOnFailure: true,
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-handler.received:
		if got != "good" || handler.attempts.Load() != 2 {
			t.Fatalf("unexpected delivery: %q, attempts %d", got, handler.attempts.Load())
		}
	case <-ctx.Done():
		t.Fatal("retry was not delivered")
	}
	if err := publisher.Handle(ctx, model.OutboxEvent[string]{
		RoutingKey:     "test.event",
		Type:           "test.event",
		Data:           "bad",
		ThrowOnFailure: true,
	}); err != nil {
		t.Fatal(err)
	}
	for {
		message, ok, err := inspectCh.Get(queueName+".dlq", true)
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			if message.Type != "test.event" || message.ContentType != "application/cloudevents+json" {
				t.Fatalf("unexpected dead-letter properties: %+v", message)
			}
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("dead-letter not delivered")
		case <-time.After(20 * time.Millisecond):
		}
	}
	conn, err := runtime.connection.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	if err := publisher.Handle(ctx, model.OutboxEvent[string]{
		RoutingKey:     "test.event",
		Type:           "test.event",
		Data:           "after reconnect",
		ThrowOnFailure: true,
	}); err != nil {
		t.Fatalf("publish after reconnect: %v", err)
	}
	select {
	case got := <-handler.received:
		if got != "after reconnect" {
			t.Fatalf("unexpected message after reconnect: %q", got)
		}
	case <-ctx.Done():
		t.Fatal("consumer did not reconnect")
	}
	if err := publisher.Handle(ctx, model.OutboxEvent[string]{
		RoutingKey:     "unroutable",
		Type:           "test.event",
		Data:           "ignored",
		ThrowOnFailure: true,
	}); err == nil {
		t.Fatal("mandatory unroutable publish succeeded")
	}
}

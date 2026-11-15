package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/trannhanlv2004/team-tickets/internal/feature/messaging/model"
)

type stubPublisher struct {
	key     string
	message amqp.Publishing
	err     error
}

func (s *stubPublisher) Publish(_ context.Context, key string, message amqp.Publishing) error {
	s.key, s.message = key, message
	return s.err
}

func TestOutboxPublisherCreatesEnvelopeAndPersistentProperties(t *testing.T) {
	publisher := &stubPublisher{}
	handler := OutboxEventPublisher[map[string]string]{
		Publisher: publisher,
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	if err := handler.Handle(context.Background(), model.OutboxEvent[map[string]string]{
		RoutingKey: "ticket.created.v1",
		Type:       "ticket.created.v1",
		Data: map[string]string{
			"name": "example",
		},
	}); err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		ID     uuid.UUID         `json:"id"`
		Source string            `json:"source"`
		Type   string            `json:"type"`
		Data   map[string]string `json:"data"`
	}
	if err := json.Unmarshal(publisher.message.Body, &envelope); err != nil {
		t.Fatal(err)
	}
	if publisher.key != "ticket.created.v1" || envelope.ID.Version() != 7 || publisher.message.MessageId != envelope.ID.String() || envelope.Source != "team-tickets" || envelope.Type != "ticket.created.v1" || envelope.Data["name"] != "example" || publisher.message.DeliveryMode != amqp.Persistent || publisher.message.ContentType != "application/cloudevents+json" || publisher.message.Type != envelope.Type {
		t.Fatalf("unexpected message: %+v envelope: %+v", publisher.message, envelope)
	}
}

func TestOutboxPublisherFailurePolicy(t *testing.T) {
	failure := errors.New("unavailable")
	publisher := &stubPublisher{
		err: failure,
	}
	handler := OutboxEventPublisher[string]{
		Publisher: publisher,
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	event := model.OutboxEvent[string]{
		RoutingKey: "key",
		Type:       "type",
	}
	if err := handler.Handle(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	event.ThrowOnFailure = true
	if err := handler.Handle(context.Background(), event); !errors.Is(err, failure) {
		t.Fatalf("expected publish error, got %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := handler.Handle(ctx, event); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func TestParseRetryDelaysAndAttempt(t *testing.T) {
	got, err := ParseRetryDelays("10s:10000,1m:60000")
	want := []RetryDelay{
		{
			Name:         "10s",
			Milliseconds: 10000,
		},
		{
			Name:         "1m",
			Milliseconds: 60000,
		},
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, %v", got, err)
	}
	for _, input := range []string{"a:0", "a:1,a:2", "bad", "a:-2"} {
		if _, err := ParseRetryDelays(input); err == nil {
			t.Errorf("expected invalid delay %q", input)
		}
	}
	if retryAttempt([]byte("2")) != 2 || retryAttempt(int32(1)) != 1 || retryAttempt(-1) != 0 {
		t.Fatal("incorrect retry attempt parsing")
	}
}

func TestCopyDeliveryPreservesPropertiesAndCopiesHeaders(t *testing.T) {
	delivery := amqp.Delivery{
		Headers: amqp.Table{
			"x-retry-attempt": int32(1),
		},
		MessageId:     "id",
		CorrelationId: "correlation",
		Type:          "type",
		Body:          []byte("body"),
	}
	copy := copyDelivery(delivery)
	copy.Headers["x-retry-attempt"] = int32(2)
	if delivery.Headers["x-retry-attempt"] != int32(1) || copy.MessageId != "id" || copy.CorrelationId != "correlation" || copy.Type != "type" || string(copy.Body) != "body" {
		t.Fatal("delivery was not copied correctly")
	}
}

func TestOutboxPublisherSerializationFailurePolicy(t *testing.T) {
	publisher := &stubPublisher{}
	handler := OutboxEventPublisher[chan int]{
		Publisher: publisher,
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	event := model.OutboxEvent[chan int]{
		RoutingKey: "key",
		Type:       "type",
		Data:       make(chan int),
	}
	if err := handler.Handle(context.Background(), event); err != nil {
		t.Fatalf("expected logged failure, got %v", err)
	}
	event.ThrowOnFailure = true
	if err := handler.Handle(context.Background(), event); err == nil {
		t.Fatal("expected serialization error")
	}
	if publisher.key != "" {
		t.Fatal("published an invalid event")
	}
}

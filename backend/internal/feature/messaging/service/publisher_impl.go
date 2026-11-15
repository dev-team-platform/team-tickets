package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	client "github.com/trannhanlv2004/team-tickets/internal/client/messaging"
	"github.com/trannhanlv2004/team-tickets/internal/config"
	"github.com/trannhanlv2004/team-tickets/internal/feature/messaging/model"
)

type publishRequest struct {
	ctx        context.Context
	routingKey string
	message    amqp.Publishing
	result     chan error
}

type Publisher struct {
	connection *client.Connection
	cfg        config.RabbitMQConfig
	consumers  []ConsumerConfig
	delays     []RetryDelay
	requests   chan publishRequest
	stopping   chan struct{}
	done       chan struct{}
	cancel     context.CancelFunc
	mu         sync.RWMutex
	stopOnce   sync.Once
}

func NewPublisher(
	conn *client.Connection,
	cfg config.RabbitMQConfig,
	consumers []ConsumerConfig,
	delays []RetryDelay,
) (*Publisher, error) {
	if conn == nil || cfg.QueueCapacity < 1 {
		return nil, errors.New("publisher requires a connection and positive queue capacity")
	}
	ctx, cancel := context.WithCancel(context.Background())
	p := &Publisher{
		connection: conn,
		cfg:        cfg,
		consumers:  consumers,
		delays:     delays,
		requests:   make(chan publishRequest, cfg.QueueCapacity),
		stopping:   make(chan struct{}),
		done:       make(chan struct{}),
		cancel:     cancel,
	}
	go p.run(ctx)
	return p, nil
}

func (p *Publisher) Publish(ctx context.Context, routingKey string, message amqp.Publishing) error {
	if strings.TrimSpace(routingKey) == "" {
		return errors.New("routing key is required")
	}
	request := publishRequest{
		ctx:        ctx,
		routingKey: routingKey,
		message:    message,
		result:     make(chan error, 1),
	}
	request.message.Body = append([]byte(nil), message.Body...)
	if message.Headers != nil {
		request.message.Headers = make(amqp.Table, len(message.Headers))
		for key, value := range message.Headers {
			request.message.Headers[key] = value
		}
	}
	p.mu.RLock()
	select {
	case <-p.stopping:
		p.mu.RUnlock()
		return errors.New("publisher is shutting down")
	default:
	}
	select {
	case p.requests <- request:
		p.mu.RUnlock()
	case <-p.stopping:
		p.mu.RUnlock()
		return errors.New("publisher is shutting down")
	case <-ctx.Done():
		p.mu.RUnlock()
		return ctx.Err()
	}
	select {
	case err := <-request.result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-p.done:
		select {
		case err := <-request.result:
			return err
		default:
			return errors.New("publisher stopped before confirming message")
		}
	}
}

func (p *Publisher) run(ctx context.Context) {
	defer close(p.done)
	for request := range p.requests {
		if err := ctx.Err(); err != nil {
			request.result <- err
			continue
		}
		opCtx, cancel := context.WithCancel(request.ctx)
		stop := context.AfterFunc(ctx, cancel)
		err := p.publish(opCtx, request.routingKey, request.message)
		stop()
		cancel()
		request.result <- err
	}
}

func (p *Publisher) publish(ctx context.Context, routingKey string, message amqp.Publishing) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	conn, err := p.connection.Get(ctx)
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("open publisher channel: %w", err)
	}
	defer ch.Close()
	if err := declareTopology(ch, p.cfg, p.consumers, p.delays); err != nil {
		return fmt.Errorf("declare RabbitMQ topology: %w", err)
	}
	return publishConfirmed(ctx, ch, p.cfg.PublisherExchange, routingKey, message)
}

func publishConfirmed(ctx context.Context, ch *amqp.Channel, exchange, routingKey string, message amqp.Publishing) error {
	if err := ch.Confirm(false); err != nil {
		return fmt.Errorf("enable publisher confirms: %w", err)
	}
	confirmations := ch.NotifyPublish(make(chan amqp.Confirmation, 1))
	returns := ch.NotifyReturn(make(chan amqp.Return, 1))
	if err := ch.PublishWithContext(ctx, exchange, routingKey, true, false, message); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	select {
	case returned := <-returns:
		return fmt.Errorf("unroutable message: broker reply %d", returned.ReplyCode)
	case confirmation, ok := <-confirmations:
		if !ok || !confirmation.Ack {
			return errors.New("RabbitMQ did not confirm message")
		}
		select {
		case returned := <-returns:
			return fmt.Errorf("unroutable message: broker reply %d", returned.ReplyCode)
		default:
			return nil
		}
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Publisher) Shutdown(ctx context.Context) error {
	p.stopOnce.Do(func() {
		close(p.stopping)
		p.mu.Lock()
		close(p.requests)
		p.mu.Unlock()
	})
	select {
	case <-p.done:
		p.cancel()
		return nil
	case <-ctx.Done():
		p.cancel()
		return ctx.Err()
	}
}

type MessagePublisher interface {
	Publish(context.Context, string, amqp.Publishing) error
}

type OutboxEventPublisher[T any] struct {
	Publisher MessagePublisher
	Logger    *slog.Logger
}

func (p OutboxEventPublisher[T]) Handle(ctx context.Context, event model.OutboxEvent[T]) error {
	if p.Publisher == nil {
		return errors.New("outbox publisher is required")
	}
	if strings.TrimSpace(event.Type) == "" || strings.TrimSpace(event.RoutingKey) == "" {
		return errors.New("event type and routing key are required")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	envelope := struct {
		ID     uuid.UUID `json:"id"`
		Source string    `json:"source"`
		Type   string    `json:"type"`
		Data   T         `json:"data"`
	}{
		ID:     id,
		Source: "team-tickets",
		Type:   event.Type,
		Data:   event.Data,
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		err = fmt.Errorf("serialize outbox event: %w", err)
	}
	if err == nil {
		err = p.Publisher.Publish(ctx, event.RoutingKey, amqp.Publishing{
			ContentType:  "application/cloudevents+json",
			DeliveryMode: amqp.Persistent,
			MessageId:    id.String(),
			Type:         event.Type,
			Body:         body,
		})
	}
	logger := p.Logger
	if logger == nil {
		logger = slog.Default()
	}
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		logger.Error("unable to publish integration event", "message_id", id.String(), "message_type", event.Type, "routing_key", event.RoutingKey, "error", err)
		if event.ThrowOnFailure {
			return err
		}
		return nil
	}
	logger.Info("published integration event", "message_id", id.String(), "message_type", event.Type, "routing_key", event.RoutingKey)
	return nil
}

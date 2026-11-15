package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	client "github.com/trannhanlv2004/team-tickets/internal/client/messaging"
	"github.com/trannhanlv2004/team-tickets/internal/config"
)

type ConsumerWorker struct {
	connection *client.Connection
	cfg        config.RabbitMQConfig
	consumer   ConsumerConfig
	all        []ConsumerConfig
	delays     []RetryDelay
	logger     *slog.Logger
}

func NewConsumerWorker(
	conn *client.Connection,
	cfg config.RabbitMQConfig,
	consumer ConsumerConfig,
	all []ConsumerConfig,
	delays []RetryDelay,
	logger *slog.Logger,
) (*ConsumerWorker, error) {
	if conn == nil || consumer.Handler == nil || consumer.Queue == "" ||
		consumer.Name == "" || consumer.Prefetch < 1 || consumer.Prefetch > 65535 || cfg.ReconnectSeconds < 1 {
		return nil, fmt.Errorf("invalid RabbitMQ consumer configuration")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &ConsumerWorker{
		connection: conn,
		cfg:        cfg,
		consumer:   consumer,
		all:        all,
		delays:     delays,
		logger:     logger,
	}, nil
}

func (w *ConsumerWorker) Run(ctx context.Context) {
	for ctx.Err() == nil {
		if err := w.consume(ctx); err != nil && ctx.Err() == nil {
			w.logger.Warn("RabbitMQ consumer disconnected", "consumer", w.consumer.Name, "error", err)
		}
		timer := time.NewTimer(time.Duration(w.cfg.ReconnectSeconds) * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (w *ConsumerWorker) consume(ctx context.Context) error {
	conn, err := w.connection.Get(ctx)
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	if err := declareTopology(ch, w.cfg, w.all, w.delays); err != nil {
		return err
	}
	if err := ch.Qos(w.consumer.Prefetch, 0, false); err != nil {
		return err
	}
	deliveries, err := ch.ConsumeWithContext(ctx, w.consumer.Queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case delivery, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("consumer channel closed")
			}
			w.handle(ctx, ch, delivery)
		}
	}
}

func (w *ConsumerWorker) handle(ctx context.Context, ch *amqp.Channel, delivery amqp.Delivery) {
	err := handleSafely(ctx, w.consumer.Handler, delivery.Body)
	if err == nil {
		if ackErr := delivery.Ack(false); ackErr != nil {
			w.logger.Error("RabbitMQ acknowledgement failed", "consumer", w.consumer.Name, "error", ackErr)
		}
		return
	}
	if ctx.Err() != nil {
		_ = delivery.Nack(false, true)
		return
	}
	attempt := retryAttempt(delivery.Headers[w.cfg.AttemptHeader])
	w.logger.Warn("RabbitMQ message processing failed", "consumer", w.consumer.Name, "retry_attempt", attempt, "error", err)
	message := copyDelivery(delivery)
	var exchange, key string
	if isNonRetryable(err) || attempt >= len(w.delays) {
		exchange, key = w.cfg.DeadLetterExchange, w.consumer.Queue
	} else {
		exchange = w.cfg.RetryExchange
		key = w.consumer.Queue + ".retry." + w.delays[attempt].Name
		if message.Headers == nil {
			message.Headers = amqp.Table{}
		}
		message.Headers[w.cfg.AttemptHeader] = int32(attempt + 1)
		message.DeliveryMode = amqp.Persistent
	}
	// Use a separate channel: publisher confirms are tied to the channel and
	// failures must not acknowledge the original delivery.
	conn, connErr := w.connection.Get(ctx)
	if connErr == nil {
		var publishCh *amqp.Channel
		publishCh, connErr = conn.Channel()
		if connErr == nil {
			connErr = publishConfirmed(ctx, publishCh, exchange, key, message)
			_ = publishCh.Close()
		}
	}
	if connErr != nil {
		w.logger.Error("RabbitMQ retry routing failed", "consumer", w.consumer.Name, "error", connErr)
		_ = delivery.Nack(false, true)
		return
	}
	if err := delivery.Ack(false); err != nil {
		w.logger.Error("RabbitMQ acknowledgement failed", "consumer", w.consumer.Name, "error", err)
	}
}

func handleSafely(ctx context.Context, handler MessagingConsumer, body []byte) (err error) {
	defer func() {
		if recover() != nil {
			err = fmt.Errorf("RabbitMQ consumer handler panicked")
		}
	}()
	return handler.Handle(ctx, body)
}

func retryAttempt(value interface{}) int {
	var n int64
	switch v := value.(type) {
	case int:
		n = int64(v)
	case int8:
		n = int64(v)
	case int16:
		n = int64(v)
	case int32:
		n = int64(v)
	case int64:
		n = v
	case string:
		n, _ = strconv.ParseInt(v, 10, 32)
	case []byte:
		n, _ = strconv.ParseInt(string(v), 10, 32)
	}
	if n < 0 || n > int64(^uint32(0)>>1) {
		return 0
	}
	return int(n)
}

func copyDelivery(delivery amqp.Delivery) amqp.Publishing {
	headers := make(amqp.Table, len(delivery.Headers))
	for key, value := range delivery.Headers {
		headers[key] = value
	}
	return amqp.Publishing{
		Headers:         headers,
		ContentType:     delivery.ContentType,
		ContentEncoding: delivery.ContentEncoding,
		DeliveryMode:    delivery.DeliveryMode,
		MessageId:       delivery.MessageId,
		CorrelationId:   delivery.CorrelationId,
		Type:            delivery.Type,
		AppId:           delivery.AppId,
		Body:            delivery.Body,
	}
}

package service

import (
	"context"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	client "github.com/trannhanlv2004/team-tickets/internal/client/messaging"
	"github.com/trannhanlv2004/team-tickets/internal/config"
)

// RunTopology declares durable topology at startup and after reconnects.
func RunTopology(
	ctx context.Context,
	connection *client.Connection,
	cfg config.RabbitMQConfig,
	consumers []ConsumerConfig,
	delays []RetryDelay,
	logger *slog.Logger,
) {
	for ctx.Err() == nil {
		conn, err := connection.Get(ctx)
		if err == nil {
			var ch *amqp.Channel
			ch, err = conn.Channel()
			if err == nil {
				err = declareTopology(ch, cfg, consumers, delays)
				_ = ch.Close()
			}
		}
		if err == nil {
			notifications := conn.NotifyClose(make(chan *amqp.Error, 1))
			select {
			case <-ctx.Done():
				return
			case <-notifications:
			}
		} else if ctx.Err() == nil {
			logger.Warn("RabbitMQ topology initialization failed", "error", err)
		}
		timer := time.NewTimer(time.Duration(cfg.ReconnectSeconds) * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

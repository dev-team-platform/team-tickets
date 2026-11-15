package service

import (
	"fmt"
	"strconv"
	"strings"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/trannhanlv2004/team-tickets/internal/config"
)

type RetryDelay struct {
	Name         string
	Milliseconds int32
}

type Binding struct {
	Exchange     string
	ExchangeType string
	RoutingKeys  []string
}

type ConsumerConfig struct {
	Name     string
	Queue    string
	Prefetch int
	Bindings []Binding
	Handler  MessagingConsumer
}

func ParseRetryDelays(value string) ([]RetryDelay, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	var result []RetryDelay
	seen := make(map[string]bool)
	for _, part := range strings.Split(value, ",") {
		parts := strings.Split(strings.TrimSpace(part), ":")
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			return nil, fmt.Errorf("invalid RABBITMQ_RETRY_DELAYS")
		}
		name := strings.TrimSpace(parts[0])
		ttl, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 32)
		if err != nil || ttl < 1 || seen[name] {
			return nil, fmt.Errorf("invalid RABBITMQ_RETRY_DELAYS")
		}
		seen[name] = true
		result = append(result, RetryDelay{
			Name:         name,
			Milliseconds: int32(ttl),
		})
	}
	return result, nil
}

func declareTopology(ch *amqp.Channel, cfg config.RabbitMQConfig, consumers []ConsumerConfig, delays []RetryDelay) error {
	exchanges := make(map[string]string)
	declare := func(name, kind string) error {
		if prev, exists := exchanges[name]; exists {
			if prev != kind {
				return fmt.Errorf("exchange %q has conflicting types", name)
			}
			return nil
		}
		if err := ch.ExchangeDeclare(name, kind, true, false, false, false, nil); err != nil {
			return err
		}
		exchanges[name] = kind
		return nil
	}
	for _, item := range []struct{ name, kind string }{
		{
			name: cfg.PublisherExchange,
			kind: "topic",
		},
		{
			name: cfg.RetryExchange,
			kind: "direct",
		},
		{
			name: cfg.ReturnExchange,
			kind: "direct",
		},
		{
			name: cfg.DeadLetterExchange,
			kind: "direct",
		},
	} {
		if err := declare(item.name, item.kind); err != nil {
			return err
		}
	}
	for _, consumer := range consumers {
		if _, err := ch.QueueDeclare(consumer.Queue, true, false, false, false, nil); err != nil {
			return err
		}
		if err := ch.QueueBind(consumer.Queue, consumer.Queue, cfg.ReturnExchange, false, nil); err != nil {
			return err
		}
		for _, binding := range consumer.Bindings {
			if err := declare(binding.Exchange, binding.ExchangeType); err != nil {
				return err
			}
			for _, key := range binding.RoutingKeys {
				if err := ch.QueueBind(consumer.Queue, key, binding.Exchange, false, nil); err != nil {
					return err
				}
			}
		}
		deadQueue := consumer.Queue + ".dlq"
		if _, err := ch.QueueDeclare(deadQueue, true, false, false, false, nil); err != nil {
			return err
		}
		if err := ch.QueueBind(deadQueue, consumer.Queue, cfg.DeadLetterExchange, false, nil); err != nil {
			return err
		}
		for _, delay := range delays {
			name := consumer.Queue + ".retry." + delay.Name
			args := amqp.Table{
				"x-message-ttl":             delay.Milliseconds,
				"x-dead-letter-exchange":    cfg.ReturnExchange,
				"x-dead-letter-routing-key": consumer.Queue,
			}
			if _, err := ch.QueueDeclare(name, true, false, false, false, args); err != nil {
				return err
			}
			if err := ch.QueueBind(name, name, cfg.RetryExchange, false, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

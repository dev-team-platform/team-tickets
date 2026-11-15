package config

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"
)

type RabbitMQConfig struct {
	User               string
	Password           string
	VHost              string
	Host               string
	Port               int
	PublisherExchange  string
	RetryExchange      string
	ReturnExchange     string
	DeadLetterExchange string
	AttemptHeader      string
	RetryDelays        string
	QueueCapacity      int
	ReconnectSeconds   int
	ShutdownSeconds    int
}

func loadRabbitMQConfig() (RabbitMQConfig, error) {
	port, err := portEnv("RABBITMQ_PORT", 0)
	if err != nil {
		return RabbitMQConfig{}, err
	}
	capacity, err := intEnv("RABBITMQ_PUBLISH_QUEUE_CAPACITY", 5000)
	if err != nil {
		return RabbitMQConfig{}, err
	}
	reconnect, err := intEnv("RABBITMQ_RECONNECT_SECONDS", 5)
	if err != nil {
		return RabbitMQConfig{}, err
	}
	shutdown, err := intEnv("RABBITMQ_SHUTDOWN_SECONDS", 10)
	if err != nil {
		return RabbitMQConfig{}, err
	}
	if capacity < 1 || reconnect < 1 || shutdown < 1 || int64(reconnect) > math.MaxInt64/int64(time.Second) || int64(shutdown) > math.MaxInt64/int64(time.Second) {
		return RabbitMQConfig{}, fmt.Errorf("RabbitMQ queue capacity, reconnect and shutdown times must be positive")
	}
	cfg := RabbitMQConfig{
		User:               os.Getenv("RABBITMQ_DEFAULT_USER"),
		Password:           os.Getenv("RABBITMQ_DEFAULT_PASS"),
		VHost:              os.Getenv("RABBITMQ_DEFAULT_VHOST"),
		Host:               os.Getenv("RABBITMQ_HOST"),
		Port:               port,
		PublisherExchange:  envOr("RABBITMQ_PUBLISHER_EXCHANGE", "team.tickets.events"),
		RetryExchange:      envOr("RABBITMQ_RETRY_EXCHANGE", "team.tickets.retry"),
		ReturnExchange:     envOr("RABBITMQ_RETRY_RETURN_EXCHANGE", "team.tickets.retry.return"),
		DeadLetterExchange: envOr("RABBITMQ_DEAD_LETTER_EXCHANGE", "team.tickets.dlx"),
		AttemptHeader:      envOr("RABBITMQ_RETRY_ATTEMPT_HEADER", "x-retry-attempt"),
		RetryDelays:        envOr("RABBITMQ_RETRY_DELAYS", "10s:10000,1m:60000,10m:600000"),
		QueueCapacity:      capacity,
		ReconnectSeconds:   reconnect,
		ShutdownSeconds:    shutdown,
	}
	if strings.TrimSpace(cfg.PublisherExchange) == "" || strings.TrimSpace(cfg.RetryExchange) == "" || strings.TrimSpace(cfg.ReturnExchange) == "" || strings.TrimSpace(cfg.DeadLetterExchange) == "" || strings.TrimSpace(cfg.AttemptHeader) == "" {
		return RabbitMQConfig{}, fmt.Errorf("RabbitMQ exchange names and attempt header cannot be blank")
	}
	return cfg, nil
}

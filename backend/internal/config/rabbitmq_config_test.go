package config

import (
	"strings"
	"testing"
)

func TestRabbitMQConfigReadsSettings(t *testing.T) {
	t.Setenv("RABBITMQ_PUBLISHER_EXCHANGE", "events")
	t.Setenv("RABBITMQ_RETRY_DELAYS", "a:50,b:100")
	t.Setenv("RABBITMQ_PUBLISH_QUEUE_CAPACITY", "9")
	cfg, err := loadRabbitMQConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PublisherExchange != "events" || cfg.RetryDelays != "a:50,b:100" || cfg.QueueCapacity != 9 {
		t.Fatalf("unexpected configuration: %+v", cfg)
	}
}

func TestRabbitMQConfigRejectsInvalidCapacity(t *testing.T) {
	t.Setenv("RABBITMQ_PUBLISH_QUEUE_CAPACITY", "bad")
	_, err := loadRabbitMQConfig()
	if err == nil || !strings.Contains(err.Error(), "RABBITMQ_PUBLISH_QUEUE_CAPACITY") {
		t.Fatalf("unexpected error: %v", err)
	}
}

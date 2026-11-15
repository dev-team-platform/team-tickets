package messaging

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	client "github.com/trannhanlv2004/team-tickets/internal/client/messaging"
	"github.com/trannhanlv2004/team-tickets/internal/config"
	"github.com/trannhanlv2004/team-tickets/internal/feature/messaging/service"
)

type Runtime struct {
	Publisher     *service.Publisher
	connection    *client.Connection
	cancel        context.CancelFunc
	consumersDone chan struct{}
}

// Bootstrap starts a publisher and the supplied consumers. RabbitMQ connections
// are established lazily and consumers reconnect in the background.
func Bootstrap(
	cfg config.RabbitMQConfig,
	consumers []service.ConsumerConfig,
	logger *slog.Logger,
) (*Runtime, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.QueueCapacity < 1 || cfg.ReconnectSeconds < 1 || cfg.ShutdownSeconds < 1 ||
		strings.TrimSpace(cfg.PublisherExchange) == "" || strings.TrimSpace(cfg.RetryExchange) == "" ||
		strings.TrimSpace(cfg.ReturnExchange) == "" || strings.TrimSpace(cfg.DeadLetterExchange) == "" ||
		strings.TrimSpace(cfg.AttemptHeader) == "" {
		return nil, fmt.Errorf("invalid RabbitMQ messaging configuration")
	}
	if cfg.PublisherExchange == cfg.RetryExchange || cfg.PublisherExchange == cfg.ReturnExchange ||
		cfg.PublisherExchange == cfg.DeadLetterExchange {
		return nil, fmt.Errorf("publisher and retry exchanges must be distinct")
	}
	delays, err := service.ParseRetryDelays(cfg.RetryDelays)
	if err != nil {
		return nil, err
	}
	conn, err := client.NewConnection(cfg)
	if err != nil {
		return nil, err
	}
	// Own the consumer configuration so callers cannot mutate active topology.
	ownedConsumers := make([]service.ConsumerConfig, len(consumers))
	for i, consumer := range consumers {
		if consumer.Prefetch == 0 {
			consumer.Prefetch = 10
		}
		consumer.Bindings = append([]service.Binding(nil), consumer.Bindings...)
		for j, binding := range consumer.Bindings {
			if strings.TrimSpace(binding.Exchange) == "" {
				return nil, fmt.Errorf("RabbitMQ binding exchange is required")
			}
			if binding.ExchangeType == "" {
				binding.ExchangeType = "topic"
			}
			binding.RoutingKeys = append([]string(nil), binding.RoutingKeys...)
			for _, key := range binding.RoutingKeys {
				if strings.TrimSpace(key) == "" {
					return nil, fmt.Errorf("RabbitMQ binding routing key is required")
				}
			}
			consumer.Bindings[j] = binding
		}
		ownedConsumers[i] = consumer
	}
	consumers = ownedConsumers
	workers := make([]*service.ConsumerWorker, 0, len(consumers))
	names := make(map[string]bool)
	queues := make(map[string]bool)
	for _, consumer := range consumers {
		if names[consumer.Name] || queues[consumer.Queue] {
			return nil, fmt.Errorf("duplicate RabbitMQ consumer name or queue")
		}
		names[consumer.Name], queues[consumer.Queue] = true, true
		worker, err := service.NewConsumerWorker(conn, cfg, consumer, consumers, delays, logger)
		if err != nil {
			return nil, err
		}
		workers = append(workers, worker)
	}
	publisher, err := service.NewPublisher(conn, cfg, consumers, delays)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	runtime := &Runtime{
		Publisher:     publisher,
		connection:    conn,
		cancel:        cancel,
		consumersDone: make(chan struct{}),
	}
	var wg sync.WaitGroup
	wg.Go(func() { service.RunTopology(ctx, conn, cfg, consumers, delays, logger) })
	for _, worker := range workers {
		wg.Go(func() { worker.Run(ctx) })
	}
	go func() { wg.Wait(); close(runtime.consumersDone) }()
	return runtime, nil
}

// StopConsumers stops delivery before dependent services begin shutdown.
func (r *Runtime) StopConsumers(ctx context.Context) error {
	r.cancel()
	select {
	case <-r.consumersDone:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Runtime) Shutdown(ctx context.Context) error {
	consumerErr := r.StopConsumers(ctx)
	err := r.Publisher.Shutdown(ctx)
	if closeErr := r.connection.Close(); err == nil {
		err = closeErr
	}
	if consumerErr != nil {
		return consumerErr
	}
	return err
}

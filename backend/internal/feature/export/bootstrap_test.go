package export

import (
	"context"
	"testing"
	"time"

	"github.com/trannhanlv2004/team-tickets/internal/config"
)

func TestBootstrapStartsConfiguredQueue(t *testing.T) {
	queue, err := Bootstrap(config.ExportConfig{
		QueueConfig: config.QueueConfig{
			QueueCapacity: 1,
		},
		Workers: 1,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	completed := make(chan struct{})
	if err := queue.TryEnqueue(func(context.Context) error {
		close(completed)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := queue.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-completed:
	default:
		t.Fatal("queued export did not complete")
	}
}

func TestBootstrapRejectsInvalidQueueConfiguration(t *testing.T) {
	if _, err := Bootstrap(config.ExportConfig{}, nil); err == nil {
		t.Fatal("expected invalid queue configuration error")
	}
}

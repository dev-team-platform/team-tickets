package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	entity "github.com/trannhanlv2004/team-tickets/internal/feature/auditlog/entity"
)

// Batch is admitted as a whole; callers must not mutate it after enqueueing.
type Batch []entity.AuditLog

type BatchHandler func(context.Context, Batch) error

type QueueMetrics struct {
	Enqueued           uint64
	Processed          uint64
	Failed             uint64
	Dropped            uint64
	Depth              int
	ProcessingDuration time.Duration
}

// Queue is a bounded, non-blocking queue. A slot contains one transaction batch.
type Queue struct {
	mu              sync.RWMutex
	closed          bool
	jobs            chan Batch
	cancel          context.CancelFunc
	done            chan struct{}
	logger          *slog.Logger
	enqueued        atomic.Uint64
	processed       atomic.Uint64
	failed          atomic.Uint64
	dropped         atomic.Uint64
	processingNanos atomic.Uint64
}

func NewQueue(capacity, workers int, handler BatchHandler, logger *slog.Logger) (*Queue, error) {
	if capacity < 1 || workers < 1 || handler == nil {
		return nil, fmt.Errorf("audit queue requires positive capacity and worker count and a handler")
	}
	if logger == nil {
		logger = slog.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())
	q := &Queue{
		jobs:   make(chan Batch, capacity),
		cancel: cancel,
		done:   make(chan struct{}),
		logger: logger,
	}
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for batch := range q.jobs {
				q.process(ctx, handler, batch)
			}
		})
	}
	go func() {
		wg.Wait()
		close(q.done)
	}()
	return q, nil
}

func (q *Queue) process(ctx context.Context, handler BatchHandler, batch Batch) {
	started := time.Now()
	defer func() { q.processingNanos.Add(uint64(time.Since(started))) }()
	defer func() {
		if recovered := recover(); recovered != nil {
			q.failed.Add(uint64(len(batch)))
			q.logger.Error("audit batch handler panicked", batchLogAttrs(batch)...)
		}
	}()
	if err := handler(ctx, batch); err != nil {
		q.failed.Add(uint64(len(batch)))
		q.logger.Error("audit batch failed", append(batchLogAttrs(batch), "error", err)...)
		return
	}
	q.processed.Add(uint64(len(batch)))
}

func batchTransactionID(batch Batch) string {
	if len(batch) > 0 && batch[0].TransactionID != nil {
		return batch[0].TransactionID.String()
	}
	return ""
}

func batchLogAttrs(batch Batch) []any {
	attrs := []any{
		"transaction_id",
		batchTransactionID(batch),
	}
	if len(batch) > 0 {
		attrs = append(attrs, "actor_id", batch[0].ActorID)
		if batch[0].TraceID != nil {
			attrs = append(attrs, "trace_id", *batch[0].TraceID)
		}
	}
	return attrs
}

// Enqueue makes an independent copy and never waits for queue capacity.
func (q *Queue) Enqueue(batch Batch) bool {
	if len(batch) == 0 {
		return true
	}
	q.mu.RLock()
	defer q.mu.RUnlock()
	if !q.closed {
		copyBatch := append(Batch(nil), batch...)
		select {
		case q.jobs <- copyBatch:
			q.enqueued.Add(uint64(len(batch)))
			return true
		default:
		}
	}
	q.dropped.Add(uint64(len(batch)))
	q.logger.Error("audit batch dropped", append(batchLogAttrs(batch), "events", len(batch))...)
	return false
}

func (q *Queue) Metrics() QueueMetrics {
	return QueueMetrics{
		Enqueued:           q.enqueued.Load(),
		Processed:          q.processed.Load(),
		Failed:             q.failed.Load(),
		Dropped:            q.dropped.Load(),
		Depth:              len(q.jobs),
		ProcessingDuration: time.Duration(q.processingNanos.Load()),
	}
}

// Shutdown closes admission and drains the queue until ctx expires.
func (q *Queue) Shutdown(ctx context.Context) error {
	q.mu.Lock()
	if !q.closed {
		q.closed = true
		close(q.jobs)
	}
	q.mu.Unlock()
	select {
	case <-q.done:
		q.cancel()
		return nil
	case <-ctx.Done():
		q.cancel()
		q.logger.Error("audit shutdown timed out", "unprocessed_batches", len(q.jobs))
		return ctx.Err()
	}
}

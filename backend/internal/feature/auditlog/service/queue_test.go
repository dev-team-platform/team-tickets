package service

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestQueueCapacityAndDrain(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	q, err := NewQueue(1, 1, func(ctx context.Context, batch Batch) error {
		once.Do(func() { close(started) })
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	batch := Batch{
		{
			ID: uuid.New(),
		},
	}
	if !q.Enqueue(batch) {
		t.Fatal("first batch rejected")
	}
	<-started
	if !q.Enqueue(batch) {
		t.Fatal("second batch rejected")
	}
	if q.Enqueue(batch) {
		t.Fatal("full queue accepted batch")
	}
	close(release)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := q.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	if q.Enqueue(batch) {
		t.Fatal("closed queue accepted batch")
	}
	metrics := q.Metrics()
	if metrics.Enqueued != 2 || metrics.Processed != 2 || metrics.Dropped != 2 || metrics.Depth != 0 {
		t.Fatalf("unexpected metrics: %+v", metrics)
	}
}

func TestQueueConcurrentProducersAndShutdown(t *testing.T) {
	q, err := NewQueue(16, 2, func(context.Context, Batch) error { return nil }, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			for range 20 {
				q.Enqueue(Batch{
					{
						ID: uuid.New(),
					},
				})
			}
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := q.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	metrics := q.Metrics()
	if metrics.Enqueued+metrics.Dropped != 400 || metrics.Processed != metrics.Enqueued {
		t.Fatalf("unexpected metrics: %+v", metrics)
	}
}

func TestQueueShutdownCancelsProcessingOnTimeout(t *testing.T) {
	started := make(chan struct{})
	canceled := make(chan struct{})
	q, err := NewQueue(1, 1, func(ctx context.Context, _ Batch) error {
		close(started)
		<-ctx.Done()
		close(canceled)
		return ctx.Err()
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	q.Enqueue(Batch{
		{
			ID: uuid.New(),
		},
	})
	<-started
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := q.Shutdown(ctx); err == nil {
		t.Fatal("expected timeout")
	}
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("worker was not canceled")
	}
}

func TestQueueProcessesBatchesConcurrently(t *testing.T) {
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	q, err := NewQueue(2, 2, func(context.Context, Batch) error {
		started <- struct{}{}
		<-release
		return nil
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if !q.Enqueue(Batch{
			{
				ID: uuid.New(),
			},
		}) {
			t.Fatal("batch rejected")
		}
	}
	for range 2 {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("workers did not run concurrently")
		}
	}
	close(release)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := q.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
}

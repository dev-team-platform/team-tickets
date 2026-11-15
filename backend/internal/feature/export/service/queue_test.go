package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestShutdownDrainsJobsAndRejectsNewWork(t *testing.T) {
	q, err := NewQueue(2, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	var count atomic.Int32
	for i := 0; i < 10; i++ {
		if err := q.TryEnqueue(func(context.Context) error { count.Add(1); return nil }); err != nil {
			t.Fatal(err)
		}
	}
	if err := q.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if count.Load() != 10 {
		t.Fatalf("processed %d jobs", count.Load())
	}
	if err := q.TryEnqueue(func(context.Context) error { return nil }); !errors.Is(err, ErrQueueClosed) {
		t.Fatalf("expected closed queue, got %v", err)
	}
}

func TestShutdownCancelsRunningJobsOnDeadline(t *testing.T) {
	q, err := NewQueue(1, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	stopped := make(chan struct{})
	if err := q.TryEnqueue(func(ctx context.Context) error { close(started); <-ctx.Done(); close(stopped); return ctx.Err() }); err != nil {
		t.Fatal(err)
	}
	<-started
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := q.Shutdown(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}
	if err := q.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentEnqueueAndShutdown(t *testing.T) {
	q, err := NewQueue(2, 8, nil)
	if err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for i := 0; i < 10; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for j := 0; j < 100; j++ {
				_ = q.TryEnqueue(func(context.Context) error { return nil })
			}
		}()
	}
	if err := q.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	group.Wait()
}

package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var ErrQueueFull = errors.New("export queue is full")
var ErrQueueClosed = errors.New("export queue is closed")

// Job must observe its context so that shutdown can cancel work when its deadline expires.
type Job func(context.Context) error

type Queue struct {
	mu     sync.Mutex
	jobs   chan Job
	closed bool
	cancel context.CancelFunc
	done   chan struct{}
}

// NewQueue starts workerCount workers. onError is called for failed jobs.
func NewQueue(workerCount, capacity int, onError func(error)) (*Queue, error) {
	if workerCount < 1 || capacity < 1 {
		return nil, fmt.Errorf("worker count and queue capacity must be positive")
	}
	ctx, cancel := context.WithCancel(context.Background())
	q := &Queue{
		jobs:   make(chan Job, capacity),
		cancel: cancel,
		done:   make(chan struct{}),
	}
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func() {
			defer workers.Done()
			for job := range q.jobs {
				if ctx.Err() != nil {
					return
				}
				func() {
					defer func() {
						if value := recover(); value != nil && onError != nil {
							onError(fmt.Errorf("export job panicked: %v", value))
						}
					}()
					if err := job(ctx); err != nil && onError != nil {
						onError(err)
					}
				}()
			}
		}()
	}
	go func() { workers.Wait(); close(q.done) }()
	return q, nil
}

// TryEnqueue rejects work immediately when the queue is full or shutting down.
func (q *Queue) TryEnqueue(job Job) error {
	if job == nil {
		return fmt.Errorf("job is required")
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return ErrQueueClosed
	}
	select {
	case q.jobs <- job:
		return nil
	default:
		return ErrQueueFull
	}
}

// Shutdown stops accepting jobs and drains accepted work. On deadline, it cancels
// active work; callers can wait again with another context for workers to exit.
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
		return ctx.Err()
	}
}

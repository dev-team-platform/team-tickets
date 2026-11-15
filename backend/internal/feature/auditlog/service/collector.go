package service

import (
	"context"
	"sync"

	"github.com/google/uuid"

	entity "github.com/trannhanlv2004/team-tickets/internal/feature/auditlog/entity"
)

type collectorKey struct{}

// Collector accumulates the events in one transaction or nested savepoint.
type Collector struct {
	TransactionID uuid.UUID
	mu            sync.Mutex
	events        []entity.AuditLog
}

func NewCollector(transactionID uuid.UUID) *Collector {
	return &Collector{
		TransactionID: transactionID,
	}
}

func WithCollector(ctx context.Context, collector *Collector) context.Context {
	return context.WithValue(ctx, collectorKey{}, collector)
}

func CollectorFromContext(ctx context.Context) (*Collector, bool) {
	collector, ok := ctx.Value(collectorKey{}).(*Collector)
	return collector, ok && collector != nil
}

func (c *Collector) Add(event entity.AuditLog) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, event)
}

// Events returns an independent copy suitable for transferring to the queue.
func (c *Collector) Events() []entity.AuditLog {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]entity.AuditLog(nil), c.events...)
}

// Merge should only be called once a nested savepoint has succeeded.
func (c *Collector) Merge(child *Collector) {
	events := child.Events()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, events...)
}

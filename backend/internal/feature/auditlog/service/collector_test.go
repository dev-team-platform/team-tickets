package service

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"

	entity "github.com/trannhanlv2004/team-tickets/internal/feature/auditlog/entity"
)

func TestCollectorNestedScopes(t *testing.T) {
	parent := NewCollector(uuid.New())
	committed := uuid.New()
	child := NewCollector(parent.TransactionID)
	child.Add(entity.AuditLog{
		ID: committed,
	})
	parent.Merge(child)
	discarded := NewCollector(parent.TransactionID)
	discarded.Add(entity.AuditLog{
		ID: uuid.New(),
	})
	events := parent.Events()
	if len(events) != 1 || events[0].ID != committed {
		t.Fatalf("unexpected events: %+v", events)
	}
	events[0].ID = uuid.New()
	if parent.Events()[0].ID != committed {
		t.Fatal("snapshot modified collector")
	}
	ctx := WithCollector(context.Background(), parent)
	if got, ok := CollectorFromContext(ctx); !ok || got != parent {
		t.Fatal("collector not available in context")
	}
}

func TestCollectorConcurrentProducers(t *testing.T) {
	collector := NewCollector(uuid.New())
	var workers sync.WaitGroup
	for range 100 {
		workers.Go(func() {
			collector.Add(entity.AuditLog{
				ID: uuid.New(),
			})
		})
	}
	workers.Wait()
	if got := len(collector.Events()); got != 100 {
		t.Fatalf("got %d events, want 100", got)
	}
}

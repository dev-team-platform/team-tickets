package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/trannhanlv2004/team-tickets/internal/config"
	entity "github.com/trannhanlv2004/team-tickets/internal/feature/auditlog/entity"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testPersister(t *testing.T) *Persister {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&entity.AuditLog{}); err != nil {
		t.Fatal(err)
	}
	if err := RegisterCallbacks(db); err != nil {
		t.Fatal(err)
	}
	p, err := NewPersister(db, config.AuditLogConfig{
		QueueConfig: config.QueueConfig{
			MaxRetry:     2,
			RetryDelayMS: []int{0, 0},
		},
		WriteTimeoutSeconds: 2,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPersisterIdempotent(t *testing.T) {
	p := testPersister(t)
	transactionID := uuid.New()
	batch := Batch{
		{
			ID:            uuid.New(),
			TransactionID: &transactionID,
			Action:        entity.AuditActionCreate,
			EntityName:    "thing",
			EntityID:      "1",
			ActorID:       "actor",
		},
	}
	for range 2 {
		if err := p.Persist(context.Background(), batch); err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	if err := p.db.Model(&entity.AuditLog{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("count = %d: %v", count, err)
	}
}

func TestPersisterRetriesOnlyTransientErrors(t *testing.T) {
	p := testPersister(t)
	attempts := 0
	p.insert = func(context.Context, Batch) error {
		attempts++
		if attempts < 3 {
			return &pgconn.PgError{
				Code: "40001",
			}
		}
		return nil
	}
	if err := p.Persist(context.Background(), Batch{}); err != nil || attempts != 3 || p.RetryAttempts() != 2 {
		t.Fatalf("attempts %d retries %d: %v", attempts, p.RetryAttempts(), err)
	}
	attempts = 0
	p.insert = func(context.Context, Batch) error {
		attempts++
		return &pgconn.PgError{
			Code: "23505",
		}
	}
	if err := p.Persist(context.Background(), Batch{}); err == nil || attempts != 1 {
		t.Fatalf("permanent failure attempts %d: %v", attempts, err)
	}
}

func TestPersisterCancellationDuringDelay(t *testing.T) {
	p := testPersister(t)
	p.config.RetryDelayMS[0] = 10000
	attempted := make(chan struct{})
	p.insert = func(context.Context, Batch) error {
		close(attempted)
		return &pgconn.PgError{
			Code: "40001",
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- p.Persist(ctx, Batch{}) }()
	<-attempted
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected cancellation, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("retry did not stop on cancellation")
	}
}

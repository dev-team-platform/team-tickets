package uow

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	auditlogentities "github.com/trannhanlv2004/team-tickets/internal/feature/auditlog/entity"
	auditlogservices "github.com/trannhanlv2004/team-tickets/internal/feature/auditlog/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type captureQueue struct{ batches []auditlogservices.Batch }

func (q *captureQueue) Enqueue(batch auditlogservices.Batch) bool {
	q.batches = append(q.batches, batch)
	return true
}

type testRecord struct {
	ID   int
	Name string
}

func TestUnitOfWorkCommitRollbackAndNestedScopes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:uowtest?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&testRecord{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Create().After("gorm:create").Register("test:collect", func(tx *gorm.DB) {
		if tx.Error != nil {
			return
		}
		collector, ok := auditlogservices.CollectorFromContext(tx.Statement.Context)
		if ok {
			collector.Add(auditlogentities.AuditLog{
				ID:            uuid.New(),
				TransactionID: &collector.TransactionID,
			})
		}
	}); err != nil {
		t.Fatal(err)
	}
	queue := &captureQueue{}
	unit, err := New(db, queue, nil)
	if err != nil {
		t.Fatal(err)
	}
	insert := func(repos Repositories, name string) error {
		tx := repos.Context().Value(transactionKey{}).(*gorm.DB)
		return tx.WithContext(repos.Context()).Create(&testRecord{
			Name: name,
		}).Error
	}
	err = unit.Execute(context.Background(), func(repos Repositories) error {
		if err := insert(repos, "outer"); err != nil {
			return err
		}
		if err := unit.Execute(repos.Context(), func(nested Repositories) error { return insert(nested, "nested") }); err != nil {
			return err
		}
		if err := unit.Execute(repos.Context(), func(nested Repositories) error {
			if err := insert(nested, "discarded"); err != nil {
				return err
			}
			return errors.New("rollback savepoint")
		}); err == nil {
			t.Fatal("expected nested rollback")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(queue.batches) != 1 || len(queue.batches[0]) != 2 {
		t.Fatalf("unexpected audit batches: %+v", queue.batches)
	}
	first := queue.batches[0]
	if first[0].TransactionID == nil || first[1].TransactionID == nil || *first[0].TransactionID != *first[1].TransactionID {
		t.Fatal("transaction IDs differ")
	}
	if err := unit.Execute(context.Background(), func(repos Repositories) error { return insert(repos, "second") }); err != nil {
		t.Fatal(err)
	}
	if *queue.batches[1][0].TransactionID == *first[0].TransactionID {
		t.Fatal("transaction ID reused")
	}
	err = unit.Execute(context.Background(), func(repos Repositories) error {
		if err := insert(repos, "rolled-back"); err != nil {
			return err
		}
		return errors.New("rollback")
	})
	if err == nil || len(queue.batches) != 2 {
		t.Fatalf("rollback enqueued events: %v %+v", err, queue.batches)
	}
	var count int64
	if err := db.Model(&testRecord{}).Count(&count).Error; err != nil || count != 3 {
		t.Fatalf("unexpected row count %d: %v", count, err)
	}
}

type rejectingQueue struct{}

func (rejectingQueue) Enqueue(auditlogservices.Batch) bool { return false }

func TestUnitOfWorkQueueFailureDoesNotUndoCommit(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:uow_reject_queue?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&auditedRecord{}); err != nil {
		t.Fatal(err)
	}
	if err := auditlogservices.RegisterCallbacks(db); err != nil {
		t.Fatal(err)
	}
	unit, err := New(db, rejectingQueue{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = unit.Execute(auditlogservices.WithActorID(context.Background(), "actor"), func(repos Repositories) error {
		tx := repos.Context().Value(transactionKey{}).(*gorm.DB).WithContext(repos.Context())
		return tx.Create(&auditedRecord{
			ID: "one",
		}).Error
	})
	if err != nil {
		t.Fatalf("committed write returned error: %v", err)
	}
	var count int64
	if err := db.Model(&auditedRecord{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("business row missing: %d %v", count, err)
	}
}

func TestUnitOfWorkShutdownWaitsForActiveTransaction(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:uow_shutdown?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	unit, err := New(db, &captureQueue{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- unit.Execute(context.Background(), func(Repositories) error {
			close(started)
			<-release
			return nil
		})
	}()
	<-started
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := unit.Shutdown(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected timeout, got %v", err)
	}
	if err := unit.Execute(context.Background(), func(Repositories) error { return nil }); err == nil {
		t.Fatal("accepted new transaction during shutdown")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := unit.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

package uow

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/trannhanlv2004/team-tickets/internal/config"
	auditlogentities "github.com/trannhanlv2004/team-tickets/internal/feature/auditlog/entity"
	auditlogservices "github.com/trannhanlv2004/team-tickets/internal/feature/auditlog/service"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Set AUDIT_TEST_POSTGRES_DSN to a disposable PostgreSQL database to run this test.
func TestPostgresAuditCommitAndSavepointRollback(t *testing.T) {
	dsn := os.Getenv("AUDIT_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("AUDIT_TEST_POSTGRES_DSN is not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	defer sqlDB.Close()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	schema := "audit_test_" + strings.ReplaceAll(id.String(), "-", "")
	if err := db.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Exec("DROP SCHEMA " + schema + " CASCADE")
	if err := db.Exec("SET search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&auditedRecord{}, &auditlogentities.AuditLog{}); err != nil {
		t.Fatal(err)
	}
	if err := auditlogservices.RegisterCallbacks(db); err != nil {
		t.Fatal(err)
	}
	q := &captureQueue{}
	unit, err := New(db, q, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := auditlogservices.WithActorID(context.Background(), "actor")
	err = unit.Execute(ctx, func(repos Repositories) error {
		tx := repos.Context().Value(transactionKey{}).(*gorm.DB).WithContext(repos.Context())
		if err := tx.Create(&auditedRecord{
			ID:   "one",
			Name: "before",
		}).Error; err != nil {
			return err
		}
		if err := unit.Execute(repos.Context(), func(child Repositories) error {
			childTx := child.Context().Value(transactionKey{}).(*gorm.DB).WithContext(child.Context())
			if err := childTx.Model(&auditedRecord{}).Where("id = ?", "one").Update("name", "discarded").Error; err != nil {
				return err
			}
			return errors.New("rollback nested")
		}); err == nil {
			t.Fatal("expected rollback")
		}
		return tx.Model(&auditedRecord{}).Where("id = ?", "one").Update("name", "after").Error
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(q.batches) != 1 || len(q.batches[0]) != 2 {
		t.Fatalf("unexpected batch: %+v", q.batches)
	}
	if *q.batches[0][0].TransactionID != *q.batches[0][1].TransactionID {
		t.Fatal("transaction IDs differ")
	}
	persister, err := auditlogservices.NewPersister(db, config.AuditLogConfig{
		QueueConfig:         config.QueueConfig{},
		WriteTimeoutSeconds: 2,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := persister.Persist(context.Background(), q.batches[0]); err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	if err := db.Model(&auditlogentities.AuditLog{}).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("unexpected audit count: %d, %v", count, err)
	}
	var row auditedRecord
	if err := db.First(&row, "id = ?", "one").Error; err != nil || row.Name != "after" {
		t.Fatalf("unexpected row: %+v, %v", row, err)
	}
}

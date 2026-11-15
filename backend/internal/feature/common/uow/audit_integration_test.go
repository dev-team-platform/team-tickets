package uow

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	auditlogentities "github.com/trannhanlv2004/team-tickets/internal/feature/auditlog/entity"
	auditlogservices "github.com/trannhanlv2004/team-tickets/internal/feature/auditlog/service"
	userentities "github.com/trannhanlv2004/team-tickets/internal/feature/user/entity"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type auditedRecord struct {
	ID        string `gorm:"primaryKey"`
	Name      string
	Password  string
	Custom    string `audit:"ignore"`
	IsDeleted bool
}

func TestAuditCallbacksTransactionLifecycle(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:auditcallbacks?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&auditedRecord{}); err != nil {
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
	ctx := auditlogservices.WithMetadata(context.Background(), auditlogservices.Metadata{
		ActorID:        "actor",
		TraceID:        "trace",
		ClientActionID: "action",
	})
	if err := db.Create(&auditedRecord{
		ID: "outside",
	}).Error; err == nil {
		t.Fatal("outside write succeeded")
	}
	err = unit.Execute(ctx, func(repos Repositories) error {
		tx := repos.Context().Value(transactionKey{}).(*gorm.DB).WithContext(repos.Context())
		if err := tx.Create(&auditedRecord{
			ID:       "one",
			Name:     "original",
			Password: "private",
			Custom:   "secret",
		}).Error; err != nil {
			return err
		}
		return tx.Model(&auditedRecord{}).Where("id = ?", "one").Updates(map[string]interface{}{
			"name": "updated",
		}).Error
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(q.batches) != 1 || len(q.batches[0]) != 2 {
		t.Fatalf("unexpected batches: %+v", q.batches)
	}
	create, update := q.batches[0][0], q.batches[0][1]
	if create.Action != auditlogentities.AuditActionCreate || update.Action != auditlogentities.AuditActionUpdate || create.ActorID != "actor" || create.TraceID == nil || *create.TraceID != "trace" || *create.TransactionID != *update.TransactionID {
		t.Fatalf("unexpected events: %+v", q.batches)
	}
	var values map[string]interface{}
	if err := json.Unmarshal([]byte(*create.NewValues), &values); err != nil {
		t.Fatal(err)
	}
	if values["name"] != "original" || values["password"] != nil || values["custom"] != nil {
		t.Fatalf("unexpected create values: %+v", values)
	}
	if err := json.Unmarshal([]byte(*update.OldValues), &values); err != nil {
		t.Fatal(err)
	}
	if values["name"] != "original" {
		t.Fatalf("unexpected old values: %+v", values)
	}
	values = nil
	if err := json.Unmarshal([]byte(*update.NewValues), &values); err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values["name"] != "updated" {
		t.Fatalf("unexpected new values: %+v", values)
	}
	err = unit.Execute(ctx, func(repos Repositories) error {
		tx := repos.Context().Value(transactionKey{}).(*gorm.DB).WithContext(repos.Context())
		if err := tx.Delete(&auditedRecord{
			ID: "one",
		}).Error; err != nil {
			return err
		}
		return errors.New("abort")
	})
	if err == nil || len(q.batches) != 1 {
		t.Fatal("rollback did not discard audit")
	}
	err = unit.Execute(ctx, func(repos Repositories) error {
		tx := repos.Context().Value(transactionKey{}).(*gorm.DB).WithContext(repos.Context())
		return tx.Delete(&auditedRecord{
			ID: "one",
		}).Error
	})
	if err != nil || len(q.batches) != 2 || q.batches[1][0].Action != auditlogentities.AuditActionDelete {
		t.Fatalf("delete not audited: %v %+v", err, q.batches)
	}
}

func TestAuditCallbacksNestedRollbackSoftDeleteAndBatchRejection(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:audit_nested?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&auditedRecord{}); err != nil {
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
			Name: "first",
		}).Error; err != nil {
			return err
		}
		if err := unit.Execute(repos.Context(), func(nested Repositories) error {
			child := nested.Context().Value(transactionKey{}).(*gorm.DB).WithContext(nested.Context())
			if err := child.Model(&auditedRecord{}).Where("id = ?", "one").Update("name", "rolled back").Error; err != nil {
				return err
			}
			return errors.New("rollback")
		}); err == nil {
			t.Fatal("expected rollback")
		}
		return tx.Model(&auditedRecord{}).Where("id = ?", "one").Update("is_deleted", true).Error
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(q.batches) != 1 || len(q.batches[0]) != 2 || q.batches[0][1].Action != auditlogentities.AuditActionDelete {
		t.Fatalf("unexpected audit: %+v old=%v new=%v", q.batches, *q.batches[0][1].OldValues, *q.batches[0][1].NewValues)
	}
	var old map[string]interface{}
	if err := json.Unmarshal([]byte(*q.batches[0][1].OldValues), &old); err != nil || old["name"] != "first" {
		t.Fatalf("unexpected deleted row: %+v %v", old, err)
	}
	err = unit.Execute(ctx, func(repos Repositories) error {
		tx := repos.Context().Value(transactionKey{}).(*gorm.DB).WithContext(repos.Context())
		return tx.Create(&[]auditedRecord{
			{
				ID: "two",
			},
			{
				ID: "three",
			},
		}).Error
	})
	if err == nil || len(q.batches) != 1 {
		t.Fatalf("batch create was not rejected: %v", err)
	}
	if err := unit.Execute(context.Background(), func(repos Repositories) error {
		tx := repos.Context().Value(transactionKey{}).(*gorm.DB).WithContext(repos.Context())
		return tx.Create(&auditedRecord{
			ID: "no-actor",
		}).Error
	}); err == nil {
		t.Fatal("missing actor was not rejected")
	}
}

func TestAuditRejectsRawWrites(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:audit_raw?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&auditedRecord{}); err != nil {
		t.Fatal(err)
	}
	if err := auditlogservices.RegisterCallbacks(db); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO audited_records (id) VALUES (?)", "bad").Error; err == nil {
		t.Fatal("raw Exec write succeeded")
	}
	var rows []auditedRecord
	if err := db.Raw("DELETE FROM audited_records RETURNING *").Scan(&rows).Error; err == nil {
		t.Fatal("raw query write succeeded")
	}
	if err := db.Raw("SELECT * FROM audited_records").Scan(&rows).Error; err != nil {
		t.Fatalf("raw read rejected: %v", err)
	}
}

func TestAuditCallbacksUUIDPrimaryKeyAndForeignKey(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:audit_uuid?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&userentities.User{}); err != nil {
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
	id := uuid.New()
	ctx := auditlogservices.WithActorID(context.Background(), "actor")
	err = unit.Execute(ctx, func(repos Repositories) error {
		tx := repos.Context().Value(transactionKey{}).(*gorm.DB).WithContext(repos.Context())
		return tx.Create(&userentities.User{
			ID:              id,
			IdentitySubject: "subject",
			EmployeeCode:    "employee",
			Username:        "username",
			Email:           "example@example.com",
			FirstName:       "First",
			DisplayName:     "First",
		}).Error
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(q.batches) != 1 || len(q.batches[0]) != 1 {
		t.Fatalf("unexpected batches: %+v", q.batches)
	}
	event := q.batches[0][0]
	if event.EntityID != id.String() || event.ID.Version() != 7 || event.TransactionID == nil || event.TransactionID.Version() != 7 {
		t.Fatalf("unexpected UUIDs: %+v", event)
	}
	var user userentities.User
	if err := db.First(&user, "id = ?", id).Error; err != nil || user.ID != id {
		t.Fatalf("UUID did not round-trip: %+v, %v", user, err)
	}
	childID := uuid.New()
	err = unit.Execute(ctx, func(repos Repositories) error {
		tx := repos.Context().Value(transactionKey{}).(*gorm.DB).WithContext(repos.Context())
		return tx.Create(&userentities.User{
			ID:              childID,
			IdentitySubject: "child-subject",
			EmployeeCode:    "child",
			Username:        "child",
			Email:           "child@example.com",
			FirstName:       "Child",
			DisplayName:     "Child",
			CreatedByID:     &id,
		}).Error
	})
	if err != nil {
		t.Fatal(err)
	}
	var child userentities.User
	if err := db.First(&child, "id = ?", childID).Error; err != nil || child.CreatedByID == nil || *child.CreatedByID != id {
		t.Fatalf("foreign key did not round-trip: %+v, %v", child, err)
	}
}

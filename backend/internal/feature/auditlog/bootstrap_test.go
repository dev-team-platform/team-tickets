package auditlog

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/trannhanlv2004/team-tickets/internal/config"
	entity "github.com/trannhanlv2004/team-tickets/internal/feature/auditlog/entity"
	"github.com/trannhanlv2004/team-tickets/internal/feature/auditlog/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBootstrapRegistersCallbacksAndStartsWorker(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:audit_bootstrap?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&entity.AuditLog{}); err != nil {
		t.Fatal(err)
	}
	queue, err := Bootstrap(db, config.AuditLogConfig{
		QueueConfig: config.QueueConfig{
			QueueCapacity: 1,
		},
		Workers:             1,
		WriteTimeoutSeconds: 1,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if db.Callback().Create().Get("audit:before_create") == nil {
		t.Fatal("audit callbacks not registered")
	}
	if !queue.Enqueue(service.Batch{
		{
			ID:         uuid.New(),
			Action:     entity.AuditActionCreate,
			EntityName: "test",
			EntityID:   "test",
			ActorID:    "actor",
		},
	}) {
		t.Fatal("audit batch rejected")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := queue.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&entity.AuditLog{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("audit batch was not persisted: count=%d error=%v", count, err)
	}
}

func TestBootstrapRejectsInvalidConfigurationBeforeRegisteringCallbacks(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Bootstrap(db, config.AuditLogConfig{}, nil); err == nil {
		t.Fatal("expected invalid configuration error")
	}
	if db.Callback().Create().Get("audit:before_create") != nil {
		t.Fatal("invalid configuration registered callbacks")
	}
}

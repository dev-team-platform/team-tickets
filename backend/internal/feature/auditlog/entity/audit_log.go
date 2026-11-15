package auditlog

import (
	"time"

	"github.com/google/uuid"
)

type AuditAction string

const (
	AuditActionCreate AuditAction = "Create"
	AuditActionUpdate AuditAction = "Update"
	AuditActionDelete AuditAction = "Delete"
)

type AuditLog struct {
	ID              uuid.UUID   `gorm:"column:id;type:uuid;primaryKey"`
	Action          AuditAction `gorm:"column:action;size:20;not null"`
	TraceID         *string     `gorm:"column:trace_id;size:64"`
	SpanID          *string     `gorm:"column:span_id;size:32"`
	RequestID       *string     `gorm:"column:request_id;size:255"`
	TransactionID   *uuid.UUID  `gorm:"column:transaction_id;size:255;index:idx_audit_logs_transaction_id"`
	ClientActionID  *string     `gorm:"column:client_action_id;size:255"`
	ClientRequestID *string     `gorm:"column:client_request_id;size:255"`
	EntityName      string      `gorm:"column:entity_name;size:255;not null;index:idx_audit_logs_entity_name_entity_id,priority:1"`
	EntityID        string      `gorm:"column:entity_id;size:255;not null;index:idx_audit_logs_entity_name_entity_id,priority:2"`
	OldValues       *string     `gorm:"column:old_values;type:jsonb"`
	NewValues       *string     `gorm:"column:new_values;type:jsonb"`
	ActorID         string      `gorm:"column:actor_id;size:255;not null;index:idx_audit_logs_actor_id"`
	CreatedAt       time.Time   `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP;index:idx_audit_logs_created_at"`
}

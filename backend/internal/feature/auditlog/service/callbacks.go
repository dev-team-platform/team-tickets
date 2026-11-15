package service

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"

	entity "github.com/trannhanlv2004/team-tickets/internal/feature/auditlog/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type internalWriteKey struct{}

func internalWrite(ctx context.Context) context.Context {
	return context.WithValue(ctx, internalWriteKey{}, true)
}

// RegisterCallbacks installs the audit callbacks on a GORM database. Writes must
// use a UnitOfWork; raw SQL and multi-row operations are not supported.
func RegisterCallbacks(db *gorm.DB) error {
	registrations := []struct {
		register func(string, func(*gorm.DB)) error
		name     string
		callback func(*gorm.DB)
	}{
		{
			register: db.Callback().Raw().Before("gorm:raw").Register,
			name:     "audit:before_raw",
			callback: beforeRaw,
		},
		{
			register: db.Callback().Row().Before("gorm:row").Register,
			name:     "audit:before_raw_row",
			callback: beforeRawQuery,
		},
		{
			register: db.Callback().Query().Before("gorm:query").Register,
			name:     "audit:before_raw_query",
			callback: beforeRawQuery,
		},
		{
			register: db.Callback().Create().Before("gorm:begin_transaction").Register,
			name:     "audit:before_create",
			callback: beforeCreate,
		},
		{
			register: db.Callback().Create().After("gorm:create").Before("gorm:commit_or_rollback_transaction").Register,
			name:     "audit:after_create",
			callback: afterCreate,
		},
		{
			register: db.Callback().Update().Before("gorm:begin_transaction").Register,
			name:     "audit:before_update",
			callback: beforeChange,
		},
		{
			register: db.Callback().Update().After("gorm:update").Before("gorm:commit_or_rollback_transaction").Register,
			name:     "audit:after_update",
			callback: afterUpdate,
		},
		{
			register: db.Callback().Delete().Before("gorm:begin_transaction").Register,
			name:     "audit:before_delete",
			callback: beforeChange,
		},
		{
			register: db.Callback().Delete().After("gorm:delete").Before("gorm:commit_or_rollback_transaction").Register,
			name:     "audit:after_delete",
			callback: afterDelete,
		},
	}
	for _, registration := range registrations {
		if err := registration.register(registration.name, registration.callback); err != nil {
			return fmt.Errorf("register %s: %w", registration.name, err)
		}
	}
	return nil
}

func skipAudit(tx *gorm.DB) bool {
	allowed, _ := tx.Statement.Context.Value(internalWriteKey{}).(bool)
	return allowed
}

func requireScope(tx *gorm.DB) bool {
	if tx.Error != nil || skipAudit(tx) {
		return false
	}
	if tx.Statement.Table == "audit_logs" {
		tx.AddError(fmt.Errorf("audit: audit logs may only be written by the audit worker"))
		return false
	}
	if _, ok := CollectorFromContext(tx.Statement.Context); !ok {
		tx.AddError(fmt.Errorf("audit: writes require a unit of work"))
		return false
	}
	if strings.TrimSpace(MetadataFromContext(tx.Statement.Context).ActorID) == "" {
		tx.AddError(fmt.Errorf("audit: actor ID is required"))
		return false
	}
	if tx.Statement.Schema == nil || tx.Statement.Schema.PrioritizedPrimaryField == nil || len(tx.Statement.Schema.PrimaryFields) != 1 {
		tx.AddError(fmt.Errorf("audit: model must have exactly one primary key"))
		return false
	}
	return true
}

// Raw writes cannot be captured reliably. Only GORM's savepoint commands are
// accepted; reads use GORM query APIs rather than Exec.
func beforeRaw(tx *gorm.DB) {
	if tx.Error != nil || skipAudit(tx) {
		return
	}
	query := strings.Fields(strings.ToUpper(tx.Statement.SQL.String()))
	var name string
	if len(query) == 2 && query[0] == "SAVEPOINT" {
		name = query[1]
	}
	if len(query) == 4 && query[0] == "ROLLBACK" && query[1] == "TO" && query[2] == "SAVEPOINT" {
		name = query[3]
	}
	valid := name != ""
	for _, c := range name {
		if !(c >= 'A' && c <= 'Z') && !(c >= '0' && c <= '9') && c != '_' {
			valid = false
			break
		}
	}
	if valid {
		if _, ok := CollectorFromContext(tx.Statement.Context); ok {
			return
		}
	}
	tx.AddError(fmt.Errorf("audit: raw SQL execution is not supported"))
}

func beforeRawQuery(tx *gorm.DB) {
	if tx.Error != nil || skipAudit(tx) {
		return
	}
	query := strings.ToUpper(strings.TrimSpace(tx.Statement.SQL.String()))
	if query != "" && (!strings.HasPrefix(query, "SELECT ") || strings.Contains(strings.TrimSuffix(query, ";"), ";")) {
		tx.AddError(fmt.Errorf("audit: raw SQL writes are not supported"))
	}
}

func beforeCreate(tx *gorm.DB) {
	if !requireScope(tx) {
		return
	}
	if _, exists := tx.Statement.Clauses["ON CONFLICT"]; exists {
		tx.AddError(fmt.Errorf("audit: upserts are not supported"))
		return
	}
	v := reflect.Indirect(reflect.ValueOf(tx.Statement.Dest))
	if !v.IsValid() || v.Kind() != reflect.Struct {
		tx.AddError(fmt.Errorf("audit: create requires a single model struct"))
	}
}

func beforeChange(tx *gorm.DB) {
	if !requireScope(tx) {
		return
	}
	query := tx.Session(&gorm.Session{
		NewDB:     true,
		SkipHooks: true,
	}).Table(tx.Statement.Table).Clauses(clause.Locking{
		Strength: "UPDATE",
	})
	if where, ok := tx.Statement.Clauses["WHERE"]; ok {
		query = query.Clauses(where.Expression)
	}
	primary := tx.Statement.Schema.PrioritizedPrimaryField
	if tx.Statement.ReflectValue.IsValid() {
		v := reflect.Indirect(tx.Statement.ReflectValue)
		if v.IsValid() && v.Kind() == reflect.Struct {
			value, zero := primary.ValueOf(tx.Statement.Context, v)
			if !zero {
				query = query.Where(clause.Eq{
					Column: clause.Column{
						Name: primary.DBName,
					},
					Value: value,
				})
			}
		}
	}
	if _, ok := query.Statement.Clauses["WHERE"]; !ok {
		tx.AddError(fmt.Errorf("audit: update or delete requires a filter"))
		return
	}
	var rows []map[string]interface{}
	if err := query.Limit(2).Find(&rows).Error; err != nil {
		tx.AddError(fmt.Errorf("audit: read old values: %w", err))
		return
	}
	if len(rows) > 1 {
		tx.AddError(fmt.Errorf("audit: multi-row updates and deletes are not supported"))
		return
	}
	if len(rows) == 1 {
		tx.InstanceSet("audit:old", rows[0])
	}
}

func afterCreate(tx *gorm.DB) {
	if tx.Error != nil || skipAudit(tx) || tx.RowsAffected == 0 {
		return
	}
	if tx.RowsAffected != 1 {
		tx.AddError(fmt.Errorf("audit: multi-row creates are not supported"))
		return
	}
	primary := tx.Statement.Schema.PrioritizedPrimaryField
	id, zero := primary.ValueOf(tx.Statement.Context, reflect.Indirect(tx.Statement.ReflectValue))
	if zero {
		tx.AddError(fmt.Errorf("audit: created row has no primary key"))
		return
	}
	row, err := findByID(tx, id)
	if err != nil {
		tx.AddError(err)
		return
	}
	if row == nil {
		tx.AddError(fmt.Errorf("audit: created row not found"))
		return
	}
	record(tx, entity.AuditActionCreate, id, nil, row)
}

func afterUpdate(tx *gorm.DB) {
	if tx.Error != nil || skipAudit(tx) || tx.RowsAffected == 0 {
		return
	}
	oldValue, ok := tx.InstanceGet("audit:old")
	if !ok || tx.RowsAffected != 1 {
		tx.AddError(fmt.Errorf("audit: update did not affect the captured row"))
		return
	}
	old := oldValue.(map[string]interface{})
	id := old[tx.Statement.Schema.PrioritizedPrimaryField.DBName]
	current, err := findByID(tx, id)
	if err != nil {
		tx.AddError(err)
		return
	}
	if current == nil {
		tx.AddError(fmt.Errorf("audit: updated row not found"))
		return
	}
	if isSoftDeleted(old, current) {
		record(tx, entity.AuditActionDelete, id, old, nil)
		return
	}
	record(tx, entity.AuditActionUpdate, id, old, current)
}

func afterDelete(tx *gorm.DB) {
	if tx.Error != nil || skipAudit(tx) || tx.RowsAffected == 0 {
		return
	}
	oldValue, ok := tx.InstanceGet("audit:old")
	if !ok || tx.RowsAffected != 1 {
		tx.AddError(fmt.Errorf("audit: delete did not affect the captured row"))
		return
	}
	old := oldValue.(map[string]interface{})
	record(tx, entity.AuditActionDelete, old[tx.Statement.Schema.PrioritizedPrimaryField.DBName], old, nil)
}

func findByID(tx *gorm.DB, id interface{}) (map[string]interface{}, error) {
	var rows []map[string]interface{}
	primary := tx.Statement.Schema.PrioritizedPrimaryField.DBName
	err := tx.Session(&gorm.Session{
		NewDB:     true,
		SkipHooks: true,
	}).Table(tx.Statement.Table).Where(clause.Eq{
		Column: clause.Column{
			Name: primary,
		},
		Value: id,
	}).Limit(1).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("audit: read persisted values: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func isSoftDeleted(old, current map[string]interface{}) bool {
	return !truthy(old["is_deleted"]) && truthy(current["is_deleted"]) || old["deleted_at"] == nil && current["deleted_at"] != nil
}
func truthy(value interface{}) bool {
	vref := reflect.ValueOf(value)
	for vref.IsValid() && (vref.Kind() == reflect.Pointer || vref.Kind() == reflect.Interface) {
		if vref.IsNil() {
			return false
		}
		vref = vref.Elem()
	}
	if !vref.IsValid() {
		return false
	}
	switch vref.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return vref.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return vref.Uint() != 0
	case reflect.Float32, reflect.Float64:
		return vref.Float() != 0
	}
	switch v := vref.Interface().(type) {
	case bool:
		return v
	case int64:
		return v != 0
	case int:
		return v != 0
	case string:
		return v == "1" || strings.EqualFold(v, "true")
	case []byte:
		return string(v) == "1" || strings.EqualFold(string(v), "true")
	default:
		return false
	}
}

func record(tx *gorm.DB, action entity.AuditAction, id interface{}, old, current map[string]interface{}) {
	collector, ok := CollectorFromContext(tx.Statement.Context)
	if !ok {
		tx.AddError(fmt.Errorf("audit: missing transaction collector"))
		return
	}
	old = filteredValues(tx, old)
	current = filteredValues(tx, current)
	if action == entity.AuditActionUpdate {
		for key, val := range old {
			if reflect.DeepEqual(val, current[key]) {
				delete(old, key)
				delete(current, key)
			}
		}
		if len(old) == 0 && len(current) == 0 {
			return
		}
	}
	oldJSON, err := encodeValues(old)
	if err != nil {
		tx.AddError(err)
		return
	}
	newJSON, err := encodeValues(current)
	if err != nil {
		tx.AddError(err)
		return
	}
	idValue, err := uuid.NewV7()
	if err != nil {
		tx.AddError(err)
		return
	}
	metadata := MetadataFromContext(tx.Statement.Context)
	collector.Add(entity.AuditLog{
		ID:              idValue,
		TransactionID:   &collector.TransactionID,
		Action:          action,
		EntityName:      tx.Statement.Schema.Name,
		EntityID:        fmt.Sprint(id),
		OldValues:       oldJSON,
		NewValues:       newJSON,
		ActorID:         metadata.ActorID,
		TraceID:         optional(metadata.TraceID),
		SpanID:          optional(metadata.SpanID),
		RequestID:       optional(metadata.RequestID),
		ClientActionID:  optional(metadata.ClientActionID),
		ClientRequestID: optional(metadata.ClientRequestID),
		CreatedAt:       time.Now().UTC(),
	})
}

func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
func encodeValues(value map[string]interface{}) (*string, error) {
	if value == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("audit: encode values: %w", err)
	}
	result := string(encoded)
	return &result, nil
}

func filteredValues(tx *gorm.DB, source map[string]interface{}) map[string]interface{} {
	if source == nil {
		return nil
	}
	result := make(map[string]interface{})
	for _, field := range tx.Statement.Schema.Fields {
		if field.DBName == "" || field.StructField.Tag.Get("audit") == "ignore" || sensitive(field.DBName) {
			continue
		}
		switch field.DBName {
		case "created_at", "created_by_id", "updated_at", "updated_by_id":
			continue
		}
		value, ok := source[field.DBName]
		if ok {
			result[field.DBName] = scrub(value)
		}
	}
	return result
}

func sensitive(name string) bool {
	name = strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(name, "_", ""), "-", ""))
	for _, part := range []string{
		"password",
		"secret",
		"token",
		"otp",
		"apikey",
		"privatekey",
		"credential",
	} {
		if strings.Contains(name, part) {
			return true
		}
	}
	return false
}

func scrub(value interface{}) interface{} {
	vref := reflect.ValueOf(value)
	for vref.IsValid() && (vref.Kind() == reflect.Pointer || vref.Kind() == reflect.Interface) {
		if vref.IsNil() {
			return nil
		}
		vref = vref.Elem()
	}
	if !vref.IsValid() {
		return nil
	}
	value = vref.Interface()
	switch v := value.(type) {
	case string:
		if strings.HasPrefix(strings.TrimSpace(v), "{") || strings.HasPrefix(strings.TrimSpace(v), "[") {
			var parsed interface{}
			if json.Unmarshal([]byte(v), &parsed) == nil {
				return scrub(parsed)
			}
		}
		return v
	case []byte:
		var parsed interface{}
		if json.Unmarshal(v, &parsed) == nil {
			return scrub(parsed)
		}
		return string(v)
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, item := range v {
			if !sensitive(key) {
				result[key] = scrub(item)
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = scrub(item)
		}
		return result
	default:
		return value
	}
}

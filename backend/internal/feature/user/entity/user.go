package entity

import (
	"time"

	"github.com/google/uuid"
)

// User is the local representation of an organization user.
type User struct {
	ID              uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	IdentitySubject string     `gorm:"column:identity_subject;size:255;not null;uniqueIndex"`
	EmployeeCode    string     `gorm:"column:employee_code;size:10;not null;uniqueIndex"`
	Username        string     `gorm:"column:username;size:100;not null;uniqueIndex"`
	Email           string     `gorm:"column:email;size:255;not null;uniqueIndex"`
	FirstName       string     `gorm:"column:first_name;size:100;not null"`
	LastName        *string    `gorm:"column:last_name;size:100"`
	DisplayName     string     `gorm:"column:display_name;size:200;not null;index"`
	CreatedAt       time.Time  `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt       *time.Time `gorm:"column:updated_at"`
	CreatedByID     *uuid.UUID `gorm:"column:created_by_id;type:uuid"`
	CreatedBy       *User      `gorm:"foreignKey:CreatedByID"`
	UpdatedByID     *uuid.UUID `gorm:"column:updated_by_id;type:uuid"`
	UpdatedBy       *User      `gorm:"foreignKey:UpdatedByID"`
	IsDeleted       bool       `gorm:"column:is_deleted;not null;default:false"`
	DeletedAt       *time.Time `gorm:"column:deleted_at"`
}

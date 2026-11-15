package entity

import (
	"time"

	"github.com/google/uuid"

	userentities "github.com/trannhanlv2004/team-tickets/internal/feature/user/entity"
)

type FileType string

const (
	FileTypeUserListView FileType = "UserListView"
	FileTypeUserAvatar   FileType = "UserAvatar"
)

type FileStorage struct {
	ID          uuid.UUID          `gorm:"column:id;type:uuid;primaryKey"`
	ReferenceID *string            `gorm:"column:reference_id;size:255;index:idx_file_storage_reference_id"`
	ObjectKey   string             `gorm:"column:object_key;size:4096;not null"`
	FileName    string             `gorm:"column:file_name;size:2048;not null"`
	FileType    FileType           `gorm:"column:file_type;size:20;not null;index:idx_file_storage_file_type"`
	ContentType string             `gorm:"column:content_type;size:255;not null"`
	FileSize    int64              `gorm:"column:file_size;not null"`
	ExpiredAt   *time.Time         `gorm:"column:expired_at"`
	CreatedAt   time.Time          `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt   *time.Time         `gorm:"column:updated_at"`
	CreatedByID *uuid.UUID         `gorm:"column:created_by_id;type:uuid"`
	CreatedBy   *userentities.User `gorm:"foreignKey:CreatedByID"`
	UpdatedByID *uuid.UUID         `gorm:"column:updated_by_id;type:uuid"`
	UpdatedBy   *userentities.User `gorm:"foreignKey:UpdatedByID"`
	IsDeleted   bool               `gorm:"column:is_deleted;not null;default:false"`
	DeletedAt   *time.Time         `gorm:"column:deleted_at"`
}

func (FileStorage) TableName() string {
	return "file_storage"
}

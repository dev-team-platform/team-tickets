package repository

import "gorm.io/gorm"

type gormUserRepository struct {
	db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) UserRepository {
	return &gormUserRepository{
		db: db,
	}
}

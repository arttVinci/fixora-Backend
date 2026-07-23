package repository

import "gorm.io/gorm"

// Repository is the generic base repository.
// Contains only basic CRUD operations that do NOT involve tables from other modules.
type Repository[T any] struct {
}

func (r *Repository[T]) Create(db *gorm.DB, entity *T) error {
	return db.Create(entity).Error
}

func (r *Repository[T]) Update(db *gorm.DB, entity *T) error {
	return db.Save(entity).Error
}

func (r *Repository[T]) Delete(db *gorm.DB, entity *T) error {
	return db.Delete(entity).Error
}

func (r *Repository[T]) CountById(db *gorm.DB, id any) (int64, error) {
	var total int64
	err := db.Model(new(T)).Where("id = ?", id).Count(&total).Error
	return total, err
}

func (r *Repository[T]) Count(db *gorm.DB) (int64, error) {
	var total int64
	err := db.Model(new(T)).Count(&total).Error
	return total, err
}

func (r *Repository[T]) FindById(db *gorm.DB, entity *T, id any) error {
	return db.Where("id = ?", id).Take(entity).Error
}
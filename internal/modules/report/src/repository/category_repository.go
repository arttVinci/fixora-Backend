package repository

import (
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	shared_repo "github.com/arttVinci/fixora-Backend/internal/shared/repository"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type CategoryRepository struct {
	shared_repo.Repository[entity.Category]
	Log *logrus.Logger
}

func NewCategoryRepository(log *logrus.Logger) *CategoryRepository {
	return &CategoryRepository{Log: log}
}

func (r *CategoryRepository) FindAll(db *gorm.DB) ([]entity.Category, error) {
	var items []entity.Category
	err := db.Order("name asc").Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}
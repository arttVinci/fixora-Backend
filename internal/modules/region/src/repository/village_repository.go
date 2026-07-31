package repository

import (
	"strings"

	"github.com/arttVinci/fixora-Backend/internal/modules/region/src/entity"
	shared_repo "github.com/arttVinci/fixora-Backend/internal/shared/repository"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type VillageRepository struct {
	shared_repo.Repository[entity.Village]
	Log *logrus.Logger
}

func NewVillageRepository(log *logrus.Logger) *VillageRepository {
	return &VillageRepository{Log: log}
}

func (r *VillageRepository) SearchByName(db *gorm.DB, item *entity.Village, name string) error {
	name = strings.TrimSpace(name)
	return db.
		Preload("District.City.Province").
		Where("LOWER(name) = LOWER(?)", name).
		Or("LOWER(name) LIKE LOWER(?)", "%"+name+"%").
		Take(item).Error
}

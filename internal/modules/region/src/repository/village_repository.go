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

func (r *VillageRepository) FindByHierarchy(db *gorm.DB, item *entity.Village, villageName, districtName, cityName, provinceName string) error {
	villageName = strings.TrimSpace(villageName)
	districtName = strings.TrimSpace(districtName)
	cityName = strings.TrimSpace(cityName)
	provinceName = strings.TrimSpace(provinceName)

	if villageName == "" {
		return gorm.ErrRecordNotFound
	}

	base := db.Table("villages").
		Select("villages.*").
		Joins("JOIN districts ON districts.id = villages.district_id").
		Joins("JOIN cities ON cities.id = districts.city_id").
		Joins("JOIN provinces ON provinces.id = cities.province_id").
		Preload("District.City.Province").
		Where("LOWER(villages.name) = LOWER(?)", villageName)

	if err := base.Session(&gorm.Session{}).
		Where("LOWER(districts.name) = LOWER(?)", districtName).
		Where("(LOWER(cities.name) = LOWER(?) OR LOWER(cities.name) LIKE LOWER(CONCAT('% ', ?)))", cityName, cityName).
		Where("LOWER(provinces.name) = LOWER(?)", provinceName).
		Order("villages.id ASC").
		Take(item).Error; err == nil {
		return nil
	} else if err != gorm.ErrRecordNotFound {
		return err
	}

	if err := base.Session(&gorm.Session{}).
		Where("(LOWER(cities.name) = LOWER(?) OR LOWER(cities.name) LIKE LOWER(CONCAT('% ', ?)))", cityName, cityName).
		Where("LOWER(provinces.name) = LOWER(?)", provinceName).
		Order("villages.id ASC").
		Take(item).Error; err == nil {
		return nil
	} else if err != gorm.ErrRecordNotFound {
		return err
	}

	if err := base.Session(&gorm.Session{}).
		Where("LOWER(provinces.name) = LOWER(?)", provinceName).
		Order("villages.id ASC").
		Take(item).Error; err == nil {
		return nil
	} else if err != gorm.ErrRecordNotFound {
		return err
	}

	return gorm.ErrRecordNotFound
}

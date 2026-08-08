package seeder

import (
	"regexp"
	"strings"

	"github.com/arttVinci/fixora-Backend/database/seeders"
	"github.com/arttVinci/fixora-Backend/internal/modules/region/src/entity"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type RegionSeeder struct {
	DB  *gorm.DB
	Log *logrus.Logger
}

func NewRegionSeeder(db *gorm.DB, log *logrus.Logger) *RegionSeeder {
	return &RegionSeeder{
		DB:  db,
		Log: log,
	}
}

func (s *RegionSeeder) SeedIfEmpty() error {
	var count int64
	if err := s.DB.Model(&entity.Province{}).Count(&count).Error; err != nil {
		s.Log.Warnf("Failed to check province count: %+v", err)
		return err
	}

	if count > 0 {
		s.Log.Info("Regions table not empty, skipping seeder")
		return nil
	}

	s.Log.Info("Starting Region seeder...")

	content := seeders.RegionsSQL
	
	// Regex to match the values inside the INSERT statement.
	// Pattern: (id, 'code', 'name', 'type', 'postal_code'|NULL, 'parent_code'|NULL)
	// Example: (1, '11', 'Aceh', 'province', NULL, NULL)
	re := regexp.MustCompile(`\((\d+),\s*'([^']+)',\s*'(.*?)',\s*'([^']+)',\s*([^,]+),\s*([^)]+)\)`)
	matches := re.FindAllStringSubmatch(content, -1)

	var provinces []entity.Province
	var cities []entity.City
	var districts []entity.District
	var villages []entity.Village

	for _, match := range matches {
		if len(match) < 7 {
			continue
		}

		code := match[2]
		name := strings.ReplaceAll(match[3], `\'`, `'`)
		regionType := match[4]
		parentCode := strings.Trim(strings.TrimSpace(match[6]), "'")

		if parentCode == "NULL" {
			parentCode = ""
		}

		switch regionType {
		case "province":
			provinces = append(provinces, entity.Province{
				ID:   code,
				Code: code,
				Name: name,
			})
		case "regency":
			cities = append(cities, entity.City{
				ID:         code,
				Code:       code,
				Name:       name,
				ProvinceID: parentCode,
			})
		case "district":
			districts = append(districts, entity.District{
				ID:     code,
				Code:   code,
				Name:   name,
				CityID: parentCode,
			})
		case "village":
			villages = append(villages, entity.Village{
				ID:         code,
				Code:       code,
				Name:       name,
				DistrictID: parentCode,
			})
		}
	}

	s.Log.Infof("Parsed %d provinces, %d cities, %d districts, %d villages", len(provinces), len(cities), len(districts), len(villages))

	// Batch insert for performance
	batchSize := 1000

	if err := s.DB.CreateInBatches(provinces, batchSize).Error; err != nil {
		s.Log.Warnf("Failed to seed provinces: %+v", err)
		return err
	}

	if err := s.DB.CreateInBatches(cities, batchSize).Error; err != nil {
		s.Log.Warnf("Failed to seed cities: %+v", err)
		return err
	}

	if err := s.DB.CreateInBatches(districts, batchSize).Error; err != nil {
		s.Log.Warnf("Failed to seed districts: %+v", err)
		return err
	}

	if err := s.DB.CreateInBatches(villages, batchSize).Error; err != nil {
		s.Log.Warnf("Failed to seed villages: %+v", err)
		return err
	}

	s.Log.Info("Region seeding completed successfully")
	return nil
}

package seeder

import (
	"encoding/json"

	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type CategorySeeder struct {
	DB  *gorm.DB
	Log *logrus.Logger
}

func NewCategorySeeder(db *gorm.DB, log *logrus.Logger) *CategorySeeder {
	return &CategorySeeder{
		DB:  db,
		Log: log,
	}
}

func (s *CategorySeeder) SeedIfEmpty() error {
	var count int64
	if err := s.DB.Model(&entity.Category{}).Count(&count).Error; err != nil {
		s.Log.Warnf("Failed to check category count: %+v", err)
		return err
	}

	if count > 0 {
		s.Log.Info("Categories table not empty, skipping seeder")
		return nil
	}

	s.Log.Info("Starting Category seeder...")

	categories := []entity.Category{
		{
			ID:   uuid.NewString(),
			Name: "Sampah",
			Slug: "sampah",
			SearchKeywords: toJSON([]string{
				"sampah menumpuk",
				"sampah di sungai",
				"sampah di jalan",
				"tumpukan sampah",
				"sampah menggunung",
				"TPS liar",
				"sampah tidak diangkut",
				"sampah berserakan",
				"pencemaran sampah",
				"sampah di drainase",
			}),
		},
		{
			ID:   uuid.NewString(),
			Name: "Jalan Rusak",
			Slug: "jalan-rusak",
			SearchKeywords: toJSON([]string{
				"jalan rusak",
				"jalan berlubang",
				"jalan ambles",
				"jalan retak",
				"aspal rusak",
				"jalan hancur",
				"jalan rusak parah",
				"jalan tidak layak",
				"kerusakan jalan",
				"lubang jalan",
			}),
		},
		{
			ID:   uuid.NewString(),
			Name: "Drainase Tersumbat",
			Slug: "drainase-tersumbat",
			SearchKeywords: toJSON([]string{
				"drainase tersumbat",
				"got tersumbat",
				"selokan tersumbat",
				"saluran air tersumbat",
				"banjir drainase",
				"genangan air",
				"drainase mampet",
				"gorong-gorong tersumbat",
				"saluran air mampet",
				"air tergenang",
			}),
		},
		{
			ID:   uuid.NewString(),
			Name: "Jembatan Rusak",
			Slug: "jembatan-rusak",
			SearchKeywords: toJSON([]string{
				"jembatan rusak",
				"jembatan ambruk",
				"jembatan rawan roboh",
				"jembatan retak",
				"jembatan tidak layak",
				"kerusakan jembatan",
				"jembatan rapuh",
				"jembatan putus",
				"jembatan bahaya",
				"perbaikan jembatan",
			}),
		},
		{
			ID:   uuid.NewString(),
			Name: "Bangunan Terbengkalai",
			Slug: "bangunan-terbengkalai",
			SearchKeywords: toJSON([]string{
				"bangunan terbengkalai",
				"gedung terbengkalai",
				"bangunan mangkrak",
				"proyek mangkrak",
				"bangunan tidak terurus",
				"gedung kosong rusak",
				"bangunan rawan roboh",
				"bangunan miring",
				"proyek terbengkalai",
				"fasilitas terbengkalai",
			}),
		},
	}

	if err := s.DB.Create(&categories).Error; err != nil {
		s.Log.Warnf("Failed to seed categories: %+v", err)
		return err
	}

	s.Log.Infof("Category seeding completed successfully (%d categories)", len(categories))
	return nil
}

func toJSON(data []string) datatypes.JSON {
	b, _ := json.Marshal(data)
	return datatypes.JSON(b)
}

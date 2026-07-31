package region

import (
	"github.com/arttVinci/fixora-Backend/internal/modules/region/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/region/src/repository"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Module struct {
	DB                *gorm.DB
	Log               *logrus.Logger
	VillageRepository *repository.VillageRepository
}

func NewModule(db *gorm.DB, log *logrus.Logger) *Module {
	return &Module{
		DB:                db,
		Log:               log,
		VillageRepository: repository.NewVillageRepository(log),
	}
}

func (m *Module) RegisterRoutes(router fiber.Router, authMiddleware fiber.Handler) {
}

func (m *Module) Migrate() error {
	return m.DB.AutoMigrate(
		&entity.Province{},
		&entity.City{},
		&entity.District{},
		&entity.Village{},
	)
}

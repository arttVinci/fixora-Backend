package crawl

import (
	"github.com/arttVinci/fixora-Backend/internal/modules/crawl/src/entity"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Module struct {
	DB  *gorm.DB
	Log *logrus.Logger
}

func NewModule(db *gorm.DB, log *logrus.Logger) *Module {
	return &Module{DB: db, Log: log}
}

func (m *Module) RegisterRoutes(router fiber.Router, authMiddleware fiber.Handler) {
}

func (m *Module) Migrate() error {
	return m.DB.AutoMigrate(&entity.CrawledArticle{})
}

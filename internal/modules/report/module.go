package report

import (
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/controller"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type reportModule struct {
	DB               *gorm.DB
	Log              *logrus.Logger
	ReportController *controller.ReportController
}

func NewReportModule(
	db *gorm.DB,
	log *logrus.Logger,
	reportController *controller.ReportController,
) *reportModule {
	return &reportModule{
		DB:               db,
		Log:              log,
		ReportController: reportController,
	}
}

func (m *reportModule) Migrate() error {
	return m.DB.AutoMigrate(
		&entity.Category{},
		&entity.Reporter{},
		&entity.Report{},
		&entity.ReportPhoto{},
		&entity.ReportConfirmation{},
		&entity.MergeLog{},
	)
}

func (m *reportModule) RegisterRoutes(router fiber.Router, authMiddleware fiber.Handler) {
	RegisterRoutes(router, authMiddleware, m.ReportController)
}
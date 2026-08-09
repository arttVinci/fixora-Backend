package report

import (
	report_client "github.com/arttVinci/fixora-Backend/internal/modules/report-client"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/controller"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/repository"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/seeder"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type Module struct {
	Controller *controller.ReportController
	client     *clientImpl
	db         *gorm.DB
	log        *logrus.Logger
}

func New(db *gorm.DB, log *logrus.Logger, validate *validator.Validate, config *viper.Viper) *Module {
	reportRepo := repository.NewReportRepository(log)
	categoryRepo := repository.NewCategoryRepository(log)
	reportUseCase := usecase.NewReportUseCase(db, log, validate, reportRepo)
	reportController := controller.NewReportController(reportUseCase, log)

	return &Module{
		Controller: reportController,
		client:     &clientImpl{db: db, reportRepository: reportRepo, categoryRepository: categoryRepo},
		db:         db,
		log:        log,
	}
}

func (m *Module) Migrate() error {
	if err := m.db.AutoMigrate(
		&entity.Category{},
		&entity.Reporter{},
		&entity.Report{},
		&entity.ReportPhoto{},
		&entity.ReportConfirmation{},
		&entity.DuplicateReport{},
	); err != nil {
		return err
	}

	s := seeder.NewCategorySeeder(m.db, m.log)
	return s.SeedIfEmpty()
}

func (m *Module) Client() report_client.Client {
	return m.client
}
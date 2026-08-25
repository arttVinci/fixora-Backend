package report

import (
	region_client "github.com/arttVinci/fixora-Backend/internal/modules/region-client"
	report_client "github.com/arttVinci/fixora-Backend/internal/modules/report-client"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/controller"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/repository"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/seeder"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/usecase"
	verification_client "github.com/arttVinci/fixora-Backend/internal/modules/verification-client"
	"github.com/arttVinci/fixora-Backend/internal/shared/client"
	"github.com/go-playground/validator/v10"
	"github.com/google/generative-ai-go/genai"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type Module struct {
	Controller *controller.ReportController
	UseCase    *usecase.ReportUseCase
	client     *clientImpl
	db         *gorm.DB
	log        *logrus.Logger
}

func New(
	db *gorm.DB,
	log *logrus.Logger,
	validate *validator.Validate,
	config *viper.Viper,
	genaiClient *genai.Client,
	regionClient region_client.Client,
) *Module {
	reportRepo := repository.NewReportRepository(log)
	categoryRepo := repository.NewCategoryRepository(log)
	duplicateRepo := repository.NewDuplicateReportRepository(log)
	reporterRepo := repository.NewReporterRepository(log)
	photoRepo := repository.NewReportPhotoRepository(log)

	nominatimClient := client.NewNominatimClient(log)
	analyzePhotoUseCase := usecase.NewAnalyzePhotoUseCase(db, log, validate, genaiClient)

	duplicateUseCase := usecase.NewDuplicateUseCase(db, log, validate, reportRepo, duplicateRepo, genaiClient)

	clientImpl := &clientImpl{
		db:                 db,
		reportRepository:   reportRepo,
		categoryRepository: categoryRepo,
		duplicateUseCase:   duplicateUseCase,
	}

	reportUseCase := usecase.NewReportUseCase(
		db, log, validate,
		reportRepo, reporterRepo, photoRepo,
		clientImpl,
		regionClient,
		nominatimClient,
	)

	reportController := controller.NewReportController(reportUseCase, analyzePhotoUseCase, log)

	return &Module{
		Controller: reportController,
		client:     clientImpl,
		db:         db,
		log:        log,
	}
}

func (m *Module) SetVerificationClient(v verification_client.Client) {
	m.UseCase.VerificationClient = v
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

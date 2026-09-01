package report

import (
	region_client "github.com/arttVinci/fixora-Backend/internal/modules/region-client"
	report_client "github.com/arttVinci/fixora-Backend/internal/modules/report-client"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/controller"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/repository"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/seeder"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/usecase"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/worker"
	verification_client "github.com/arttVinci/fixora-Backend/internal/modules/verification-client"
	"github.com/arttVinci/fixora-Backend/internal/shared/client"
	sharedconfig "github.com/arttVinci/fixora-Backend/internal/shared/config"
	"github.com/go-playground/validator/v10"
	"github.com/google/generative-ai-go/genai"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"time"
)

type Module struct {
	Controller         *controller.ReportController
	CategoryController *controller.CategoryController
	UseCase            *usecase.ReportUseCase
	cleanupWorker      *worker.StagingCleanupWorker
	client             *clientImpl
	db                 *gorm.DB
	log                *logrus.Logger
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

	cloudinarySDK, _ := sharedconfig.NewCloudinary(config)
	cloudinaryClient := client.NewCloudinaryClient(cloudinarySDK, log)
	nominatimClient := client.NewNominatimClient(log)

	ttlHours := config.GetInt("cloudinary.staging_ttl_hours")
	if ttlHours <= 0 {
		ttlHours = 24
	}
	cleanupWorker := worker.NewStagingCleanupWorker(log, cloudinaryClient, time.Duration(ttlHours)*time.Hour)

	analyzePhotoUseCase := usecase.NewAnalyzePhotoUseCase(db, log, validate, genaiClient, cloudinaryClient, categoryRepo)

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
		cloudinaryClient,
	)

	reportController := controller.NewReportController(reportUseCase, analyzePhotoUseCase, log)
	categoryUseCase := usecase.NewCategoryUseCase(db, log, categoryRepo)
	categoryController := controller.NewCategoryController(categoryUseCase, log)

	return &Module{
		Controller:         reportController,
		CategoryController: categoryController,
		UseCase:            reportUseCase,
		cleanupWorker:      cleanupWorker,
		client:             clientImpl,
		db:                 db,
		log:                log,
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

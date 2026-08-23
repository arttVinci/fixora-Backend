package verification

import (
	report_client "github.com/arttVinci/fixora-Backend/internal/modules/report-client"
	verification_client "github.com/arttVinci/fixora-Backend/internal/modules/verification-client"
	"github.com/arttVinci/fixora-Backend/internal/modules/verification/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/verification/src/repository"
	"github.com/arttVinci/fixora-Backend/internal/modules/verification/src/usecase"
	"github.com/arttVinci/fixora-Backend/internal/modules/verification/src/worker"
	"github.com/arttVinci/fixora-Backend/internal/shared/client"
	"github.com/arttVinci/fixora-Backend/internal/shared/config"
	"github.com/go-playground/validator/v10"
	"github.com/google/generative-ai-go/genai"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type Module struct {
	db      *gorm.DB
	log     *logrus.Logger
	UseCase *usecase.VerificationUseCase
	worker  *worker.VerificationWorker
	client  *clientImpl
}

func New(db *gorm.DB, log *logrus.Logger, validate *validator.Validate, viperConfig *viper.Viper, genaiClient *genai.Client, reportClient report_client.Client) *Module {
	sr := repository.NewVerificationSessionRepository(log)
	lr := repository.NewVerificationLogRepository(log)

	llmProvider := config.NewLLMProvider(viperConfig, log)
	var llmClient *client.LLMClient
	if llmProvider != nil {
		llmClient = client.NewLLMClient(*llmProvider, log)
	}

	uc := usecase.NewVerificationUseCase(db, log, validate, sr, lr, reportClient, llmClient)
	return &Module{db: db, log: log, UseCase: uc, worker: worker.NewVerificationWorker(log, uc), client: &clientImpl{useCase: uc}}
}
func (m *Module) Migrate() error {
	return m.db.AutoMigrate(&entity.VerificationSession{}, &entity.VerificationLog{})
}
func (m *Module) Client() verification_client.Client { return m.client }


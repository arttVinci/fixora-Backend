package crawl

import (
	crawl_client "github.com/arttVinci/fixora-Backend/internal/modules/crawl-client"
	"github.com/arttVinci/fixora-Backend/internal/modules/crawl/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/crawl/src/infra"
	"github.com/arttVinci/fixora-Backend/internal/modules/crawl/src/repository"
	"github.com/arttVinci/fixora-Backend/internal/modules/crawl/src/usecase"
	"github.com/arttVinci/fixora-Backend/internal/modules/crawl/src/worker"
	region_client "github.com/arttVinci/fixora-Backend/internal/modules/region-client"
	report_client "github.com/arttVinci/fixora-Backend/internal/modules/report-client"
	"github.com/arttVinci/fixora-Backend/internal/shared/client"
	"github.com/go-playground/validator/v10"
	"github.com/google/generative-ai-go/genai"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type Module struct {
	client *clientImpl
	db     *gorm.DB
	worker *worker.CrawlerWorker
}

func New(
	db *gorm.DB,
	log *logrus.Logger,
	validate *validator.Validate,
	config *viper.Viper,
	genai *genai.Client,
	reportClient report_client.Client,
	regionClient region_client.Client,
) *Module {
	crawledRepo := repository.NewCrawledRepository(log)
	crawlerUseCase := usecase.NewCrawlerUseCase(db, log, validate, crawledRepo, reportClient, regionClient)

	rssClient := infra.NewRssClient(log)

	llmClient := infra.NewLlmClient(log, genai)

	nominatimClient := client.NewNominatimClient(log)

	crawlerWorker := worker.NewCrawlerWorker(log, rssClient, llmClient, nominatimClient, crawlerUseCase, reportClient, regionClient)

	return &Module{
		worker: crawlerWorker,
		client: &clientImpl{},
		db:     db,
	}
}

func (m *Module) Client() crawl_client.Client {
	return m.client
}

func (m *Module) Migrate() error {
	return m.db.AutoMigrate(&entity.CrawledArticle{})
}

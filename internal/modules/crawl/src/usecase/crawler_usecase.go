package usecase

import (
	"context"
	"time"

	"github.com/arttVinci/fixora-Backend/internal/modules/crawl/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/crawl/src/infra"
	"github.com/arttVinci/fixora-Backend/internal/modules/crawl/src/model"
	"github.com/arttVinci/fixora-Backend/internal/modules/crawl/src/repository"
	report_client "github.com/arttVinci/fixora-Backend/internal/modules/report-client"
	region_client "github.com/arttVinci/fixora-Backend/internal/modules/region-client"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type CrawlerUseCase struct {
	DB                       *gorm.DB
	Log                      *logrus.Logger
	Validate                 *validator.Validate
	CrawledArticleRepository *repository.CrawledRepository
	ReportClient             report_client.Client
	RegionClient             region_client.Client
}

func NewCrawlerUseCase(
	db *gorm.DB,
	log *logrus.Logger,
	validate *validator.Validate,
	crawledRepo *repository.CrawledRepository,
	reportClient report_client.Client,
	regionClient region_client.Client,
) *CrawlerUseCase {
	return &CrawlerUseCase{
		DB:                       db,
		Log:                      log,
		Validate:                 validate,
		CrawledArticleRepository: crawledRepo,
		ReportClient:             reportClient,
		RegionClient:             regionClient,
	}
}

func (c *CrawlerUseCase) IsArticleProcessed(ctx context.Context, url string) bool {
	article := new(entity.CrawledArticle)
	err := c.CrawledArticleRepository.FindByURL(c.DB.WithContext(ctx), article, url)
	return err == nil && article.ID != ""
}

func (c *CrawlerUseCase) SaveCrawledReport(ctx context.Context, req *model.ProcessCrawledArticleRequest) error {
	tx := c.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	articleID := infra.GenerateArticleID(req.SourceName)
	content := req.Content
	crawled := &entity.CrawledArticle{
		ID:          articleID,
		URL:         req.URL,
		Title:       req.Title,
		Content:     &content,
		SourceName:  req.SourceName,
		PublishedAt: &req.PublishedAt,
		CrawledAt:   &req.CrawledAt,
		Status:      "processed",
	}

	if err := c.CrawledArticleRepository.Create(tx, crawled); err != nil {
		c.Log.Warnf("Failed to create crawled article : %+v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan artikel hasil crawl")
	}

	address := req.Address
	description := req.Content
	report := &report_client.ReportClientRequest{
		ID:              infra.GenerateReportID(req.CategorySlug),
		CategoryID:      req.CategoryID,
		VillageID:       req.VillageID,
		Title:           req.Title,
		Description:     &description,
		Latitude:        req.Latitude,
		Longitude:       req.Longitude,
		Address:         &address,
		Severity:        req.Severity,
		Status:          "pending_verification",
		SourceType:      "ai_news",
		ConfidenceScore: 0.8,
		FirstReportedAt: &req.PublishedAt,
	}

	reportRes, err := c.ReportClient.CreateReport(tx, report)
	if err != nil {
		c.Log.Warnf("Failed to create auto-report via client : %+v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat laporan otomatis")
	}

	if err := c.CrawledArticleRepository.UpdateStatusAndReportID(tx, crawled.ID, "processed", reportRes.ID); err != nil {
		c.Log.Warnf("Failed to update crawled article relation : %+v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengaitkan artikel dengan laporan")
	}

	if err := tx.Commit().Error; err != nil {
		c.Log.Warnf("Failed commit transaction : %+v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan data sistem crawler")
	}

	return nil
}

func (c *CrawlerUseCase) SaveRejectedArticle(ctx context.Context, url, title, content, sourceName, reason string, publishedAt, crawledAt time.Time) error {
	tx := c.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	article := &entity.CrawledArticle{
		ID:           infra.GenerateArticleID(sourceName),
		URL:          url,
		Title:        title,
		Content:      &content,
		SourceName:   sourceName,
		RejectReason: &reason,
		PublishedAt:  &publishedAt,
		CrawledAt:    &crawledAt,
	}

	if err := c.CrawledArticleRepository.SaveRejected(tx, article); err != nil {
		c.Log.Warnf("Failed to save rejected crawled article : %+v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan artikel yang ditolak")
	}

	if err := tx.Commit().Error; err != nil {
		c.Log.Warnf("Failed commit transaction : %+v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan artikel yang ditolak")
	}

	return nil
}

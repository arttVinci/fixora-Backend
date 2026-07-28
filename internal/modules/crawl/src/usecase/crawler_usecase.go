package usecase

import (
	"context"

	crawl_entity "github.com/arttVinci/fixora-Backend/internal/modules/crawl/src/entity"
	crawl_model "github.com/arttVinci/fixora-Backend/internal/modules/crawl/src/model"
	crawl_repo "github.com/arttVinci/fixora-Backend/internal/modules/crawl/src/repository"
	report_client "github.com/arttVinci/fixora-Backend/internal/modules/report/client"
	report_entity "github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type CrawlerUseCase struct {
	DB                       *gorm.DB
	Log                      *logrus.Logger
	Validate                 *validator.Validate
	CrawledArticleRepository *crawl_repo.CrawledRepository
	ReportClient             report_client.ReportClient
}

func NewCrawlerUseCase(
	db *gorm.DB,
	log *logrus.Logger,
	validate *validator.Validate,
	crawledRepo *crawl_repo.CrawledRepository,
	reportClient report_client.ReportClient,
) *CrawlerUseCase {
	return &CrawlerUseCase{
		DB:                       db,
		Log:                      log,
		Validate:                 validate,
		CrawledArticleRepository: crawledRepo,
		ReportClient:             reportClient,
	}
}

func (c *CrawlerUseCase) IsArticleProcessed(ctx context.Context, url string) bool {
	article := new(crawl_entity.CrawledArticle)
	err := c.CrawledArticleRepository.FindByURL(c.DB.WithContext(ctx), article, url)
	return err == nil && article.ID != ""
}

func (c *CrawlerUseCase) SaveCrawledReport(ctx context.Context, req *crawl_model.ProcessCrawledArticleRequest) error {
	tx := c.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	content := req.Content
	crawled := &crawl_entity.CrawledArticle{
		URL:                 req.URL,
		Title:               req.Title,
		Content:             &content,
		SourceName:          req.SourceName,
		CrawledAt:           &req.CrawledAt,
		Status:              "processed",
		ExtractedLocation:   &req.Extraction.Location,
		ExtractedCategoryID: &req.Extraction.CategoryID,
		ExtractedSeverity:   &req.Extraction.Severity,
		ExtractedLatitude:   &req.Geocode.Latitude,
		ExtractedLongitude:  &req.Geocode.Longitude,
	}
	
	if err := c.CrawledArticleRepository.Create(tx, crawled); err != nil {
		c.Log.Warnf("Failed to create crawled article : %+v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan artikel hasil crawl")
	}

	address := req.Geocode.Address
	report := &report_entity.Report{
		CategoryID:      req.Extraction.CategoryID,
		VillageID:       req.Geocode.VillageID,
		Title:           req.Title,
		Description:     &content,
		Latitude:        req.Geocode.Latitude,
		Longitude:       req.Geocode.Longitude,
		Address:         &address,
		Severity:        req.Extraction.Severity,
		Status:          "pending_verification",
		SourceType:      "ai_news",
		ConfidenceScore: 0.8, // Default score AI news
		FirstReportedAt: &req.CrawledAt,
	}
	
	if err := c.ReportClient.CreateReport(tx, report); err != nil {
		c.Log.Warnf("Failed to create auto-report via client : %+v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat laporan otomatis")
	}

	crawled.ReportID = &report.ID
	if err := c.CrawledArticleRepository.UpdateStatusAndReportID(tx, crawled.ID, "processed", report.ID); err != nil {
		c.Log.Warnf("Failed to update crawled article relation : %+v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengaitkan artikel dengan laporan")
	}

	if err := tx.Commit().Error; err != nil {
		c.Log.Warnf("Failed commit transaction : %+v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan data sistem crawler")
	}

	return nil
}

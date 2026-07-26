package usecase

import (
	"context"

	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/model"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/repository"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type CrawlerUseCase struct {
	DB                       *gorm.DB
	Log                      *logrus.Logger
	Validate                 *validator.Validate
	CrawledArticleRepository *repository.CrawledArticleRepository
	ReportRepository         *repository.ReportRepository
}

func NewCrawlerUseCase(
	db *gorm.DB,
	log *logrus.Logger,
	validate *validator.Validate,
	crawledRepo *repository.CrawledArticleRepository,
	reportRepo *repository.ReportRepository,
) *CrawlerUseCase {
	return &CrawlerUseCase{
		DB:                       db,
		Log:                      log,
		Validate:                 validate,
		CrawledArticleRepository: crawledRepo,
		ReportRepository:         reportRepo,
	}
}

func (c *CrawlerUseCase) IsArticleProcessed(ctx context.Context, url string) bool {
	// Pengecekan cepat tanpa transaksi karena cuma 1 operasi read sederhana (Fase 2)
	article := new(entity.CrawledArticle)
	err := c.CrawledArticleRepository.FindByURL(c.DB.WithContext(ctx), article, url)
	return err == nil && article.ID != ""
}

func (c *CrawlerUseCase) SaveCrawledReport(ctx context.Context, req *model.ProcessCrawledArticleRequest) error {
	// TRANSAKSI WAJIB DIBUKA DI AWAL (Sesuai AGENTS.md 4.3)
	tx := c.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	// 1. Insert ke tabel crawled_articles
	content := req.Content
	crawled := &entity.CrawledArticle{
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

	// 2. Insert ke tabel reports
	address := req.Geocode.Address
	report := &entity.Report{
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
	
	if err := c.ReportRepository.Create(tx, report); err != nil {
		c.Log.Warnf("Failed to create auto-report : %+v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat laporan otomatis")
	}

	// Update foreign key di crawled_article
	crawled.ReportID = &report.ID
	if err := c.CrawledArticleRepository.UpdateStatusAndReportID(tx, crawled.ID, "processed", report.ID); err != nil {
		c.Log.Warnf("Failed to update crawled article relation : %+v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengaitkan artikel dengan laporan")
	}

	// Commit eksplisit
	if err := tx.Commit().Error; err != nil {
		c.Log.Warnf("Failed commit transaction : %+v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan data sistem crawler")
	}

	return nil
}

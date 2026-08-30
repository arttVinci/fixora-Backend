package repository

import (
	"time"

	"github.com/arttVinci/fixora-Backend/internal/modules/crawl/src/entity"
	shared_repo "github.com/arttVinci/fixora-Backend/internal/shared/repository"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type CrawledRepository struct {
	shared_repo.Repository[entity.CrawledArticle]
	Log *logrus.Logger
}

func NewCrawledRepository(log *logrus.Logger) *CrawledRepository {
	return &CrawledRepository{Log: log}
}

func (r *CrawledRepository) FindByURL(db *gorm.DB, article *entity.CrawledArticle, url string) error {
	// reject_reason = "llm_extraction_failed" bersifat transien (rate limit/
	// deadline), bukan keputusan bisnis final. Baris dengan reason ini dianggap
	// belum selesai diproses sehingga URL-nya tetap bisa di-retry.
	return db.Where("url = ? AND (reject_reason IS NULL OR reject_reason != ?)", url, "llm_extraction_failed").First(article).Error
}

// DeleteByURL menghapus baris lama untuk URL tertentu sebelum insert ulang.
// Dipakai saat me-retry artikel yang sebelumnya gagal transien (LLM) sehingga
// tidak bentrok dengan unique index pada kolom url.
func (r *CrawledRepository) DeleteByURL(db *gorm.DB, url string) error {
	return db.Where("url = ?", url).Delete(&entity.CrawledArticle{}).Error
}

func (r *CrawledRepository) UpdateStatusAndReportID(db *gorm.DB, id string, status string, reportID string) error {
	now := time.Now()
	return db.Model(&entity.CrawledArticle{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":       status,
		"report_id":    reportID,
		"processed_at": &now,
	}).Error
}

func (r *CrawledRepository) SaveRejected(db *gorm.DB, article *entity.CrawledArticle) error {
	article.Status = "rejected"
	now := time.Now()
	article.ProcessedAt = &now
	return db.Create(article).Error
}

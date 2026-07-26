package repository

import (
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	shared_repo "github.com/arttVinci/fixora-Backend/internal/shared/repository"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type CrawledArticleRepository struct {
	shared_repo.Repository[entity.CrawledArticle]
	Log *logrus.Logger
}

func NewCrawledArticleRepository(log *logrus.Logger) *CrawledArticleRepository {
	return &CrawledArticleRepository{Log: log}
}

// FindByURL checks if an article with the given URL already exists to prevent duplicate crawling (Fase 2).
func (r *CrawledArticleRepository) FindByURL(db *gorm.DB, item *entity.CrawledArticle, url string) error {
	return db.Where("url = ?", url).Take(item).Error
}

// UpdateStatusAndReportID updates the status of the crawled article and links it to the newly created report (Fase 6).
func (r *CrawledArticleRepository) UpdateStatusAndReportID(db *gorm.DB, id string, status string, reportID *string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if reportID != nil {
		updates["report_id"] = *reportID
	}
	return db.Model(&entity.CrawledArticle{}).Where("id = ?", id).Updates(updates).Error
}

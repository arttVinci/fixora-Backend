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
	return &CrawledArticleRepository{
		Log: log,
	}
}

func (r *CrawledArticleRepository) FindByURL(db *gorm.DB, article *entity.CrawledArticle, url string) error {
	return db.Where("url = ?", url).First(article).Error
}

func (r *CrawledArticleRepository) UpdateStatusAndReportID(db *gorm.DB, id string, status string, reportID string) error {
	return db.Model(&entity.CrawledArticle{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":    status,
		"report_id": reportID,
	}).Error
}

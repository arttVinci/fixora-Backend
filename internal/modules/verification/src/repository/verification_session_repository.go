package repository

import (
	"github.com/arttVinci/fixora-Backend/internal/modules/verification/src/entity"
	shared_repo "github.com/arttVinci/fixora-Backend/internal/shared/repository"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type VerificationSessionRepository struct {
	shared_repo.Repository[entity.VerificationSession]
	Log *logrus.Logger
}

func NewVerificationSessionRepository(log *logrus.Logger) *VerificationSessionRepository {
	return &VerificationSessionRepository{Log: log}
}

func (r *VerificationSessionRepository) FindActiveByReportID(db *gorm.DB, reportID string) (*entity.VerificationSession, error) {
	var s entity.VerificationSession
	err := db.Where("report_id = ? AND status IN ?", reportID, []string{"pending", "in_progress"}).Order("created_at desc").Take(&s).Error
	return &s, err
}

func (r *VerificationSessionRepository) FindByIDWithLogs(db *gorm.DB, dest *entity.VerificationSession, id string) error {
	return db.Preload("Logs").Where("id = ?", id).Take(dest).Error
}

func (r *VerificationSessionRepository) FindByReportIDWithLogs(db *gorm.DB, reportID string) ([]entity.VerificationSession, error) {
	var items []entity.VerificationSession
	err := db.Preload("Logs").Where("report_id = ?", reportID).Order("created_at desc").Find(&items).Error
	return items, err
}

func (r *VerificationSessionRepository) FindPending(db *gorm.DB, limit int) ([]entity.VerificationSession, error) {
	var items []entity.VerificationSession
	err := db.Where("status = ?", "pending").Order("created_at asc").Limit(limit).Find(&items).Error
	return items, err
}

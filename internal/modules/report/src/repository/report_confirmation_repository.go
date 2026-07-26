package repository

import (
	"time"

	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	shared_repo "github.com/arttVinci/fixora-Backend/internal/shared/repository"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ReportConfirmationRepository struct {
	shared_repo.Repository[entity.ReportConfirmation]
	Log *logrus.Logger
}

func NewReportConfirmationRepository(log *logrus.Logger) *ReportConfirmationRepository {
	return &ReportConfirmationRepository{Log: log}
}

func (r *ReportConfirmationRepository) HasConfirmedByIP(db *gorm.DB, reportID string, ip string) (bool, error) {
	var count int64
	yesterday := time.Now().Add(-24 * time.Hour)

	err := db.Model(&entity.ReportConfirmation{}).
		Where("report_id = ?", reportID).
		Where("confirmed_by_ip = ?", ip).
		Where("created_at > ?", yesterday).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

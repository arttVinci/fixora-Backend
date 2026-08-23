package repository

import (
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	shared_repo "github.com/arttVinci/fixora-Backend/internal/shared/repository"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type DuplicateReportRepository struct {
	shared_repo.Repository[entity.DuplicateReport]
	Log *logrus.Logger
}

func NewDuplicateReportRepository(log *logrus.Logger) *DuplicateReportRepository {
	return &DuplicateReportRepository{Log: log}
}

func (r *DuplicateReportRepository) FindByParentID(db *gorm.DB, parentID string) ([]entity.DuplicateReport, error) {
	var duplicates []entity.DuplicateReport
	err := db.Where("parent_id = ?", parentID).Find(&duplicates).Error
	return duplicates, err
}

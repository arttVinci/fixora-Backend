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

// ExistsByReportAndParent reports whether a similarity row already links the
// two reports, so the audit trail stays idempotent across re-checks.
func (r *DuplicateReportRepository) ExistsByReportAndParent(db *gorm.DB, reportID string, parentID string) (bool, error) {
	var count int64
	err := db.Model(&entity.DuplicateReport{}).
		Where("report_id = ? AND parent_id = ?", reportID, parentID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// FindRelatedReportIDs returns the distinct report IDs related to reportID from
// both directions: reports it duplicates (child → parent) and reports that
// duplicate it (parent → child). The input ID is excluded from the result.
func (r *DuplicateReportRepository) FindRelatedReportIDs(db *gorm.DB, reportID string) ([]string, error) {
	var ids []string
	err := db.Model(&entity.DuplicateReport{}).
		Select("parent_id AS id").
		Where("report_id = ?", reportID).
		Distinct().
		Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}

	var childIDs []string
	err = db.Model(&entity.DuplicateReport{}).
		Select("report_id AS id").
		Where("parent_id = ?", reportID).
		Distinct().
		Pluck("id", &childIDs).Error
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{reportID: true}
	for _, id := range childIDs {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}

	return ids, nil
}

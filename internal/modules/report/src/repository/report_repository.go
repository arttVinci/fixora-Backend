package repository

import (
	"time"

	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/model"
	shared_repo "github.com/arttVinci/fixora-Backend/internal/shared/repository"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ReportRepository struct {
	shared_repo.Repository[entity.Report]
	Log *logrus.Logger
}

func NewReportRepository(log *logrus.Logger) *ReportRepository {
	return &ReportRepository{Log: log}
}

func (r *ReportRepository) SearchMap(db *gorm.DB, request *model.SearchReportMapRequest) ([]entity.Report, error) {
	var items []entity.Report
	err := db.Scopes(r.FilterMap(request)).
		Preload("Category").
		Preload("Photos").
		Limit(500).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *ReportRepository) SearchList(db *gorm.DB, request *model.SearchReportListRequest) ([]entity.Report, int64, error) {
	var items []entity.Report
	query := db.Scopes(r.FilterList(request))

	switch request.Sort {
	case "newest":
		query = query.Order("first_reported_at desc")
	case "oldest":
		query = query.Order("first_reported_at asc")
	case "most_confirmed":
		query = query.Order("confidence_score desc")
	default:
		query = query.Order("created_at desc")
	}

	err := query.
		Offset((request.Page - 1) * request.Size).
		Limit(request.Size).
		Find(&items).Error
	if err != nil {
		return nil, 0, err
	}

	var total int64
	err = db.Model(&entity.Report{}).Scopes(r.FilterList(request)).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *ReportRepository) FindDetailByID(db *gorm.DB, item *entity.Report, id string) error {
	return db.
		Preload("Category").
		Preload("Photos").
		Preload("ReportConfirmations").
		Where("id = ?", id).
		Take(item).Error
}

func (r *ReportRepository) FilterMap(request *model.SearchReportMapRequest) func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		tx = tx.Where("latitude >= ? AND latitude <= ?", request.MinLat, request.MaxLat).
			Where("longitude >= ? AND longitude <= ?", request.MinLng, request.MaxLng)

		if request.CategoryID != "" {
			tx = tx.Where("category_id = ?", request.CategoryID)
		}
		if request.Status != "" {
			tx = tx.Where("status = ?", request.Status)
		}
		if request.Severity != "" {
			tx = tx.Where("severity = ?", request.Severity)
		}
		if request.SourceType != "" {
			tx = tx.Where("source_type = ?", request.SourceType)
		}

		tx = tx.Where("merged_into_id IS NULL")

		return tx
	}
}

func (r *ReportRepository) FilterList(request *model.SearchReportListRequest) func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		if request.Title != "" {
			tx = tx.Where("title LIKE ?", "%"+request.Title+"%")
		}
		if len(request.CategoryID) > 0 {
			tx = tx.Where("category_id IN ?", request.CategoryID)
		}
		if len(request.Status) > 0 {
			tx = tx.Where("status IN ?", request.Status)
		}
		if len(request.Severity) > 0 {
			tx = tx.Where("severity IN ?", request.Severity)
		}
		if len(request.SourceType) > 0 {
			tx = tx.Where("source_type IN ?", request.SourceType)
		}

		tx = tx.Where("merged_into_id IS NULL")

		return tx
	}
}

func (r *ReportRepository) FindMergedChildren(db *gorm.DB, parentID string) ([]entity.Report, error) {
	var items []entity.Report
	err := db.
		Preload("Category").
		Preload("Photos").
		Where("merged_into_id = ?", parentID).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

// FindParentCandidate returns the deterministic merge parent for a same-category
// report located within radiusMeters: the earliest non-merged report that is
// no newer than the candidate report itself (the parent is always the oldest
// report in the cluster, independent of check order). Proximity alone decides
// the merge; no lower time window is applied.
func (r *ReportRepository) FindParentCandidate(db *gorm.DB, lat, lng float64, radiusMeters float64, categoryID string, excludeID string, maxFirstReportedAt time.Time) (*entity.Report, error) {
	var item entity.Report
	// Numerically stable haversine (ASIN form): the ACOS form returns NULL for
	// identical coordinates because its argument can round above 1.0.
	haversine := "(6371000 * 2 * ASIN(SQRT(POWER(SIN(RADIANS(latitude - ?) / 2), 2) + COS(RADIANS(?)) * COS(RADIANS(latitude)) * POWER(SIN(RADIANS(longitude - ?) / 2), 2))))"

	err := db.
		Preload("Category").
		Preload("Photos").
		Where("category_id = ?", categoryID).
		Where("merged_into_id IS NULL").
		Where("id != ?", excludeID).
		Where("first_reported_at <= ?", maxFirstReportedAt).
		Where(haversine+" <= ?", lat, lat, lng, radiusMeters).
		Order("first_reported_at ASC, created_at ASC, id ASC").
		Take(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ReportRepository) FindNearby(db *gorm.DB, lat, lng float64, radiusMeters float64, excludeID string) ([]entity.Report, error) {
	var items []entity.Report
	// Numerically stable haversine (ASIN form): the ACOS form returns NULL for
	// identical coordinates because its argument can round above 1.0.
	haversine := "(6371000 * 2 * ASIN(SQRT(POWER(SIN(RADIANS(latitude - ?) / 2), 2) + COS(RADIANS(?)) * COS(RADIANS(latitude)) * POWER(SIN(RADIANS(longitude - ?) / 2), 2))))"

	err := db.
		Preload("Category").
		Preload("Photos").
		Where("merged_into_id IS NULL").
		Where("id != ?", excludeID).
		Where(haversine+" <= ?", lat, lat, lng, radiusMeters).
		Limit(20).
		Find(&items).Error

	if err != nil {
		return nil, err
	}
	return items, nil
}
func (r *ReportRepository) SetMergedInto(db *gorm.DB, reportID string, parentID string) error {
	return db.Model(&entity.Report{}).Where("id = ?", reportID).Update("merged_into_id", parentID).Error
}

func (r *ReportRepository) FindClientByID(db *gorm.DB, item *entity.Report, id string) error {
	return db.Preload("Category").Preload("Photos").Where("id = ?", id).Take(item).Error
}

func (r *ReportRepository) UpdateStatus(db *gorm.DB, reportID string, status string, rejectReason *string) error {
	return db.Model(&entity.Report{}).Where("id = ?", reportID).Updates(map[string]any{"status": status, "reject_reason": rejectReason}).Error
}

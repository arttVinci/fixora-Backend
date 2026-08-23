package report_client

import "gorm.io/gorm"

type Client interface {
	CreateReport(tx *gorm.DB, req *ReportClientRequest) (*ReportClientResponse, error)
	GetAllCategories(tx *gorm.DB) ([]CategoryClientResponse, error)
	GetCategoryBySlug(tx *gorm.DB, slug string) (*CategoryClientResponse, error)
	CheckDuplicate(tx *gorm.DB, reportID string) error
	UpdateReportStatus(tx *gorm.DB, reportID string, status string, rejectReason *string) error
	GetReportByID(tx *gorm.DB, reportID string) (*ReportClientResponse, error)
}

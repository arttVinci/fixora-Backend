package report_client

import (
	"gorm.io/gorm"
)




type Client interface {
	CreateReport(tx *gorm.DB, req *ReportClientRequest) (*ReportClientResponse, error)
}

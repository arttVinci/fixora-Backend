package report

import (
	report_client "github.com/arttVinci/fixora-Backend/internal/modules/report-client"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/repository"
	"gorm.io/gorm"
)

type clientImpl struct {
	db               *gorm.DB
	reportRepository *repository.ReportRepository
}

func (c *clientImpl) CreateReport(tx *gorm.DB, req *report_client.ReportClientRequest) (*report_client.ReportClientResponse, error) {
	report := &entity.Report{
		CategoryID:      req.CategoryID,
		VillageID:       req.VillageID,
		Title:           req.Title,
		Description:     req.Description,
		Latitude:        req.Latitude,
		Longitude:       req.Longitude,
		Address:         req.Address,
		Severity:        req.Severity,
		Status:          req.Status,
		SourceType:      req.SourceType,
		ConfidenceScore: req.ConfidenceScore,
		FirstReportedAt: req.FirstReportedAt,
	}

	if err := c.reportRepository.Create(tx, report); err != nil {
		return nil, err
	}

	return &report_client.ReportClientResponse{
		ID: report.ID,
	}, nil
}
package client

import (
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/repository"
	"gorm.io/gorm"
)

// ReportClient adalah interface publik untuk modul lain agar bisa berinteraksi dengan data Report
type ReportClient interface {
	CreateReport(tx *gorm.DB, report *entity.Report) error
}

type reportClientImpl struct {
	ReportRepository *repository.ReportRepository
}

func NewReportClient(reportRepo *repository.ReportRepository) ReportClient {
	return &reportClientImpl{
		ReportRepository: reportRepo,
	}
}

func (c *reportClientImpl) CreateReport(tx *gorm.DB, report *entity.Report) error {
	return c.ReportRepository.Create(tx, report)
}

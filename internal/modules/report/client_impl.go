package report

import (
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/repository"
	"gorm.io/gorm"
)

type clientImpl struct {
	db          *gorm.DB
	reportRepository *repository.ReportRepository
}

func (c *clientImpl) CreateReport(tx *gorm.DB, report *entity.Report) error {
	return c.reportRepository.Create(tx, report)
}
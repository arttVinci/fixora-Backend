package report

import (
	report_client "github.com/arttVinci/fixora-Backend/internal/modules/report-client"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/repository"
	"gorm.io/gorm"
)

type clientImpl struct {
	db                 *gorm.DB
	reportRepository   *repository.ReportRepository
	categoryRepository *repository.CategoryRepository
}

func (c *clientImpl) CreateReport(tx *gorm.DB, req *report_client.ReportClientRequest) (*report_client.ReportClientResponse, error) {
	report := &entity.Report{
		ID:              req.ID,
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

func (c *clientImpl) GetAllCategories(tx *gorm.DB) ([]report_client.CategoryClientResponse, error) {
	categories, err := c.categoryRepository.FindAll(tx)
	if err != nil {
		return nil, err
	}

	responses := make([]report_client.CategoryClientResponse, len(categories))
	for i, category := range categories {
		responses[i] = report_client.CategoryClientResponse{
			ID:   category.ID,
			Name: category.Name,
			Slug: category.Slug,
		}
	}

	return responses, nil
}

func (c *clientImpl) GetCategoryBySlug(tx *gorm.DB, slug string) (*report_client.CategoryClientResponse, error) {
	category := new(entity.Category)
	if err := c.categoryRepository.FindBySlug(tx, category, slug); err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &report_client.CategoryClientResponse{
		ID:   category.ID,
		Name: category.Name,
		Slug: category.Slug,
	}, nil
}

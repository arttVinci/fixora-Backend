package report

import (
	report_client "github.com/arttVinci/fixora-Backend/internal/modules/report-client"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/repository"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/usecase"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type clientImpl struct {
	db                 *gorm.DB
	reportRepository   *repository.ReportRepository
	categoryRepository *repository.CategoryRepository
	duplicateUseCase   *usecase.DuplicateUseCase
}

func (c *clientImpl) CreateReport(tx *gorm.DB, req *report_client.ReportClientRequest) (*report_client.ReportClientResponse, error) {
	if req.ID == "" {
		req.ID = uuid.NewString()
	}

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
			ID:             category.ID,
			Name:           category.Name,
			Slug:           category.Slug,
			SearchKeywords: category.SearchKeywords,
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

func (c *clientImpl) CheckDuplicate(tx *gorm.DB, reportID string) error {
	ctx := tx.Statement.Context
	return c.duplicateUseCase.CheckDuplicate(ctx, reportID)
}

func (c *clientImpl) UpdateReportStatus(tx *gorm.DB, reportID string, status string, rejectReason *string) error {
	return c.reportRepository.UpdateStatus(tx, reportID, status, rejectReason)
}

func (c *clientImpl) GetReportByID(tx *gorm.DB, reportID string) (*report_client.ReportClientResponse, error) {
	report := new(entity.Report)
	if err := c.reportRepository.FindClientByID(tx, report, reportID); err != nil {
		return nil, err
	}
	description := ""
	if report.Description != nil {
		description = *report.Description
	}
	address := ""
	if report.Address != nil {
		address = *report.Address
	}
	categorySlug := ""
	categoryName := ""
	if report.Category != nil {
		categorySlug = report.Category.Slug
		categoryName = report.Category.Name
	}
	photoURL := ""
	for _, photo := range report.Photos {
		if photo.IsPrimary {
			photoURL = photo.PhotoURL
			break
		}
	}
	if photoURL == "" && len(report.Photos) > 0 {
		photoURL = report.Photos[0].PhotoURL
	}
	return &report_client.ReportClientResponse{ID: report.ID, Title: report.Title, Description: description, Severity: report.Severity, CategorySlug: categorySlug, CategoryName: categoryName, SourceType: report.SourceType, PrimaryPhotoURL: photoURL, Address: address}, nil
}

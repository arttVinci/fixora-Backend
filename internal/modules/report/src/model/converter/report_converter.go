package converter

import (
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/model"
)

func ReportToMapResponse(report *entity.Report) *model.ReportMapResponse {
	if report == nil {
		return nil
	}

	categorySlug := ""
	if report.Category != nil {
		categorySlug = report.Category.Slug
	}

	primaryPhoto := ""
	for _, photo := range report.Photos {
		if photo.IsPrimary {
			primaryPhoto = photo.PhotoURL
			break
		}
	}

	return &model.ReportMapResponse{
		ID:              report.ID,
		Title:           report.Title,
		Latitude:        report.Latitude,
		Longitude:       report.Longitude,
		Severity:        report.Severity,
		CategorySlug:    categorySlug,
		Status:          report.Status,
		PhotoURL:        primaryPhoto,
		Source:          report.SourceType,
	}
}

func ReportToDetailResponse(report *entity.Report) *model.ReportDetailResponse {
	if report == nil {
		return nil
	}

	categoryName := ""
	categorySlug := ""
	if report.Category != nil {
		categoryName = report.Category.Name
		categorySlug = report.Category.Slug
	}

	primaryPhoto := ""
	additionalPhotos := make([]string, 0)
	for _, photo := range report.Photos {
		if photo.IsPrimary {
			primaryPhoto = photo.PhotoURL
		} else {
			additionalPhotos = append(additionalPhotos, photo.PhotoURL)
		}
	}

	totalConfirmations := int64(len(report.ReportConfirmations))

	return &model.ReportDetailResponse{
		ID:                 report.ID,
		Title:              report.Title,
		Description:        report.Description,
		Latitude:           report.Latitude,
		Longitude:          report.Longitude,
		Address:            report.Address,
		Severity:           report.Severity,
		Status:             report.Status,
		Source:             report.SourceType,
		SourceURL:          report.SourceURL,
		CategoryName:       categoryName,
		CategorySlug:       categorySlug,
		PhotoURL:           &primaryPhoto,
		AdditionalPhotos:   &additionalPhotos,
		TotalConfirmations: totalConfirmations,
		MergedIntoID:       report.MergedIntoID,
		FirstReportedAt:    report.FirstReportedAt,
		LastConfirmedAt:    report.LastConfirmedAt,
	}
}

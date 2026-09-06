package usecase

import (
	"context"
	"errors"
	"time"

	region_client "github.com/arttVinci/fixora-Backend/internal/modules/region-client"
	report_client "github.com/arttVinci/fixora-Backend/internal/modules/report-client"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/model"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/model/converter"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/repository"
	verification_client "github.com/arttVinci/fixora-Backend/internal/modules/verification-client"
	"github.com/arttVinci/fixora-Backend/internal/shared/client"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ReportUseCase struct {
	DB                        *gorm.DB
	Log                       *logrus.Logger
	Validate                  *validator.Validate
	ReportRepository          *repository.ReportRepository
	ReporterRepository        *repository.ReporterRepository
	ReportPhotoRepo           *repository.ReportPhotoRepository
	DuplicateReportRepository *repository.DuplicateReportRepository
	ReportClient              report_client.Client
	RegionClient              region_client.Client
	NominatimClient           client.NominatimClient
	Cloudinary                *client.CloudinaryClient
	VerificationClient        verification_client.Client
}

func NewReportUseCase(
	db *gorm.DB,
	log *logrus.Logger,
	validate *validator.Validate,
	reportRepo *repository.ReportRepository,
	reporterRepo *repository.ReporterRepository,
	photoRepo *repository.ReportPhotoRepository,
	duplicateRepo *repository.DuplicateReportRepository,
	reportClient report_client.Client,
	regionClient region_client.Client,
	nominatimClient client.NominatimClient,
	cloudinary *client.CloudinaryClient,
) *ReportUseCase {
	return &ReportUseCase{
		DB:                        db,
		Log:                       log,
		Validate:                  validate,
		ReportRepository:          reportRepo,
		ReporterRepository:        reporterRepo,
		ReportPhotoRepo:           photoRepo,
		DuplicateReportRepository: duplicateRepo,
		ReportClient:              reportClient,
		RegionClient:              regionClient,
		NominatimClient:           nominatimClient,
		Cloudinary:                cloudinary,
	}
}

func (c *ReportUseCase) GetDetail(ctx context.Context, id string) (*model.ReportDetailResponse, error) {
	tx := c.DB.WithContext(ctx)

	report := new(entity.Report)
	if err := c.ReportRepository.FindDetailByID(tx, report, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.Log.Warnf("Report not found : %+v", err)
			return nil, fiber.NewError(fiber.StatusNotFound, "Laporan tidak ditemukan")
		}
		c.Log.Warnf("Failed to get report detail : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data laporan")
	}

	resp := converter.ReportToDetailResponse(report)
	resp.RelatedReports = c.collectRelatedReports(tx, report)
	return resp, nil
}

// collectRelatedReports returns reports related to the given report, sourced
// from two places:
//  1. The merge cluster: merged children, the parent (if merged), and siblings.
//  2. The similarity audit trail (duplicate_reports), read in both directions —
//     reports this one duplicates, and reports that duplicate this one.
//
// Results are de-duplicated and exclude the report itself.
func (c *ReportUseCase) collectRelatedReports(tx *gorm.DB, report *entity.Report) []model.ReportMapResponse {
	seen := map[string]bool{report.ID: true}
	var related []entity.Report
	appendUnique := func(items []entity.Report) {
		for _, item := range items {
			if !seen[item.ID] {
				seen[item.ID] = true
				related = append(related, item)
			}
		}
	}

	// 1. Children merged into this report (this report is the parent).
	if children, err := c.ReportRepository.FindMergedChildren(tx, report.ID); err == nil {
		appendUnique(children)
	} else {
		c.Log.Warnf("Failed to load merged children for %s: %+v", report.ID, err)
	}

	// 1b. Reports merged into this report's own merged children (multi-level
	// merge chains), so every child still exposes its subtree.
	for _, child := range related {
		if child.MergedIntoID != nil && *child.MergedIntoID == report.ID {
			if grandChildren, err := c.ReportRepository.FindMergedChildren(tx, child.ID); err == nil {
				appendUnique(grandChildren)
			} else {
				c.Log.Warnf("Failed to load grandchildren for %s: %+v", child.ID, err)
			}
		}
	}

	// 2. If this report is itself merged, include the parent and its other children.
	if report.MergedIntoID != nil && *report.MergedIntoID != "" {
		parent := new(entity.Report)
		if err := c.ReportRepository.FindDetailByID(tx, parent, *report.MergedIntoID); err == nil {
			appendUnique([]entity.Report{*parent})
		} else {
			c.Log.Warnf("Failed to load parent %s for %s: %+v", *report.MergedIntoID, report.ID, err)
		}
		if siblings, err := c.ReportRepository.FindMergedChildren(tx, *report.MergedIntoID); err == nil {
			appendUnique(siblings)
		}
	}

	// 3. Similarity audit trail, both directions (related without being merged).
	if relatedIDs, err := c.DuplicateReportRepository.FindRelatedReportIDs(tx, report.ID); err == nil {
		if relatedReports, err := c.ReportRepository.FindRelatedByIDs(tx, relatedIDs); err == nil {
			appendUnique(relatedReports)
		} else {
			c.Log.Warnf("Failed to load related reports for %s: %+v", report.ID, err)
		}
	} else {
		c.Log.Warnf("Failed to load related report IDs for %s: %+v", report.ID, err)
	}

	responses := make([]model.ReportMapResponse, 0, len(related))
	for _, item := range related {
		if resp := converter.ReportToMapResponse(&item); resp != nil {
			responses = append(responses, *resp)
		}
	}
	return responses
}

func (c *ReportUseCase) SearchMap(ctx context.Context, request *model.SearchReportMapRequest) ([]model.ReportMapResponse, error) {
	if err := c.Validate.Struct(request); err != nil {
		c.Log.Warnf("Invalid request body : %+v", err)
		return nil, fiber.NewError(fiber.StatusBadRequest, "Format data request tidak valid")
	}

	tx := c.DB.WithContext(ctx)
	items, err := c.ReportRepository.SearchMap(tx, request)
	if err != nil {
		c.Log.Warnf("Failed to search map reports : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mencari data laporan")
	}

	responses := make([]model.ReportMapResponse, len(items))
	for i, item := range items {
		responses[i] = *converter.ReportToMapResponse(&item)
	}

	return responses, nil
}

func (c *ReportUseCase) CreateReport(ctx context.Context, request *model.CreateReportRequest) (*model.ReportDetailResponse, error) {
	if err := c.Validate.Struct(request); err != nil {
		c.Log.Warnf("Invalid request body : %+v", err)
		return nil, fiber.NewError(fiber.StatusBadRequest, "Format data request tidak valid")
	}

	tx := c.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	reverseResult, err := c.NominatimClient.ReverseGeocode(ctx, request.Latitude, request.Longitude)
	if err != nil {
		c.Log.Warnf("Failed to reverse geocode : %+v", err)
		return nil, fiber.NewError(fiber.StatusBadRequest, "Gagal mendeteksi lokasi dari koordinat, coba geser pin di peta")
	}

	village, err := c.RegionClient.ResolveVillageByAddress(tx, reverseResult.Village, reverseResult.District, reverseResult.City, reverseResult.Province)
	if err != nil || village == nil {
		c.Log.Warnf("Failed to resolve village : %+v", err)
		return nil, fiber.NewError(fiber.StatusBadRequest, "Lokasi tidak teridentifikasi, coba geser pin di peta")
	}

	reportID := uuid.NewString()

	if c.Cloudinary == nil {
		c.Log.Warnf("Cloudinary is not configured")
		return nil, fiber.NewError(fiber.StatusServiceUnavailable, "Penyimpanan foto belum dikonfigurasi")
	}

	promoted, err := c.Cloudinary.Promote(
		ctx,
		client.StagingPublicID(request.StagingSessionID, client.PrimarySlot),
		client.ReportsPublicID(reportID, client.PrimarySlot),
	)
	if err != nil {
		c.Log.Warnf("Failed to promote staged photo : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal memproses foto laporan")
	}

	var reporterIDPtr *string
	if request.ReporterEmail != "" {
		reporter := new(entity.Reporter)
		if err := c.ReporterRepository.FindByEmail(tx, reporter, request.ReporterEmail); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				reporter = &entity.Reporter{
					ID:    uuid.NewString(),
					Email: request.ReporterEmail,
				}
				if err := c.ReporterRepository.Create(tx, reporter); err != nil {
					c.Log.Warnf("Failed to create reporter : %+v", err)
					return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal memproses data pelapor")
				}
			} else {
				c.Log.Warnf("Failed to find reporter : %+v", err)
				return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal memproses data pelapor")
			}
		}
		reporterIDPtr = &reporter.ID
	}

	var descPtr *string
	if request.Description != "" {
		descPtr = &request.Description
	}

	now := time.Now()
	report := &entity.Report{
		ID:              reportID,
		ReporterID:      reporterIDPtr,
		CategoryID:      request.CategoryID,
		VillageID:       village.ID,
		Title:           request.Title,
		Description:     descPtr,
		Latitude:        request.Latitude,
		Longitude:       request.Longitude,
		Address:         &reverseResult.FullAddress,
		Severity:        request.Severity,
		Status:          "pending_verification",
		SourceType:      "user_report",
		ConfidenceScore: 1.0,
		FirstReportedAt: &now,
	}
	if err := c.ReportRepository.Create(tx, report); err != nil {
		c.Log.Warnf("Failed to create report : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat laporan")
	}

	photo := &entity.ReportPhoto{
		ID:        uuid.NewString(),
		ReportID:  reportID,
		PhotoURL:  promoted.SecureURL,
		IsPrimary: true,
	}
	if err := c.ReportPhotoRepo.Create(tx, photo); err != nil {
		c.Log.Warnf("Failed to create report photo : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan foto laporan")
	}

	if err := tx.Commit().Error; err != nil {
		c.Log.Warnf("Failed commit transaction : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat laporan")
	}

	ctxNoTx := c.DB.WithContext(ctx)
	if err := c.ReportClient.CheckDuplicate(ctxNoTx, reportID); err != nil {
		c.Log.Warnf("Duplicate check failed for report %s : %+v", reportID, err)
	}
	if c.VerificationClient != nil {
		if _, err := c.VerificationClient.CreateVerification(ctxNoTx, reportID); err != nil {
			c.Log.Warnf("Verification create failed for report %s : %+v", reportID, err)
		}
	}

	saved := new(entity.Report)
	if err := c.ReportRepository.FindDetailByID(ctxNoTx, saved, reportID); err != nil {
		c.Log.Warnf("Failed to reload report detail : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data laporan")
	}

	return converter.ReportToDetailResponse(saved), nil
}

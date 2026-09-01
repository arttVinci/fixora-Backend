package usecase

import (
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/repository"
	"github.com/corona10/goimagehash"
	"github.com/go-playground/validator/v10"
	"github.com/google/generative-ai-go/genai"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type DuplicateUseCase struct {
	DB                        *gorm.DB
	Log                       *logrus.Logger
	Validate                  *validator.Validate
	ReportRepository          *repository.ReportRepository
	DuplicateReportRepository *repository.DuplicateReportRepository
	Genai                     *genai.Client
}

func NewDuplicateUseCase(
	db *gorm.DB,
	log *logrus.Logger,
	validate *validator.Validate,
	reportRepo *repository.ReportRepository,
	duplicateRepo *repository.DuplicateReportRepository,
	genaiClient *genai.Client,
) *DuplicateUseCase {
	return &DuplicateUseCase{
		DB:                        db,
		Log:                       log,
		Validate:                  validate,
		ReportRepository:          reportRepo,
		DuplicateReportRepository: duplicateRepo,
		Genai:                     genaiClient,
	}
}

func (u *DuplicateUseCase) CheckDuplicate(ctx context.Context, reportID string) error {
	db := u.DB.WithContext(ctx)

	report := new(entity.Report)
	if err := u.ReportRepository.FindDetailByID(db, report, reportID); err != nil {
		u.Log.Warnf("Report not found for duplicate check: %+v", err)
		return err
	}

	if report.MergedIntoID != nil {
		u.Log.Infof("Report %s is already merged, skipping duplicate check", reportID)
		return nil
	}

	// Coarse filter: same category, non-merged, within 100 meters, excluding
	// itself. Ordered oldest-first so the merge parent is always the earliest
	// report in the cluster, independent of check order.
	candidates, err := u.ReportRepository.FindSameCategoryNearby(
		db,
		report.Latitude,
		report.Longitude,
		100.0,
		report.CategoryID,
		report.ID,
	)
	if err != nil {
		u.Log.Warnf("Failed to find nearby same-category candidates for report %s: %+v", reportID, err)
		return err
	}
	if len(candidates) == 0 {
		return nil
	}

	// Fine filter: perceptual hash of the primary photo.
	currentHash, _ := u.primaryPhotoHash(db, report)

	// 1. ALWAYS record every detected similarity to duplicate_reports,
	//    regardless of whether a merge happens.
	// 2. Merge ONLY when the photo is identical AND the source is the same.
	var mergeParent *entity.Report
	for i := range candidates {
		candidate := &candidates[i]

		reason := "nearby_location"
		score := u.proximityScore(report.Latitude, report.Longitude, candidate.Latitude, candidate.Longitude)
		identical := false

		if currentHash != nil {
			if candidateHash, err := u.photoHashFor(db, candidate); err == nil && candidateHash != nil {
				if dist, err := currentHash.Distance(candidateHash); err == nil && dist <= 5 {
					u.Log.Infof("pHash distance between %s and %s: %d", report.ID, candidate.ID, dist)
					reason = "identical_photo"
					score = 1.0 - (float64(dist) / 64.0)
					identical = true
				}
			}
		}

		if err := u.recordSimilarity(ctx, report.ID, candidate.ID, reason, score); err != nil {
			return err
		}

		if identical && report.SourceType == candidate.SourceType && mergeParent == nil {
			mergeParent = candidate
		}
	}

	if mergeParent == nil {
		u.Log.Infof("Report %s has %d related report(s) but no identical same-source match; no merge", reportID, len(candidates))
		return nil
	}

	u.Log.Infof("Report %s merged into %s (identical photo, same source %s)", reportID, mergeParent.ID, report.SourceType)

	if err := u.ReportRepository.SetMergedInto(db, reportID, mergeParent.ID); err != nil {
		u.Log.Warnf("Failed to set merged_into_id for report %s: %+v", reportID, err)
		return err
	}

	return nil
}

// recordSimilarity persists a similarity audit row idempotently, so re-running
// the duplicate check never duplicates the audit trail.
func (u *DuplicateUseCase) recordSimilarity(ctx context.Context, reportID, parentID, reason string, score float64) error {
	db := u.DB.WithContext(ctx)

	exists, err := u.DuplicateReportRepository.ExistsByReportAndParent(db, reportID, parentID)
	if err != nil {
		u.Log.Warnf("Failed to check existing duplicate entry: %+v", err)
		return err
	}
	if exists {
		return nil
	}

	entry := &entity.DuplicateReport{
		ID:              uuid.NewString(),
		ReportID:        reportID,
		ParentID:        parentID,
		Reason:          reason,
		SimilarityScore: score,
	}

	if err := u.DuplicateReportRepository.Create(db, entry); err != nil {
		u.Log.Warnf("Failed to create duplicate_reports entry: %+v", err)
		return err
	}

	return nil
}

// proximityScore converts a geodesic distance (meters) into a 0..1 similarity
// score: ~0.99 at 0m decaying to ~0.5 at the 100m radius boundary.
func (u *DuplicateUseCase) proximityScore(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusMeters = 6371000.0

	toRad := func(deg float64) float64 { return deg * math.Pi / 180.0 }

	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	dist := 2 * earthRadiusMeters * math.Asin(math.Sqrt(a))

	if dist > 100.0 {
		dist = 100.0
	}
	return 1.0 - (dist / 100.0 * 0.5)
}

// primaryPhotoHash returns the perceptual hash of the report's primary photo,
// computing and persisting it when missing. Returns (nil, nil) when the report
// has no primary photo or the hash cannot be computed.
func (u *DuplicateUseCase) primaryPhotoHash(db *gorm.DB, report *entity.Report) (*goimagehash.ImageHash, *entity.ReportPhoto) {
	var currentPhoto *entity.ReportPhoto

	for i := range report.Photos {
		if report.Photos[i].IsPrimary {
			currentPhoto = &report.Photos[i]
			break
		}
	}

	if currentPhoto == nil {
		return nil, nil
	}

	h, err := u.getOrComputePHash(db, currentPhoto)
	if err != nil {
		u.Log.Warnf("Failed to compute pHash for report %s photo: %+v", report.ID, err)
		return nil, nil
	}
	return h, currentPhoto
}

// photoHashFor returns the primary photo hash of an arbitrary report.
func (u *DuplicateUseCase) photoHashFor(db *gorm.DB, report *entity.Report) (*goimagehash.ImageHash, error) {
	for i := range report.Photos {
		if report.Photos[i].IsPrimary {
			return u.getOrComputePHash(db, &report.Photos[i])
		}
	}
	return nil, fmt.Errorf("report %s has no primary photo", report.ID)
}

func (u *DuplicateUseCase) getOrComputePHash(db *gorm.DB, photo *entity.ReportPhoto) (*goimagehash.ImageHash, error) {
	if photo.PerceptualHash != nil && *photo.PerceptualHash != "" {
		uval, err := strconv.ParseUint(*photo.PerceptualHash, 16, 64)
		if err == nil {
			return goimagehash.NewImageHash(uval, goimagehash.PHash), nil
		}
	}

	if !strings.HasPrefix(photo.PhotoURL, "https://") {
		return nil, fmt.Errorf("invalid photo URL scheme (must be https): %s", photo.PhotoURL)
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}
	resp, err := httpClient.Get(photo.PhotoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch photo URL %s: %w", photo.PhotoURL, err)
	}
	defer resp.Body.Close()

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to decode photo %s: %w", photo.PhotoURL, err)
	}

	hash, err := goimagehash.PerceptionHash(img)
	if err != nil {
		return nil, fmt.Errorf("failed to compute pHash: %w", err)
	}

	hashHex := fmt.Sprintf("%016x", hash.GetHash())
	photo.PerceptualHash = &hashHex
	_ = db.Model(&entity.ReportPhoto{}).Where("id = ?", photo.ID).Update("perceptual_hash", hashHex).Error

	return hash, nil
}

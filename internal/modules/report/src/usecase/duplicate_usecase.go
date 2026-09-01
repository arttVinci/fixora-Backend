package usecase

import (
	"context"
	"errors"
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

	// 1. Deterministic proximity merge: same category within 100 meters.
	//    Proximity alone decides the merge (no lower time window). The parent
	//    must be no newer than this report so it is always the oldest report
	//    in the cluster, independent of check order.
	maxFirstReportedAt := time.Now()
	if report.FirstReportedAt != nil {
		maxFirstReportedAt = *report.FirstReportedAt
	}

	parent, err := u.ReportRepository.FindParentCandidate(
		db,
		report.Latitude,
		report.Longitude,
		100.0, // 100 meters radius
		report.CategoryID,
		report.ID,
		maxFirstReportedAt,
	)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			u.Log.Warnf("Failed to find nearby parent candidate for report %s: %+v", reportID, err)
			return err
		}
		u.Log.Infof("No nearby same-category candidate within 100m found for report %s", reportID)
		return nil
	}

	u.Log.Infof("Found proximity parent %s within 100m for report %s (distance-based merge)", parent.ID, reportID)

	matchReason := "nearby_location"
	matchScore := u.proximityScore(report.Latitude, report.Longitude, parent.Latitude, parent.Longitude)

	// 2. If the proximity parent's photo is perceptually identical to this
	//    report's primary photo, upgrade the evidence to identical_photo.
	if currentHash, currentPhoto := u.primaryPhotoHash(db, report); currentHash != nil && currentPhoto != nil {
		if parentHash, err := u.photoHashFor(db, parent); err == nil && parentHash != nil {
			if dist, err := currentHash.Distance(parentHash); err == nil && dist <= 5 {
				u.Log.Infof("pHash distance between %s and proximity parent %s: %d", report.ID, parent.ID, dist)
				matchReason = "identical_photo"
				matchScore = 1.0 - (float64(dist) / 64.0)
			}
		}
	}

	// 3. Execute Merge
	u.Log.Infof("Report %s matched as duplicate of parent %s (reason: %s, score: %.2f)", reportID, parent.ID, matchReason, matchScore)

	tx := u.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := u.ReportRepository.SetMergedInto(tx, reportID, parent.ID); err != nil {
		u.Log.Warnf("Failed to set merged_into_id for report %s: %+v", reportID, err)
		return err
	}

	dupEntry := &entity.DuplicateReport{
		ID:              uuid.NewString(),
		ReportID:        reportID,
		ParentID:        parent.ID,
		Reason:          matchReason,
		SimilarityScore: matchScore,
	}

	if err := u.DuplicateReportRepository.Create(tx, dupEntry); err != nil {
		u.Log.Warnf("Failed to create duplicate_reports entry: %+v", err)
		return err
	}

	if err := tx.Commit().Error; err != nil {
		u.Log.Warnf("Failed commit duplicate merge transaction for report %s: %+v", reportID, err)
		return err
	}

	u.Log.Infof("Successfully merged report %s into parent %s", reportID, parent.ID)

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

package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
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

type llmDuplicateResult struct {
	IsDuplicate     bool    `json:"is_duplicate"`
	MatchedReportID string  `json:"matched_report_id"`
	Reason          string  `json:"reason"`
	Confidence      float64 `json:"confidence"`
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

	// 1. Get or compute perceptual hash for current report if it has primary photo
	var currentHash *goimagehash.ImageHash
	var currentPhoto *entity.ReportPhoto

	for i := range report.Photos {
		if report.Photos[i].IsPrimary {
			currentPhoto = &report.Photos[i]
			break
		}
	}

	if currentPhoto != nil {
		h, err := u.getOrComputePHash(db, currentPhoto)
		if err != nil {
			u.Log.Warnf("Failed to compute pHash for report %s photo: %+v", reportID, err)
		} else {
			currentHash = h
		}
	}

	// 2. Coarse Filter: Find candidates within 100 meters, same category, 7 days window
	since := time.Now().Add(-7 * 24 * time.Hour)
	if report.FirstReportedAt != nil {
		since = report.FirstReportedAt.Add(-7 * 24 * time.Hour)
	}

	candidates, err := u.ReportRepository.FindNearbyByCategory(
		db,
		report.Latitude,
		report.Longitude,
		100.0, // 100 meters radius
		report.CategoryID,
		report.ID,
		since,
	)
	if err != nil {
		u.Log.Warnf("Failed to find nearby candidates for report %s: %+v", reportID, err)
		return err
	}

	if len(candidates) == 0 {
		u.Log.Infof("No nearby candidates found for report %s", reportID)
		return nil
	}

	u.Log.Infof("Found %d coarse candidates for report %s", len(candidates), reportID)

	// 3. Fine Filter & Evaluation
	var matchedParentID string
	var matchReason string
	var matchScore float64

	var llmCandidates []entity.Report

	for _, cand := range candidates {
		var candPhoto *entity.ReportPhoto
		for i := range cand.Photos {
			if cand.Photos[i].IsPrimary {
				candPhoto = &cand.Photos[i]
				break
			}
		}

		if currentHash != nil && candPhoto != nil {
			candHash, err := u.getOrComputePHash(db, candPhoto)
			if err == nil && candHash != nil {
				dist, err := currentHash.Distance(candHash)
				if err == nil {
					u.Log.Infof("pHash distance between %s and candidate %s: %d", report.ID, cand.ID, dist)
					if dist <= 5 {
						matchedParentID = cand.ID
						matchReason = "identical_photo"
						matchScore = 1.0 - (float64(dist) / 64.0)
						break
					} else if dist > 15 {
						// Photos are clearly different, skip this candidate
						continue
					}
				}
			}
		}

		// Keep candidate for LLM / location-based evaluation
		llmCandidates = append(llmCandidates, cand)
	}

	// 4. LLM Semantic Evaluation if no exact photo match found yet
	if matchedParentID == "" && len(llmCandidates) > 0 && u.Genai != nil {
		llmMatch, err := u.evaluateWithLLM(ctx, report, llmCandidates)
		if err != nil {
			u.Log.Warnf("LLM duplicate evaluation failed for report %s: %+v", reportID, err)
		} else if llmMatch != nil && llmMatch.IsDuplicate && llmMatch.MatchedReportID != "" {
			matchedParentID = llmMatch.MatchedReportID
			matchReason = "similar_location"
			if llmMatch.Reason != "" {
				matchReason = llmMatch.Reason
			}
			matchScore = llmMatch.Confidence
			if matchScore <= 0 {
				matchScore = 0.85
			}
		}
	}

	// 5. Fallback: If LLM unavailable/skipped and candidates exist, do NOT auto-merge
	// without evidence. Log and skip to avoid incorrectly merging distinct reports.
	if matchedParentID == "" && len(llmCandidates) > 0 {
		u.Log.Infof("No confident duplicate match for report %s — %d candidates exist but no photo/LLM evidence, skipping merge", reportID, len(llmCandidates))
	}

	// 6. Execute Merge if match found
	if matchedParentID != "" {
		u.Log.Infof("Report %s matched as duplicate of parent %s (reason: %s, score: %.2f)", reportID, matchedParentID, matchReason, matchScore)

		tx := u.DB.WithContext(ctx).Begin()
		defer tx.Rollback()

		if err := u.ReportRepository.SetMergedInto(tx, reportID, matchedParentID); err != nil {
			u.Log.Warnf("Failed to set merged_into_id for report %s: %+v", reportID, err)
			return err
		}

		dupEntry := &entity.DuplicateReport{
			ID:              uuid.NewString(),
			ReportID:        reportID,
			ParentID:        matchedParentID,
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

		u.Log.Infof("Successfully merged report %s into parent %s", reportID, matchedParentID)
	}

	return nil
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

func (u *DuplicateUseCase) evaluateWithLLM(ctx context.Context, newReport *entity.Report, candidates []entity.Report) (*llmDuplicateResult, error) {
	model := u.Genai.GenerativeModel("gemini-3.5-flash")
	model.ResponseSchema = &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"is_duplicate": {
				Type:        genai.TypeBoolean,
				Description: "Apakah laporan baru mendeskripsikan masalah infrastruktur yang sama dengan salah satu laporan kandidat.",
			},
			"matched_report_id": {
				Type:        genai.TypeString,
				Description: "ID laporan kandidat yang cocok. Kosongkan jika is_duplicate false.",
			},
			"reason": {
				Type:        genai.TypeString,
				Description: "Alasan pencocokan: identical_photo, similar_location, atau same_incident.",
			},
			"confidence": {
				Type:        genai.TypeNumber,
				Description: "Skor keyakinan pencocokan antara 0.0 hingga 1.0.",
			},
		},
		Required: []string{"is_duplicate", "matched_report_id", "reason", "confidence"},
	}
	model.ResponseMIMEType = "application/json"

	newDesc := ""
	if newReport.Description != nil {
		newDesc = *newReport.Description
	}
	newAddr := ""
	if newReport.Address != nil {
		newAddr = *newReport.Address
	}

	candListJSON := []map[string]string{}
	for _, c := range candidates {
		desc := ""
		if c.Description != nil {
			desc = *c.Description
		}
		addr := ""
		if c.Address != nil {
			addr = *c.Address
		}
		candListJSON = append(candListJSON, map[string]string{
			"id":          c.ID,
			"title":       c.Title,
			"description": desc,
			"address":     addr,
			"severity":    c.Severity,
			"source_type": c.SourceType,
		})
	}

	candBytes, _ := json.Marshal(candListJSON)

	prompt := fmt.Sprintf(`Tugas kamu adalah mengevaluasi apakah laporan baru merupakan DUPLIKAT dari salah satu laporan kandidat yang berada di lokasi dan waktu berdekatan.

Laporan Baru:
ID: %s
Judul: %s
Deskripsi: %s
Alamat: %s
Sumber: %s

Daftar Laporan Kandidat:
%s

Panduan:
1. Bandingkan judul, deskripsi, dan alamat. Jika mereka merujuk pada kerusakan/insiden infrastruktur spesifik yang sama, set is_duplicate = true dan matched_report_id = ID kandidat.
2. Jika semua kandidat berbeda masalah/lokasi spesifiknya, set is_duplicate = false dan matched_report_id = "".
3. Balas JSON sesuai schema.`, newReport.ID, newReport.Title, newDesc, newAddr, newReport.SourceType, string(candBytes))

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from Gemini")
	}

	part := resp.Candidates[0].Content.Parts[0]
	text, ok := part.(genai.Text)
	if !ok {
		return nil, fmt.Errorf("unexpected LLM response format")
	}

	jsonStr := strings.TrimPrefix(string(text), "```json\n")
	jsonStr = strings.TrimSuffix(jsonStr, "\n```")
	jsonStr = strings.TrimSpace(jsonStr)

	var res llmDuplicateResult
	if err := json.Unmarshal([]byte(jsonStr), &res); err != nil {
		return nil, fmt.Errorf("failed to unmarshal LLM response: %w", err)
	}

	return &res, nil
}

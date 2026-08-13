package usecase

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	report_client "github.com/arttVinci/fixora-Backend/internal/modules/report-client"
	"github.com/arttVinci/fixora-Backend/internal/modules/verification/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/verification/src/model"
	"github.com/arttVinci/fixora-Backend/internal/modules/verification/src/model/converter"
	"github.com/arttVinci/fixora-Backend/internal/modules/verification/src/repository"
	"github.com/arttVinci/fixora-Backend/internal/modules/verification/src/utils"
	"github.com/arttVinci/fixora-Backend/internal/shared/client"
	"github.com/arttVinci/fixora-Backend/internal/shared/dto"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type VerificationUseCase struct {
	DB                            *gorm.DB
	Log                           *logrus.Logger
	Validate                      *validator.Validate
	VerificationSessionRepository *repository.VerificationSessionRepository
	VerificationLogRepository     *repository.VerificationLogRepository
	ReportClient                  report_client.Client
	LLMClient                     client.LLMClient
}

func NewVerificationUseCase(
	db *gorm.DB, 
	log *logrus.Logger, 
	validate *validator.Validate, 
	verificationSessionRepository *repository.VerificationSessionRepository, 
	verificationLogRepository *repository.VerificationLogRepository, 
	reportClient report_client.Client, 
	llmClient client.LLMClient,
) *VerificationUseCase {
	return &VerificationUseCase{
		DB:                            db,
		Log:                           log,
		Validate:                      validate,
		VerificationSessionRepository: verificationSessionRepository,
		VerificationLogRepository:     verificationLogRepository,
		ReportClient:                  reportClient,
		LLMClient:                     llmClient,
	}
}

func (c *VerificationUseCase) CreateVerification(ctx context.Context, reportID string) (*model.VerificationSessionResponse, error) {
	tx := c.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if VerificationSessionEntity, err := c.VerificationSessionRepository.FindActiveByReportID(tx, reportID); err == nil {
		return converter.VerificationSessionToResponse(VerificationSessionEntity), nil
	}

	VerificationSessionEntity := &entity.VerificationSession{
		ID:         uuid.NewString(),
		ReportID:   reportID,
		Status:     "pending",
	}

	if err := c.VerificationSessionRepository.Create(tx, VerificationSessionEntity); err != nil {
		c.Log.Warnf("Failed create verification session : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat sesi verifikasi")
	}

	if err := tx.Commit().Error; err != nil {
		c.Log.Warnf("Failed commit transaction : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat sesi verifikasi")
	}

	return converter.VerificationSessionToResponse(VerificationSessionEntity), nil
}

func (c *VerificationUseCase) RetrySession(ctx context.Context, sessionID string) (*model.VerificationSessionResponse, error) {
	tx := c.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	VerificationSessionEntity := new(entity.VerificationSession)
	
	if err := c.VerificationSessionRepository.FindById(tx, VerificationSessionEntity, sessionID); err != nil {
		c.Log.Warnf("Verification session not found : %+v", err)
		return nil, fiber.NewError(fiber.StatusNotFound, "Sesi verifikasi tidak ditemukan")
	}

	if VerificationSessionEntity.Status != "error" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Hanya sesi error yang dapat diulang")
	}

	VerificationSessionEntity.Status = "pending"
	VerificationSessionEntity.FinalVerdict = nil
	VerificationSessionEntity.FinalReasoning = nil
	VerificationSessionEntity.RejectReason = nil
	VerificationSessionEntity.DecidedBy = nil
	VerificationSessionEntity.StartedAt = nil
	VerificationSessionEntity.CompletedAt = nil

	if err := c.VerificationSessionRepository.Update(tx, VerificationSessionEntity); err != nil {
		c.Log.Warnf("Failed update verification session : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui sesi verifikasi")
	}

	if err := tx.Commit().Error; err != nil {
		c.Log.Warnf("Failed commit transaction : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui sesi verifikasi")
	}

	return converter.VerificationSessionToResponse(VerificationSessionEntity), nil
}

func (c *VerificationUseCase) GetSessionsByReportID(ctx context.Context, reportID string) ([]model.VerificationSessionResponse, error) {
	items, err := c.VerificationSessionRepository.FindByReportIDWithLogs(c.DB.WithContext(ctx), reportID)

	if err != nil {
		c.Log.Warnf("Failed get verification sessions : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil sesi verifikasi")
	}

	res := make([]model.VerificationSessionResponse, len(items))
	for i := range items {
		res[i] = *converter.VerificationSessionToResponse(&items[i])
	}

	return res, nil
}

func (c *VerificationUseCase) RunVerification(ctx context.Context, sessionID string) error {
	tx := c.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	VerificationSessionEntity := new(entity.VerificationSession)

	if err := c.VerificationSessionRepository.FindById(tx, VerificationSessionEntity, sessionID); err != nil {
		c.Log.Warnf("Verification session not found : %+v", err)
		return fiber.NewError(fiber.StatusNotFound, "Sesi verifikasi tidak ditemukan")
	}

	report, err := c.ReportClient.GetReportByID(tx, VerificationSessionEntity.ReportID)
	if err != nil {
		c.Log.Warnf("Report not found : %+v", err)
		return c.markError(tx, VerificationSessionEntity, err.Error())
	}

	now := time.Now()
	if report.SourceType == "gov_data" {
		verdict := true
		reason := "gov_data"

		VerificationSessionEntity.Status = "approved"
		VerificationSessionEntity.FinalVerdict = &verdict
		VerificationSessionEntity.SkipReason = &reason
		VerificationSessionEntity.DecidedBy = &reason
		VerificationSessionEntity.StartedAt = &now
		VerificationSessionEntity.CompletedAt = &now

		if err := c.ReportClient.UpdateReportStatus(tx, VerificationSessionEntity.ReportID, "verified", nil); err != nil {
			c.Log.Warnf("Failed update report status : %+v", err)
			return err
		}

		if err := c.VerificationSessionRepository.Update(tx, VerificationSessionEntity); err != nil {
			c.Log.Warnf("Failed update verification session : %+v", err)
			return err
		}

		return tx.Commit().Error
	}

	VerificationSessionEntity.Status = "in_progress"
	VerificationSessionEntity.StartedAt = &now

	if err := c.VerificationSessionRepository.Update(tx, VerificationSessionEntity); err != nil {
		c.Log.Warnf("Failed update verification session : %+v", err)
		return err
	}

	req := &model.VerificationRequest{
		ReportTitle:       report.Title,
		ReportDescription: report.Description,
		ReportSeverity:    report.Severity,
		ReportCategory:    report.CategorySlug,
		ReportSourceType:  report.SourceType,
		ReportPhotoURL:    report.PrimaryPhotoURL,
		ReportAddress:     report.Address,
	}

	advocateReq := &dto.LLMGenerateContentRequest{
		Content:  utils.RolePrompt("advocate", req),
		Provider: "CommandCode",
		Model:    "qwen/qwen3.7-flash",
	}

	advocateAgent, err := c.runAgent(ctx, VerificationSessionEntity.ID, "advocate", advocateReq)
	if err != nil {
		return c.markError(tx, VerificationSessionEntity, "advocate_failed")
	}

	skepticReq := &dto.LLMGenerateContentRequest{
		Content:  utils.RolePrompt("skeptic", req),
		Provider: "CommandCode",
		Model:    "qwen/qwen3.7-flash",
	}

	skepticAgent, err := c.runAgent(ctx, VerificationSessionEntity.ID, "skeptic", skepticReq)
	if err != nil {
		if advocateAgent.Confidence >= 0.9 && advocateAgent.Verdict {
			return c.finalize(tx, VerificationSessionEntity, advocateAgent, true, "consensus")
		}
		return c.markError(tx, VerificationSessionEntity, "skeptic_failed")
	}

	req.AdvocateArgument = advocateAgent.Argument
	req.AdvocateVerdict = advocateAgent.Verdict
	req.AdvocateConfidence = advocateAgent.Confidence
	req.SkepticArgument = skepticAgent.Argument
	req.SkepticVerdict = skepticAgent.Verdict
	req.SkepticConfidence = skepticAgent.Confidence

	final := advocateAgent
	by := "consensus"

	if !(advocateAgent.Verdict == skepticAgent.Verdict && advocateAgent.Confidence >= 0.8 && skepticAgent.Confidence >= 0.8) {
		managerReq := &dto.LLMGenerateContentRequest{
			Content:  utils.RolePrompt("manager", req),
			Provider: "CommandCode",
			Model:    "qwen/qwen3.7-flash",
		}

		final, err = c.runAgent(ctx, VerificationSessionEntity.ID, "manager", managerReq)
		by = "manager"
		if err != nil {
			final = &model.VerificationResult{
				Verdict: false,
				Confidence: 1,
				CategorySlug: advocateAgent.CategorySlug,
				Severity: advocateAgent.Severity,
				Argument: "verification_agent_error",
			}
			c.Log.Warnf("Failed update verification session : %+v", err)
			return c.markError(tx, VerificationSessionEntity, "manager_failed")
		}
	} else if !advocateAgent.Verdict {
		final = skepticAgent
	}

	return c.finalize(tx, VerificationSessionEntity, final, final.Verdict, by)
}

func (c *VerificationUseCase) runAgent(ctx context.Context, sessionID string, role string, request *dto.LLMGenerateContentRequest) (*model.VerificationResult, error) {
	tx := c.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	start := time.Now()

	llmResponse, err := c.LLMClient.GenerateContent(ctx, request)
	if err != nil {
		c.Log.Warnf("Failed to call LLM provider : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal menghubungi penyedia LLM")
	}

	verificationResult, err := parseResult(llmResponse)
	if err != nil {
		c.Log.Warnf("Failed to parse LLM response : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal memproses respons dari LLM")
	}

	lat := time.Since(start).Milliseconds()

	verificationLogEntity := &entity.VerificationLog{
		ID:            uuid.NewString(),
		SessionID:     sessionID,
		AgentRole:     role,
		LLMProvider:   request.Provider,
		LLMModel:      request.Model,
		LatencyMs:     lat,
		PromptUsed:    role,
	}

	if err != nil {
        errMsg := err.Error()
        verificationLogEntity.ErrorMessage = &errMsg
        verificationLogEntity.RawArgument = ""

        if logErr := c.VerificationLogRepository.Create(tx, verificationLogEntity); logErr != nil {
            c.Log.Warnf("Failed create error verification log : %+v", logErr)
        }
        
        tx.Commit()
        return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal memproses respons agen: "+errMsg)
    }

	verificationLogEntity.Verdict = &verificationResult.Verdict
	verificationLogEntity.Confidence = verificationResult.Confidence
	verificationLogEntity.CategorySlug = &verificationResult.CategorySlug
	verificationLogEntity.Severity = &verificationResult.Severity
	verificationLogEntity.RawArgument = verificationResult.Argument

	if err := c.VerificationLogRepository.Create(tx, verificationLogEntity); err != nil {
		c.Log.Warnf("Failed create verification log : %+v", err)
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
        c.Log.Warnf("Failed to commit transaction : %+v", err)
        return nil, err
    }

	return verificationResult, nil
}

func (c *VerificationUseCase) finalize(tx *gorm.DB, verificationSessionEntity *entity.VerificationSession, r *model.VerificationResult, verdict bool, by string) error {
	now := time.Now()
	verificationSessionEntity.FinalVerdict = &verdict
	verificationSessionEntity.FinalCategorySlug = &r.CategorySlug
	verificationSessionEntity.FinalReasoning = &r.Argument
	verificationSessionEntity.DecidedBy = &by
	verificationSessionEntity.CompletedAt = &now
	status := "verified"
	rejectReason := (*string)(nil)
	if verdict {
		verificationSessionEntity.Status = "approved"
	} else {
		verificationSessionEntity.Status = "rejected"
		status = "rejected"
		rejectReason = &r.Argument
		verificationSessionEntity.RejectReason = &r.Argument
	}
	if err := c.ReportClient.UpdateReportStatus(tx, verificationSessionEntity.ReportID, status, rejectReason); err != nil {
		return err
	}
	if err := c.VerificationSessionRepository.Update(tx, verificationSessionEntity); err != nil {
		return err
	}
	return tx.Commit().Error
}

func (c *VerificationUseCase) markError(tx *gorm.DB, verificationSessionEntity *entity.VerificationSession, msg string) error {
	now := time.Now()
	verificationSessionEntity.Status = "error"
	verificationSessionEntity.RejectReason = &msg
	verificationSessionEntity.CompletedAt = &now
	if err := c.VerificationSessionRepository.Update(tx, verificationSessionEntity); err != nil {
		c.Log.Warnf("Failed update verification session : %+v", err)
		return err
	}
	return tx.Commit().Error
}

func (c *VerificationUseCase) FindPending(ctx context.Context, limit int) ([]entity.VerificationSession, error) {
	return c.VerificationSessionRepository.FindPending(c.DB.WithContext(ctx), limit)
}

func parseResult(text *dto.LLMGenerateContentResponse) (*model.VerificationResult, error) {
	start := strings.Index(text.Content, "{")
	end := strings.LastIndex(text.Content, "}")
	if start >= 0 && end >= start {
		text.Content = text.Content[start : end+1]
	}
	var r model.VerificationResult
	if err := json.Unmarshal([]byte(text.Content), &r); err != nil {
		return nil, err
	}
	if r.Confidence < 0 {
		r.Confidence = 0
	}
	if r.Confidence > 1 {
		r.Confidence = 1
	}
	return &r, nil
}
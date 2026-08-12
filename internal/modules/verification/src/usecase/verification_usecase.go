package usecase

import (
	"context"
	"time"

	report_client "github.com/arttVinci/fixora-Backend/internal/modules/report-client"
	"github.com/arttVinci/fixora-Backend/internal/modules/verification/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/verification/src/infra"
	"github.com/arttVinci/fixora-Backend/internal/modules/verification/src/model"
	"github.com/arttVinci/fixora-Backend/internal/modules/verification/src/model/converter"
	"github.com/arttVinci/fixora-Backend/internal/modules/verification/src/repository"
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
	AdvocateProvider              infra.LLMProvider
	SkepticProvider               infra.LLMProvider
	ManagerProvider               infra.LLMProvider
}

func NewVerificationUseCase(db *gorm.DB, log *logrus.Logger, validate *validator.Validate, sr *repository.VerificationSessionRepository, lr *repository.VerificationLogRepository, rc report_client.Client, advocate, skeptic, manager infra.LLMProvider) *VerificationUseCase {
	return &VerificationUseCase{db, log, validate, sr, lr, rc, advocate, skeptic, manager}
}

func (c *VerificationUseCase) CreateVerification(ctx context.Context, reportID string) (*model.VerificationSessionResponse, error) {
	tx := c.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if s, err := c.VerificationSessionRepository.FindActiveByReportID(tx, reportID); err == nil {
		return converter.VerificationSessionToResponse(s), nil
	}

	s := &entity.VerificationSession{ID: uuid.NewString(), ReportID: reportID, Status: "pending"}
	if err := c.VerificationSessionRepository.Create(tx, s); err != nil {
		c.Log.Warnf("Failed create verification session : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat sesi verifikasi")
	}
	if err := tx.Commit().Error; err != nil {
		c.Log.Warnf("Failed commit transaction : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat sesi verifikasi")
	}
	return converter.VerificationSessionToResponse(s), nil
}
func (c *VerificationUseCase) RetrySession(ctx context.Context, sessionID string) (*model.VerificationSessionResponse, error) {
	tx := c.DB.WithContext(ctx).Begin()
	defer tx.Rollback()
	s := new(entity.VerificationSession)
	if err := c.VerificationSessionRepository.FindById(tx, s, sessionID); err != nil {
		c.Log.Warnf("Verification session not found : %+v", err)
		return nil, fiber.NewError(fiber.StatusNotFound, "Sesi verifikasi tidak ditemukan")
	}
	if s.Status != "error" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Hanya sesi error yang dapat diulang")
	}
	s.Status = "pending"
	s.FinalVerdict = nil
	s.FinalReasoning = nil
	s.RejectReason = nil
	s.DecidedBy = nil
	s.StartedAt = nil
	s.CompletedAt = nil
	if err := c.VerificationSessionRepository.Update(tx, s); err != nil {
		c.Log.Warnf("Failed update verification session : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui sesi verifikasi")
	}
	if err := tx.Commit().Error; err != nil {
		c.Log.Warnf("Failed commit transaction : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui sesi verifikasi")
	}
	return converter.VerificationSessionToResponse(s), nil
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
	s := new(entity.VerificationSession)
	if err := c.VerificationSessionRepository.FindById(tx, s, sessionID); err != nil {
		return err
	}
	report, err := c.ReportClient.GetReportByID(tx, s.ReportID)
	if err != nil {
		return c.markError(tx, s, err.Error())
	}
	now := time.Now()
	if report.SourceType == "gov_data" {
		verdict := true
		reason := "gov_data"
		s.Status = "approved"
		s.FinalVerdict = &verdict
		s.SkipReason = &reason
		s.DecidedBy = &reason
		s.StartedAt = &now
		s.CompletedAt = &now
		if err := c.ReportClient.UpdateReportStatus(tx, s.ReportID, "verified", nil); err != nil {
			return err
		}
		if err := c.VerificationSessionRepository.Update(tx, s); err != nil {
			return err
		}
		return tx.Commit().Error
	}

	s.Status = "in_progress"
	s.StartedAt = &now

	if err := c.VerificationSessionRepository.Update(tx, s); err != nil {
		return err
	}

	req := &infra.VerificationRequest{
		ReportTitle:       report.Title,
		ReportDescription: report.Description,
		ReportSeverity:    report.Severity,
		ReportCategory:    report.CategorySlug,
		ReportSourceType:  report.SourceType,
		ReportPhotoURL:    report.PrimaryPhotoURL,
		ReportAddress:     report.Address,
	}

	adv, err := c.runAgent(ctx, tx, s.ID, "advocate", c.AdvocateProvider, req)
	if err != nil {
		return c.markError(tx, s, "advocate_failed")
	}
	sk, err := c.runAgent(ctx, tx, s.ID, "skeptic", c.SkepticProvider, req)
	if err != nil {
		if adv.Confidence >= 0.9 && adv.Verdict {
			return c.finalize(tx, s, adv, true, "consensus")
		}
		return c.markError(tx, s, "skeptic_failed")
	}
	req.AdvocateArgument = adv.Argument
	req.AdvocateVerdict = adv.Verdict
	req.AdvocateConfidence = adv.Confidence
	req.SkepticArgument = sk.Argument
	req.SkepticVerdict = sk.Verdict
	req.SkepticConfidence = sk.Confidence
	final := adv
	by := "consensus"
	if !(adv.Verdict == sk.Verdict && adv.Confidence >= 0.8 && sk.Confidence >= 0.8) {
		final, err = c.runAgent(ctx, tx, s.ID, "manager", c.ManagerProvider, req)
		by = "manager"
		if err != nil {
			final = &infra.VerificationResult{Verdict: false, Confidence: 1, CategorySlug: adv.CategorySlug, Severity: adv.Severity, Argument: "verification_agent_error"}
		}
	} else if !adv.Verdict {
		final = sk
	}
	return c.finalize(tx, s, final, final.Verdict, by)
}
func (c *VerificationUseCase) runAgent(ctx context.Context, tx *gorm.DB, sid, role string, p infra.LLMProvider, req *infra.VerificationRequest) (*infra.VerificationResult, error) {
	start := time.Now()
	var r *infra.VerificationResult
	var err error

	if role == "advocate" {
		r, err = p.AnalyzeAsAdvocate(ctx, req)
	} else if role == "skeptic" {
		r, err = p.AnalyzeAsSkeptic(ctx, req)
	} else {
		r, err = p.AnalyzeAsManager(ctx, req)
	}
	lat := time.Since(start).Milliseconds()

	l := &entity.VerificationLog{
		ID:            uuid.NewString(),
		SessionID:     sid,
		AgentRole:     role,
		LLMProvider: p.ProviderName(),
		LLMModel:      p.ModelName(),
		LatencyMs:     lat,
	}

	if err != nil {
		e := err.Error()
		l.RawArgument = ""
		l.PromptUsed = role
		l.ErrorMessage = &e
		_ = c.VerificationLogRepository.Create(tx, l)
		return nil, err
	}
	l.Verdict = &r.Verdict
	l.Confidence = r.Confidence
	l.CategorySlug = &r.CategorySlug
	l.Severity = &r.Severity
	l.RawArgument = r.Argument
	l.PromptUsed = role
	if err := c.VerificationLogRepository.Create(tx, l); err != nil {
		return nil, err
	}
	return r, nil
}
func (c *VerificationUseCase) finalize(tx *gorm.DB, s *entity.VerificationSession, r *infra.VerificationResult, verdict bool, by string) error {
	now := time.Now()
	s.FinalVerdict = &verdict
	s.FinalCategorySlug = &r.CategorySlug
	s.FinalSeverity = &r.Severity
	s.FinalReasoning = &r.Argument
	s.DecidedBy = &by
	s.CompletedAt = &now
	status := "verified"
	rejectReason := (*string)(nil)
	if verdict {
		s.Status = "approved"
	} else {
		s.Status = "rejected"
		status = "rejected"
		rejectReason = &r.Argument
		s.RejectReason = &r.Argument
	}
	if err := c.ReportClient.UpdateReportStatus(tx, s.ReportID, status, rejectReason); err != nil {
		return err
	}
	if err := c.VerificationSessionRepository.Update(tx, s); err != nil {
		return err
	}
	return tx.Commit().Error
}
func (c *VerificationUseCase) markError(tx *gorm.DB, s *entity.VerificationSession, msg string) error {
	now := time.Now()
	s.Status = "error"
	s.RejectReason = &msg
	s.CompletedAt = &now
	if err := c.VerificationSessionRepository.Update(tx, s); err != nil {
		return err
	}
	return tx.Commit().Error
}
func (c *VerificationUseCase) FindPending(ctx context.Context, limit int) ([]entity.VerificationSession, error) {
	return c.VerificationSessionRepository.FindPending(c.DB.WithContext(ctx), limit)
}

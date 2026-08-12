package converter

import (
	"github.com/arttVinci/fixora-Backend/internal/modules/verification/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/verification/src/model"
)

func VerificationSessionToResponse(session *entity.VerificationSession) *model.VerificationSessionResponse {
	logs := make([]model.VerificationLogResponse, len(session.Logs))
	for i, log := range session.Logs {
		logs[i] = *VerificationLogToResponse(&log)
	}
	return &model.VerificationSessionResponse{
		ID: session.ID, ReportID: session.ReportID, Status: session.Status,
		FinalVerdict: session.FinalVerdict, FinalCategorySlug: session.FinalCategorySlug, FinalSeverity: session.FinalSeverity,
		FinalReasoning: session.FinalReasoning, RejectReason: session.RejectReason, DecidedBy: session.DecidedBy, SkipReason: session.SkipReason,
		StartedAt: session.StartedAt, CompletedAt: session.CompletedAt, CreatedAt: session.CreatedAt, UpdatedAt: session.UpdatedAt, Logs: logs,
	}
}

func VerificationLogToResponse(log *entity.VerificationLog) *model.VerificationLogResponse {
	return &model.VerificationLogResponse{
		ID: log.ID, SessionID: log.SessionID, AgentRole: log.AgentRole, LLMProvider: log.LLMProvider, LLMModel: log.LLMModel,
		Verdict: log.Verdict, Confidence: log.Confidence, CategorySlug: log.CategorySlug, Severity: log.Severity,
		RawArgument: log.RawArgument, PromptUsed: log.PromptUsed, LatencyMs: log.LatencyMs, ErrorMessage: log.ErrorMessage, CreatedAt: log.CreatedAt,
	}
}

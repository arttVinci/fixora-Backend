package verification

import (
	"github.com/arttVinci/fixora-Backend/internal/shared/response"
	"github.com/gofiber/fiber/v2"
)

func (m *Module) RegisterRoutes(router fiber.Router) {
	m.worker.StartScheduler()
	group := router.Group("/crawl/verify")
	group.Post("/trigger/:reportId", m.triggerVerification)
	group.Post("/retry/:sessionId", m.retrySession)
	group.Get("/sessions/:reportId", m.getSessionsByReportID)
}

// TriggerVerification godoc
// @Summary Trigger verification
// @Description Trigger verification for report
// @Tags Verification
// @Accept json
// @Produce json
// @Param reportId path string true "Report ID"
// @Success 200 {object} response.WebResponse[model.VerificationSessionResponse]
// @Failure 400 {object} response.WebResponse[any]
// @Router /crawl/verify/trigger/{reportId} [post]
func (m *Module) triggerVerification(ctx *fiber.Ctx) error {
	resp, err := m.UseCase.CreateVerification(ctx.UserContext(), ctx.Params("reportId"))
	if err != nil {
		return err
	}
	return ctx.JSON(response.WebResponse[any]{Data: resp, Message: "Berhasil memicu verifikasi", Success: true})
}

// RetrySession godoc
// @Summary Retry verification session
// @Description Retry error verification session
// @Tags Verification
// @Accept json
// @Produce json
// @Param sessionId path string true "Session ID"
// @Success 200 {object} response.WebResponse[model.VerificationSessionResponse]
// @Failure 400 {object} response.WebResponse[any]
// @Router /crawl/verify/retry/{sessionId} [post]
func (m *Module) retrySession(ctx *fiber.Ctx) error {
	resp, err := m.UseCase.RetrySession(ctx.UserContext(), ctx.Params("sessionId"))
	if err != nil {
		return err
	}
	return ctx.JSON(response.WebResponse[any]{Data: resp, Message: "Berhasil mengulang verifikasi", Success: true})
}

// GetSessionsByReportID godoc
// @Summary Get verification sessions
// @Description Get verification sessions by report ID
// @Tags Verification
// @Accept json
// @Produce json
// @Param reportId path string true "Report ID"
// @Success 200 {object} response.WebResponse[[]model.VerificationSessionResponse]
// @Failure 400 {object} response.WebResponse[any]
// @Router /crawl/verify/sessions/{reportId} [get]
func (m *Module) getSessionsByReportID(ctx *fiber.Ctx) error {
	resp, err := m.UseCase.GetSessionsByReportID(ctx.UserContext(), ctx.Params("reportId"))
	if err != nil {
		return err
	}
	return ctx.JSON(response.WebResponse[any]{Data: resp, Message: "Berhasil menampilkan sesi verifikasi", Success: true})
}

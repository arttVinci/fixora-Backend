package controller

import (
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/model"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/usecase"
	"github.com/arttVinci/fixora-Backend/internal/shared/dto"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

type ReportController struct {
	Log                 *logrus.Logger
	UseCase             *usecase.ReportUseCase
	AnalyzePhotoUseCase *usecase.AnalyzePhotoUseCase
}

func NewReportController(useCase *usecase.ReportUseCase, analyzePhotoUseCase *usecase.AnalyzePhotoUseCase, logger *logrus.Logger) *ReportController {
	return &ReportController{
		Log:                 logger,
		UseCase:             useCase,
		AnalyzePhotoUseCase: analyzePhotoUseCase,
	}
}

// SearchMap godoc
// @Summary      Get interactive map data
// @Description  Get report points for interactive map based on bounding box
// @Tags         reports
// @Accept       json
// @Produce      json
// @Param        min_lat query number true "Minimum Latitude"
// @Param        max_lat query number true "Maximum Latitude"
// @Param        min_lng query number true "Minimum Longitude"
// @Param        max_lng query number true "Maximum Longitude"
// @Param        category_id query string false "Category ID Filter"
// @Param        status query string false "Status Filter"
// @Param        severity query string false "Severity Filter"
// @Param        source_type query string false "Source Type Filter"
// @Success      200  {object}  response.WebResponse[[]model.ReportMapResponse]
// @Failure      400  {object}  response.WebResponse[any]
// @Failure      500  {object}  response.WebResponse[any]
// @Router       /api/v1/reports/map [get]
func (c *ReportController) SearchMap(ctx *fiber.Ctx) error {
	request := new(model.SearchReportMapRequest)
	if err := ctx.QueryParser(request); err != nil {
		c.Log.Warnf("Failed to parse query params : %+v", err)
		return fiber.NewError(fiber.StatusBadRequest, "Format query parameter tidak valid")
	}

	resp, err := c.UseCase.SearchMap(ctx.UserContext(), request)
	if err != nil {
		return err
	}

	return ctx.JSON(dto.WebResponse[[]model.ReportMapResponse]{
		Data:    resp,
		Message: "Berhasil menampilkan data peta",
		Success: true,
	})
}

// GetDetail godoc
// @Summary      Get report detail
// @Description  Get full detail of a single infrastructure report by ID
// @Tags         reports
// @Accept       json
// @Produce      json
// @Param        id path string true "Report ID"
// @Success      200  {object}  response.WebResponse[model.ReportDetailResponse]
// @Failure      404  {object}  response.WebResponse[any]
// @Failure      500  {object}  response.WebResponse[any]
// @Router       /api/v1/reports/{id} [get]
func (c *ReportController) GetDetail(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	resp, err := c.UseCase.GetDetail(ctx.UserContext(), id)
	if err != nil {
		c.Log.Warnf("Failed to get report detail : %+v", err)
		return err
	}

	return ctx.JSON(dto.WebResponse[*model.ReportDetailResponse]{
		Data:    resp,
		Message: "Berhasil menampilkan detail laporan",
		Success: true,
	})
}

// AnalyzePhoto godoc
// @Summary      Analyze uploaded photo (CV classifier)
// @Description  Upload an infrastructure problem photo to get AI-generated draft (title, description, category, severity).
// @Tags         reports
// @Accept       multipart/form-data
// @Produce      json
// @Param        photo formData file true "Problem photo"
// @Success      200  {object}  response.WebResponse[*model.IssueAnalysisResultResponse]
// @Failure      400  {object}  response.WebResponse[any]
// @Failure      500  {object}  response.WebResponse[any]
// @Router       /api/v1/reports/analyze-photo [post]
func (c *ReportController) AnalyzePhoto(ctx *fiber.Ctx) error {
	file, err := ctx.FormFile("photo")
	if err != nil {
		c.Log.Warnf("Failed to get photo from form : %+v", err)
		return fiber.NewError(fiber.StatusBadRequest, "File foto tidak ditemukan pada form data")
	}

	resp, err := c.AnalyzePhotoUseCase.AnalyzeIssueImage(ctx.UserContext(), file)
	if err != nil {
		c.Log.Warnf("Failed to analyze photo : %+v", err)
		return err
	}

	return ctx.JSON(dto.WebResponse[*model.IssueAnalysisResultResponse]{
		Data:    resp,
		Message: "Berhasil menganalisis foto",
		Success: true,
	})
}

// Create godoc
// @Summary      Create report (user submission)
// @Description  Submit a new infrastructure problem report with photo URL, location, and optional reporter email.
// @Tags         reports
// @Accept       json
// @Produce      json
// @Param        request body model.CreateReportRequest true "Create report payload"
// @Success      201  {object}  response.WebResponse[*model.ReportDetailResponse]
// @Failure      400  {object}  response.WebResponse[any]
// @Failure      500  {object}  response.WebResponse[any]
// @Router       /api/v1/reports [post]
func (c *ReportController) Create(ctx *fiber.Ctx) error {
	request := new(model.CreateReportRequest)
	if err := ctx.BodyParser(request); err != nil {
		c.Log.Warnf("Failed to parse request body : %+v", err)
		return fiber.NewError(fiber.StatusBadRequest, "Format data request tidak valid")
	}

	resp, err := c.UseCase.CreateReport(ctx.UserContext(), request)
	if err != nil {
		c.Log.Warnf("Failed to create report : %+v", err)
		return err
	}

	return ctx.Status(fiber.StatusCreated).JSON(dto.WebResponse[*model.ReportDetailResponse]{
		Data:    resp,
		Message: "Berhasil membuat laporan",
		Success: true,
	})
}

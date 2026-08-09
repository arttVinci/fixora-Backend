package controller

import (
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/model"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/usecase"
	"github.com/arttVinci/fixora-Backend/internal/shared/response"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

type ReportController struct {
	Log     *logrus.Logger
	UseCase *usecase.ReportUseCase
}

func NewReportController(useCase *usecase.ReportUseCase, logger *logrus.Logger) *ReportController {
	return &ReportController{
		Log:     logger,
		UseCase: useCase,
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

	return ctx.JSON(response.WebResponse[[]model.ReportMapResponse]{
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

	return ctx.JSON(response.WebResponse[*model.ReportDetailResponse]{
		Data:    resp,
		Message: "Berhasil menampilkan detail laporan",
		Success: true,
	})
}
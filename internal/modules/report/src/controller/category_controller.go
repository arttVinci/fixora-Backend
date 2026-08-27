package controller

import (
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/model"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/usecase"
	"github.com/arttVinci/fixora-Backend/internal/shared/dto"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

type CategoryController struct {
	Log     *logrus.Logger
	UseCase *usecase.CategoryUseCase
}

func NewCategoryController(useCase *usecase.CategoryUseCase, logger *logrus.Logger) *CategoryController {
	return &CategoryController{
		Log:     logger,
		UseCase: useCase,
	}
}

// List godoc
// @Summary      Get all categories
// @Description  Get list of all infrastructure problem categories
// @Tags         categories
// @Accept       json
// @Produce      json
// @Success      200  {object}  dto.WebResponse[[]model.CategoryResponse]
// @Failure      500  {object}  dto.WebResponse[any]
// @Router       /api/v1/categories [get]
func (c *CategoryController) List(ctx *fiber.Ctx) error {
	resp, err := c.UseCase.List(ctx.UserContext())
	if err != nil {
		c.Log.Warnf("Failed to get categories : %+v", err)
		return err
	}

	return ctx.JSON(dto.WebResponse[[]model.CategoryResponse]{
		Data:    resp,
		Message: "Berhasil menampilkan daftar kategori",
		Success: true,
	})
}

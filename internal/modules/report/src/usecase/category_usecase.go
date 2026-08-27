package usecase

import (
	"context"

	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/model"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/model/converter"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/repository"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type CategoryUseCase struct {
	DB                 *gorm.DB
	Log                *logrus.Logger
	CategoryRepository *repository.CategoryRepository
}

func NewCategoryUseCase(
	db *gorm.DB,
	log *logrus.Logger,
	categoryRepo *repository.CategoryRepository,
) *CategoryUseCase {
	return &CategoryUseCase{
		DB:                 db,
		Log:                log,
		CategoryRepository: categoryRepo,
	}
}

func (c *CategoryUseCase) List(ctx context.Context) ([]model.CategoryResponse, error) {
	tx := c.DB.WithContext(ctx)

	items, err := c.CategoryRepository.FindAll(tx)
	if err != nil {
		c.Log.Warnf("Failed to get categories : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data kategori")
	}

	responses := make([]model.CategoryResponse, len(items))
	for i, item := range items {
		responses[i] = *converter.CategoryToResponse(&item)
	}

	return responses, nil
}

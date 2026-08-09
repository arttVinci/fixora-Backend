package usecase

import (
	"context"

	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/model"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/model/converter"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/repository"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ReportUseCase struct {
	DB               *gorm.DB
	Log              *logrus.Logger
	Validate         *validator.Validate
	ReportRepository *repository.ReportRepository
}

func NewReportUseCase(
	db *gorm.DB,
	log *logrus.Logger,
	validate *validator.Validate,
	reportRepo *repository.ReportRepository,
) *ReportUseCase {
	return &ReportUseCase{
		DB:               db,
		Log:              log,
		Validate:         validate,
		ReportRepository: reportRepo,
	}
}

func (c *ReportUseCase) GetDetail(ctx context.Context, id string) (*model.ReportDetailResponse, error) {
	tx := c.DB.WithContext(ctx)

	report := new(entity.Report)
	if err := c.ReportRepository.FindDetailByID(tx, report, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.Log.Warnf("Report not found : %+v", err)
			return nil, fiber.NewError(fiber.StatusNotFound, "Laporan tidak ditemukan")
		}
		c.Log.Warnf("Failed to get report detail : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data laporan")
	}

	return converter.ReportToDetailResponse(report), nil
}

func (c *ReportUseCase) SearchMap(ctx context.Context, request *model.SearchReportMapRequest) ([]model.ReportMapResponse, error) {
	if err := c.Validate.Struct(request); err != nil {
		c.Log.Warnf("Invalid request body : %+v", err)
		return nil, fiber.NewError(fiber.StatusBadRequest, "Format data request tidak valid")
	}

	tx := c.DB.WithContext(ctx)
	items, err := c.ReportRepository.SearchMap(tx, request)
	if err != nil {
		c.Log.Warnf("Failed to search map reports : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mencari data laporan")
	}

	responses := make([]model.ReportMapResponse, len(items))
	for i, item := range items {
		responses[i] = *converter.ReportToMapResponse(&item)
	}

	return responses, nil
}
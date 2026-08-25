package repository

import (
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	shared_repo "github.com/arttVinci/fixora-Backend/internal/shared/repository"
	"github.com/sirupsen/logrus"
)

type ReportPhotoRepository struct {
	shared_repo.Repository[entity.ReportPhoto]
	Log *logrus.Logger
}

func NewReportPhotoRepository(log *logrus.Logger) *ReportPhotoRepository {
	return &ReportPhotoRepository{Log: log}
}

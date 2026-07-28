package repository

import (
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	shared_repo "github.com/arttVinci/fixora-Backend/internal/shared/repository"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ReporterRepository struct {
	shared_repo.Repository[entity.Reporter]
	Log *logrus.Logger
}

func NewReporterRepository(log *logrus.Logger) *ReporterRepository {
	return &ReporterRepository{Log: log}
}

func (r *ReporterRepository) FindByEmail(db *gorm.DB, item *entity.Reporter, email string) error {
	return db.Where("email = ?", email).Take(item).Error
}

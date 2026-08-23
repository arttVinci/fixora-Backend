package repository

import (
	"github.com/arttVinci/fixora-Backend/internal/modules/verification/src/entity"
	shared_repo "github.com/arttVinci/fixora-Backend/internal/shared/repository"
	"github.com/sirupsen/logrus"
)

type VerificationLogRepository struct {
	shared_repo.Repository[entity.VerificationLog]
	Log *logrus.Logger
}

func NewVerificationLogRepository(log *logrus.Logger) *VerificationLogRepository {
	return &VerificationLogRepository{Log: log}
}

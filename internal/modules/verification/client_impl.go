package verification

import (
	verification_client "github.com/arttVinci/fixora-Backend/internal/modules/verification-client"
	"github.com/arttVinci/fixora-Backend/internal/modules/verification/src/usecase"
	"gorm.io/gorm"
)

type clientImpl struct{ useCase *usecase.VerificationUseCase }

func (c *clientImpl) CreateVerification(tx *gorm.DB, reportID string) (*verification_client.VerificationClientResponse, error) {
	resp, err := c.useCase.CreateVerification(tx.Statement.Context, reportID)
	if err != nil {
		return nil, err
	}
	return &verification_client.VerificationClientResponse{ID: resp.ID, ReportID: resp.ReportID, Status: resp.Status}, nil
}

package verification_client

import "gorm.io/gorm"

type Client interface {
	CreateVerification(tx *gorm.DB, reportID string) (*VerificationClientResponse, error)
}

package model

type TriggerVerificationRequest struct {
	ReportID string `validate:"required"`
}

type RetryVerificationRequest struct {
	SessionID string `validate:"required"`
}

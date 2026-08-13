package model

type VerificationRequest struct {
	ReportTitle        string
	ReportDescription  string
	ReportSeverity     string
	ReportCategory     string
	ReportSourceType   string
	ReportPhotoURL     string
	ReportAddress      string
	AdvocateArgument   string
	AdvocateVerdict    bool
	AdvocateConfidence float64
	SkepticArgument    string
	SkepticVerdict     bool
	SkepticConfidence  float64
}

type TriggerVerificationRequest struct {
	ReportID string `validate:"required"`
}

type RetryVerificationRequest struct {
	SessionID string `validate:"required"`
}

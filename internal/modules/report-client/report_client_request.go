package report_client

import "time"

type ReportClientRequest struct {
	ID              string
	CategoryID      string
	VillageID       string
	Title           string
	Description     *string
	Latitude        float64
	Longitude       float64
	Address         *string
	Severity        string
	Status          string
	SourceType      string
	ConfidenceScore float64
	FirstReportedAt *time.Time
}

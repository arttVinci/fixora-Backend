package model

import "time"

type ReportMapResponse struct {
	ID              string  `json:"id"`
	Title           string  `json:"title"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	Severity        string  `json:"severity"`
	CategorySlug    string  `json:"category_slug"`
	Status          string  `json:"status"`
	PrimaryPhotoURL string  `json:"primary_photo_url"`
	SourceType      string  `json:"source_type"`
}

type ReportDetailResponse struct {
	ID                 string     `json:"id"`
	Title              string     `json:"title"`
	Description        *string    `json:"description,omitempty"`
	Latitude           float64    `json:"latitude"`
	Longitude          float64    `json:"longitude"`
	Address            *string    `json:"address,omitempty"`
	Severity           string     `json:"severity"`
	Status             string     `json:"status"`
	SourceType         string     `json:"source_type"`
	ConfidenceScore    float64    `json:"confidence_score"`
	BudgetInfo         *string    `json:"budget_info,omitempty"`
	CategoryName       string     `json:"category_name"`
	CategorySlug       string     `json:"category_slug"`
	PrimaryPhotoURL    string     `json:"primary_photo_url"`
	AdditionalPhotos   []string   `json:"additional_photos,omitempty"`
	TotalConfirmations int64      `json:"total_confirmations"`
	MergedIntoID       *string    `json:"merged_into_id,omitempty"`
	FirstReportedAt    *time.Time `json:"first_reported_at"`
	LastConfirmedAt    *time.Time `json:"last_confirmed_at,omitempty"`
}

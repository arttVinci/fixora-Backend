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
	PhotoURL        string  `json:"photo_url"`
	Source          string  `json:"source"`
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
	Source         	   string     `json:"source"`
	SourceURL          *string    `json:"source_url,omitempty"`
	CategoryName       string     `json:"category_name"`
	CategorySlug       string     `json:"category_slug"`
	PhotoURL           *string     `json:"photo_url"`
	AdditionalPhotos   *[]string   `json:"additional_photos,omitempty"`
	TotalConfirmations int64      `json:"total_confirmations"`
	MergedIntoID       *string    `json:"merged_into_id,omitempty"`
	FirstReportedAt    *time.Time `json:"first_reported_at"`
	LastConfirmedAt    *time.Time `json:"last_confirmed_at,omitempty"`
}

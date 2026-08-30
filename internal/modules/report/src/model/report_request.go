package model

type SearchReportMapRequest struct {
	MinLat     float64 `json:"min_lat" query:"min_lat" validate:"required,min=-90,max=90"`
	MaxLat     float64 `json:"max_lat" query:"max_lat" validate:"required,min=-90,max=90"`
	MinLng     float64 `json:"min_lng" query:"min_lng" validate:"required,min=-180,max=180"`
	MaxLng     float64 `json:"max_lng" query:"max_lng" validate:"required,min=-180,max=180"`
	CategoryID string  `json:"category_id" query:"category_id" validate:"omitempty,uuid"`
	Status     string  `json:"status" query:"status" validate:"omitempty"`
	Severity   string  `json:"severity" query:"severity" validate:"omitempty,oneof=ringan sedang parah"`
	SourceType string  `json:"source_type" query:"source_type" validate:"omitempty,oneof=user_report ai_news gov_data"`
}

type SearchReportListRequest struct {
	Title      string   `json:"title" query:"title" validate:"omitempty,max=200"`
	CategoryID []string `json:"category_id" query:"category_id" validate:"omitempty"`
	Status     []string `json:"status" query:"status" validate:"omitempty"`
	Severity   []string `json:"severity" query:"severity" validate:"omitempty"`
	SourceType []string `json:"source_type" query:"source_type" validate:"omitempty"`
	Page       int      `json:"page" query:"page" validate:"min=1"`
	Size       int      `json:"size" query:"size" validate:"min=1,max=100"`
	Sort       string   `json:"sort" query:"sort" validate:"omitempty"`
}

type CreateReportRequest struct {
	CategoryID       string  `json:"category_id" validate:"required,uuid"`
	Title            string  `json:"title" validate:"required,max=200"`
	Description      string  `json:"description" validate:"omitempty"`
	Latitude         float64 `json:"latitude" validate:"required,min=-90,max=90"`
	Longitude        float64 `json:"longitude" validate:"required,min=-180,max=180"`
	Address          string  `json:"address" validate:"omitempty,max=500"`
	Severity         string  `json:"severity" validate:"required,oneof=ringan sedang parah"`
	StagingSessionID string  `json:"staging_session_id" validate:"required,uuid"`
	ReporterEmail    string  `json:"reporter_email" validate:"omitempty,email,max=150"`
}

package model

type IssueAnalysisResultResponse struct {
	SessionID   string  `json:"session_id"`
	PhotoURL    string  `json:"photo_url"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Severity    string  `json:"severity"`
	Location    string  `json:"location,omitempty"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
	Address     string  `json:"address,omitempty"`
	Reason      string  `json:"reason,omitempty"`
	IsRelevant  bool    `json:"is_relevant"`
}

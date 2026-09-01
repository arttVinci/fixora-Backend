package model

type IssueAnalysisResultResponse struct {
	SessionID   string `json:"session_id"`
	PhotoURL    string `json:"photo_url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	IsRelevant  bool   `json:"is_relevant"`
}

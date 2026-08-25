package model

type IssueAnalysisResultResponse struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Severity    string `json:"severity"`
}

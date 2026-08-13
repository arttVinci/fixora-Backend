package model

import "time"

type VerificationResult struct {
	Verdict      bool    `json:"verdict"`
	Confidence   float64 `json:"confidence"`
	CategorySlug string  `json:"category_slug"`
	Severity     string  `json:"severity"`
	Argument     string  `json:"argument"`
}

type VerificationLogResponse struct {
	ID           string     `json:"id"`
	SessionID    string     `json:"session_id"`
	AgentRole    string     `json:"agent_role"`
	LLMProvider  string     `json:"llm_provider"`
	LLMModel     string     `json:"llm_model"`
	Verdict      *bool      `json:"verdict,omitempty"`
	Confidence   float64    `json:"confidence"`
	CategorySlug *string    `json:"category_slug,omitempty"`
	Severity     *string    `json:"severity,omitempty"`
	RawArgument  string     `json:"raw_argument"`
	PromptUsed   string     `json:"prompt_used"`
	LatencyMs    int64      `json:"latency_ms"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	CreatedAt    *time.Time `json:"created_at,omitempty"`
}

type VerificationSessionResponse struct {
	ID                string                    `json:"id"`
	ReportID          string                    `json:"report_id"`
	Status            string                    `json:"status"`
	FinalVerdict      *bool                     `json:"final_verdict,omitempty"`
	FinalCategorySlug *string                   `json:"final_category_slug,omitempty"`
	FinalSeverity     *string                   `json:"final_severity,omitempty"`
	FinalReasoning    *string                   `json:"final_reasoning,omitempty"`
	RejectReason      *string                   `json:"reject_reason,omitempty"`
	DecidedBy         *string                   `json:"decided_by,omitempty"`
	SkipReason        *string                   `json:"skip_reason,omitempty"`
	StartedAt         *time.Time                `json:"started_at,omitempty"`
	CompletedAt       *time.Time                `json:"completed_at,omitempty"`
	CreatedAt         *time.Time                `json:"created_at,omitempty"`
	UpdatedAt         *time.Time                `json:"updated_at,omitempty"`
	Logs              []VerificationLogResponse `json:"logs,omitempty"`
}

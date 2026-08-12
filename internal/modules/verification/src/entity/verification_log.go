package entity

import "time"

type VerificationLog struct {
	ID           string     `gorm:"column:id;primaryKey;type:varchar(36)"`
	SessionID    string     `gorm:"column:session_id;type:varchar(36);not null;index"`
	AgentRole    string     `gorm:"column:agent_role;type:varchar(20);not null"`
	LLMProvider  string     `gorm:"column:llm_provider;type:varchar(20);not null"`
	LLMModel     string     `gorm:"column:llm_model;type:varchar(50);not null"`
	Verdict      *bool      `gorm:"column:verdict;type:boolean"`
	Confidence   float64    `gorm:"column:confidence;type:float;not null;default:0"`
	CategorySlug *string    `gorm:"column:category_slug;type:varchar(50)"`
	Severity     *string    `gorm:"column:severity;type:varchar(10)"`
	RawArgument  string     `gorm:"column:raw_argument;type:text;not null"`
	PromptUsed   string     `gorm:"column:prompt_used;type:text;not null"`
	LatencyMs    int64      `gorm:"column:latency_ms;type:int;not null;default:0"`
	ErrorMessage *string    `gorm:"column:error_message;type:text"`
	CreatedAt    *time.Time `gorm:"column:created_at;autoCreateTime"`

	Session *VerificationSession `gorm:"foreignKey:SessionID;references:ID;constraint:OnDelete:CASCADE"`
}

func (VerificationLog) TableName() string { return "verification_logs" }

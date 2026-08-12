package entity

import "time"

type VerificationSession struct {
	ID                string     `gorm:"column:id;primaryKey;type:varchar(36)"`
	ReportID          string     `gorm:"column:report_id;type:varchar(100);not null;index"`
	Status            string     `gorm:"column:status;type:varchar(20);not null;default:'pending';index"`
	FinalVerdict      *bool      `gorm:"column:final_verdict;type:boolean"`
	FinalCategorySlug *string    `gorm:"column:final_category_slug;type:varchar(50)"`
	FinalSeverity     *string    `gorm:"column:final_severity;type:varchar(10)"`
	FinalReasoning    *string    `gorm:"column:final_reasoning;type:text"`
	RejectReason      *string    `gorm:"column:reject_reason;type:text"`
	DecidedBy         *string    `gorm:"column:decided_by;type:varchar(20)"`
	SkipReason        *string    `gorm:"column:skip_reason;type:varchar(50)"`
	StartedAt         *time.Time `gorm:"column:started_at"`
	CompletedAt       *time.Time `gorm:"column:completed_at"`
	CreatedAt         *time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         *time.Time `gorm:"column:updated_at;autoUpdateTime"`

	Logs []VerificationLog `gorm:"foreignKey:SessionID;references:ID"`
}

func (VerificationSession) TableName() string { return "verification_sessions" }

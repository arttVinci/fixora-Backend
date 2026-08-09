package entity

import "time"

type DuplicateReport struct {
	ID              string     `gorm:"column:id;primaryKey;type:varchar(36)"`
	ReportID        string     `gorm:"column:report_id;type:varchar(36);not null"`
	ParentID        string     `gorm:"column:parent_id;type:varchar(36);not null"`
	Reason          string     `gorm:"column:reason;type:varchar(50);not null"`
	SimilarityScore float64    `gorm:"column:similarity_score;type:float;not null"`
	CreatedAt       *time.Time `gorm:"column:created_at;autoCreateTime"`

	// Relasi
	Report *Report `gorm:"foreignKey:ReportID;references:ID"`
	Parent *Report `gorm:"foreignKey:ParentID;references:ID"`
}

func (DuplicateReport) TableName() string {
	return "duplicate_reports"
}

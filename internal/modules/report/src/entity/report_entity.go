package entity

import "time"

type Report struct {
	ID              string     `gorm:"column:id;primaryKey;type:varchar(36)"`
	CategoryID      string     `gorm:"column:category_id;type:varchar(36);not null"`
	VillageID       *string    `gorm:"column:village_id;type:varchar(36)"`
	Title           string     `gorm:"column:title;type:varchar(200);not null"`
	Description     *string    `gorm:"column:description;type:text"`
	Latitude        float64    `gorm:"column:latitude;type:decimal(10,8);not null"`
	Longitude       float64    `gorm:"column:longitude;type:decimal(11,8);not null"`
	Address         *string    `gorm:"column:address;type:varchar(500)"`
	Severity        int        `gorm:"column:severity;type:smallint;not null"`
	Status          string     `gorm:"column:status;type:varchar(25);not null;default:'pending_verification'"`
	SourceType      string     `gorm:"column:source_type;type:varchar(15);not null"`
	PerceptualHash  *string    `gorm:"column:perceptual_hash;type:varchar(64)"`
	MergedIntoID    *string    `gorm:"column:merged_into_id;type:varchar(36)"`
	FirstReportedAt *time.Time `gorm:"column:first_reported_at;not null"`
	CreatedAt       *time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       *time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relasi
	Category     *Category     `gorm:"foreignKey:CategoryID;references:ID"`
	Photos       []ReportPhoto `gorm:"foreignKey:ReportID;references:ID"`
	MergedReport *Report       `gorm:"foreignKey:MergedIntoID;references:ID"`
}

func (Report) TableName() string {
	return "reports"
}

type ReportPhoto struct {
	ID             string     `gorm:"column:id;primaryKey;type:varchar(36)"`
	ReportID       string     `gorm:"column:report_id;type:varchar(36);not null"`
	PhotoURL       string     `gorm:"column:photo_url;type:varchar(500);not null"`
	IsPrimary      bool       `gorm:"column:is_primary;type:boolean;not null;default:false"`
	PerceptualHash *string    `gorm:"column:perceptual_hash;type:varchar(64)"`
	CreatedAt      *time.Time `gorm:"column:created_at;autoCreateTime"`

	// Relasi
	Report *Report `gorm:"foreignKey:ReportID;references:ID;constraint:OnDelete:CASCADE"`
}

func (ReportPhoto) TableName() string {
	return "report_photos"
}
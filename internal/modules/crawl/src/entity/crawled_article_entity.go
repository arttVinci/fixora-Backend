package entity

import "time"

type CrawledArticle struct {
	ID                  string     `gorm:"column:id;primaryKey;type:varchar(36)"`
	URL                 string     `gorm:"column:url;type:varchar(1000);not null;uniqueIndex"`
	Title               string     `gorm:"column:title;type:varchar(500);not null"`
	Content             *string    `gorm:"column:content;type:text"`
	SourceName          string     `gorm:"column:source_name;type:varchar(100);not null"`
	ExtractedLocation   *string    `gorm:"column:extracted_location;type:text"`
	ExtractedCategoryID *string    `gorm:"column:extracted_category_id;type:varchar(36)"`
	ExtractedSeverity   *string    `gorm:"column:extracted_severity;type:varchar(10)"`
	ExtractedLatitude   *float64   `gorm:"column:extracted_latitude;type:decimal(10,8)"`
	ExtractedLongitude  *float64   `gorm:"column:extracted_longitude;type:decimal(11,8)"`
	Status              string     `gorm:"column:status;type:varchar(15);not null;default:'pending'"`
	ReportID            *string    `gorm:"column:report_id;type:varchar(36)"`
	CrawledAt           *time.Time `gorm:"column:crawled_at;not null"`
	ProcessedAt         *time.Time `gorm:"column:processed_at"`
	CreatedAt           *time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt           *time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relasi lintas modul dihapus sesuai aturan AGENTS.md
}

func (CrawledArticle) TableName() string {
	return "crawled_articles"
}

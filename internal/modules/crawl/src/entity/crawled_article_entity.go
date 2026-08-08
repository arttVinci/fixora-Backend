package entity

import "time"

type CrawledArticle struct {
	ID           string     `gorm:"column:id;primaryKey;type:varchar(100)"`
	URL 		 string 	`gorm:"column:url;type:varchar(700);not null;uniqueIndex"`
	Title        string     `gorm:"column:title;type:varchar(500);not null"`
	Content      *string    `gorm:"column:content;type:text"`
	SourceName   string     `gorm:"column:source_name;type:varchar(100);not null"`
	Status       string     `gorm:"column:status;type:varchar(15);not null;default:'pending'"`
	RejectReason *string    `gorm:"column:reject_reason;type:varchar(100)"`
	ReportID     *string    `gorm:"column:report_id;type:varchar(100)"`
	PublishedAt  *time.Time `gorm:"column:published_at"`
	CrawledAt    *time.Time `gorm:"column:crawled_at;not null"`
	ProcessedAt  *time.Time `gorm:"column:processed_at"`
	CreatedAt    *time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    *time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (CrawledArticle) TableName() string {
	return "crawled_articles"
}

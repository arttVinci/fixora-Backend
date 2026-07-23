package entity

type CrawledArticle struct {
	ID                  string   `gorm:"column:id;primaryKey;type:varchar(36)"`
	URL                 string   `gorm:"column:url;type:varchar(1000);not null;uniqueIndex"`
	Title               string   `gorm:"column:title;type:varchar(500);not null"`
	Content             *string  `gorm:"column:content;type:text"`
	SourceName          string   `gorm:"column:source_name;type:varchar(100);not null"`
	ExtractedLocation   *string  `gorm:"column:extracted_location;type:text"`
	ExtractedCategoryID *string  `gorm:"column:extracted_category_id;type:varchar(36)"`
	ExtractedSeverity   *int     `gorm:"column:extracted_severity;type:smallint"`
	ExtractedLatitude   *float64 `gorm:"column:extracted_latitude;type:decimal(10,8)"`
	ExtractedLongitude  *float64 `gorm:"column:extracted_longitude;type:decimal(11,8)"`
	Status              string   `gorm:"column:status;type:varchar(15);not null;default:'pending'"`
	ReportID            *string  `gorm:"column:report_id;type:varchar(36)"`
	CrawledAt           int64    `gorm:"column:crawled_at;type:bigint;not null"`
	ProcessedAt         *int64   `gorm:"column:processed_at;type:bigint"`
	CreatedAt           int64    `gorm:"column:created_at;autoCreateTime:milli"`
	UpdatedAt           int64    `gorm:"column:updated_at;autoUpdateTime:milli"`

	// Relasi
	ExtractedCategory *Category `gorm:"foreignKey:ExtractedCategoryID;references:ID"`
	Report            *Report   `gorm:"foreignKey:ReportID;references:ID"`
}

func (CrawledArticle) TableName() string {
	return "crawled_articles"
}

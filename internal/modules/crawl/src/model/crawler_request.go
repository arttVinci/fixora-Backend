package model

import "time"

type ProcessCrawledArticleRequest struct {
	URL         string
	Title       string
	Content     string
	SourceName  string
	PublishedAt time.Time
	CrawledAt   time.Time
	CategoryID  string
	CategorySlug string
	VillageID   string
	Latitude    float64
	Longitude   float64
	Address     string
	Severity    string
}

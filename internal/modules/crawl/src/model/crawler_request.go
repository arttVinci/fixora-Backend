package model

import (
	"time"

	"github.com/arttVinci/fixora-Backend/internal/shared/client"
)

type ProcessCrawledArticleRequest struct {
	URL        string
	Title      string
	Content    string
	SourceName string
	CrawledAt  time.Time
	
	Extraction *client.ExtractionResult
	Geocode    *client.GeocodeResult
}

package worker

import (
	"context"
	"sync"
	"time"

	"github.com/arttVinci/fixora-Backend/internal/modules/crawl/src/infra"
	"github.com/arttVinci/fixora-Backend/internal/modules/crawl/src/model"
	"github.com/arttVinci/fixora-Backend/internal/modules/crawl/src/usecase"
	"github.com/arttVinci/fixora-Backend/internal/modules/crawl/src/utils"
	region_client "github.com/arttVinci/fixora-Backend/internal/modules/region-client"
	report_client "github.com/arttVinci/fixora-Backend/internal/modules/report-client"
	"github.com/arttVinci/fixora-Backend/internal/shared/client"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

type CrawlerWorker struct {
	Log             *logrus.Logger
	RssClient       infra.RssClient
	LlmClient       infra.LlmClient
	NominatimClient client.NominatimClient
	CrawlerUseCase  *usecase.CrawlerUseCase
	ReportClient    report_client.Client
	RegionClient    region_client.Client
	Cron            *cron.Cron
}

func NewCrawlerWorker(
	log *logrus.Logger,
	rss infra.RssClient,
	llm infra.LlmClient,
	nominatim client.NominatimClient,
	crawlerUseCase *usecase.CrawlerUseCase,
	reportClient report_client.Client,
	regionClient region_client.Client,
) *CrawlerWorker {
	return &CrawlerWorker{
		Log:             log,
		RssClient:       rss,
		LlmClient:       llm,
		NominatimClient: nominatim,
		CrawlerUseCase:  crawlerUseCase,
		ReportClient:    reportClient,
		RegionClient:    regionClient,
		Cron:            cron.New(),
	}
}

func (w *CrawlerWorker) StartScheduler() {
	// Run immediately on startup so we don't wait for the first cron tick
	go func() {
		w.Log.Info("Starting initial AI News Crawler run on startup...")
		w.RunCrawler()
	}()

	_, err := w.Cron.AddFunc("0 */2 * * *", func() {
		w.Log.Info("Starting scheduled AI News Crawler...")
		w.RunCrawler()
	})
	if err != nil {
		w.Log.Fatalf("Failed to schedule crawler worker: %+v", err)
	}

	w.Cron.Start()
	w.Log.Info("AI News Crawler scheduler started")
}

func (w *CrawlerWorker) RunCrawler() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	categories, err := w.ReportClient.GetAllCategories(w.CrawlerUseCase.DB.WithContext(ctx))
	if err != nil {
		w.Log.Warnf("Failed to fetch categories: %+v", err)
		return
	}

	allArticles := make(map[string]infra.RSSArticle)
	for _, category := range categories {
		articles, err := w.RssClient.FetchArticles(ctx, category.Name)
		if err != nil {
			w.Log.Warnf("Failed to fetch RSS for '%s': %+v", category.Name, err)
			continue
		}

		for _, article := range articles {
			if utils.ArticleRssFilter(article) {
				continue
			}
			if _, exists := allArticles[article.URL]; !exists {
				allArticles[article.URL] = article
			}
		}
	}

	w.processArticles(ctx, allArticles)
}

func (w *CrawlerWorker) processArticles(ctx context.Context, articles map[string]infra.RSSArticle) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5)

	for _, article := range articles {
		if w.CrawlerUseCase.IsArticleProcessed(ctx, article.URL) {
			continue
		}

		wg.Add(1)
		semaphore <- struct{}{}

		go func(art infra.RSSArticle) {
			defer wg.Done()
			defer func() { <-semaphore }()

			artCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()

			crawledAt := time.Now()
			publishedAt := parsePublishedAt(art.PublishedAt)

			extraction, err := w.LlmClient.ExtractNewsInfo(artCtx, art.Title, art.Content)
			if err != nil || extraction == nil {
				_ = w.CrawlerUseCase.SaveRejectedArticle(artCtx, art.URL, art.Title, art.Content, art.SourceName, "llm_extraction_failed", publishedAt, crawledAt)
				return
			}

			if !extraction.IsRelevant {
				_ = w.CrawlerUseCase.SaveRejectedArticle(artCtx, art.URL, art.Title, art.Content, art.SourceName, "not_relevant", publishedAt, crawledAt)
				return
			}

			category, err := w.ReportClient.GetCategoryBySlug(w.CrawlerUseCase.DB.WithContext(artCtx), extraction.Category)
			if err != nil || category == nil {
				_ = w.CrawlerUseCase.SaveRejectedArticle(artCtx, art.URL, art.Title, art.Content, art.SourceName, "category_not_found", publishedAt, crawledAt)
				return
			}

			geocode, err := w.NominatimClient.Geocode(artCtx, extraction.Location)
			if err != nil || geocode == nil {
				_ = w.CrawlerUseCase.SaveRejectedArticle(artCtx, art.URL, art.Title, art.Content, art.SourceName, "geocoding_failed", publishedAt, crawledAt)
				return
			}

			reverseGeocode, err := w.NominatimClient.ReverseGeocode(artCtx, geocode.Latitude, geocode.Longitude)
			if err != nil || reverseGeocode == nil {
				_ = w.CrawlerUseCase.SaveRejectedArticle(artCtx, art.URL, art.Title, art.Content, art.SourceName, "reverse_geocoding_failed", publishedAt, crawledAt)
				return
			}

			village, err := w.RegionClient.ResolveVillageByAddress(
				w.CrawlerUseCase.DB.WithContext(artCtx),
				reverseGeocode.Village,
				reverseGeocode.District,
				reverseGeocode.City,
				reverseGeocode.Province,
			)
			if err != nil || village == nil {
				_ = w.CrawlerUseCase.SaveRejectedArticle(artCtx, art.URL, art.Title, art.Content, art.SourceName, "village_not_resolved", publishedAt, crawledAt)
				return
			}

			req := &model.ProcessCrawledArticleRequest{
				URL:          art.URL,
				Title:        art.Title,
				Content:      art.Content,
				SourceName:   art.SourceName,
				PublishedAt:  publishedAt,
				CrawledAt:    crawledAt,
				CategoryID:   category.ID,
				CategorySlug: category.Slug,
				VillageID:    village.ID,
				Latitude:     geocode.Latitude,
				Longitude:    geocode.Longitude,
				Address:      reverseGeocode.FullAddress,
				Severity:     extraction.Severity,
			}

			if err := w.CrawlerUseCase.SaveCrawledReport(artCtx, req); err != nil {
				w.Log.Warnf("Failed to save crawled report: %+v", err)
			}
		}(article)
	}

	wg.Wait()
}

func parsePublishedAt(value string) time.Time {
	if value == "" {
		return time.Now()
	}

	layouts := []string{time.RFC1123Z, time.RFC1123, time.RFC3339, "Mon, 02 Jan 2006 15:04:05 MST"}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed
		}
	}

	return time.Now()
}

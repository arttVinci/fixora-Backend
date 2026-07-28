package worker

import (
	"context"
	"sync"
	"time"

	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/model"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/usecase"
	"github.com/arttVinci/fixora-Backend/internal/shared/client"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

type CrawlerWorker struct {
	Log             *logrus.Logger
	RssClient       client.RssClient
	LlmClient       client.LlmClient
	NominatimClient client.NominatimClient
	CrawlerUseCase  *usecase.CrawlerUseCase
	Cron            *cron.Cron
}

func NewCrawlerWorker(
	log *logrus.Logger,
	rss client.RssClient,
	llm client.LlmClient,
	nominatim client.NominatimClient,
	usecase *usecase.CrawlerUseCase,
) *CrawlerWorker {
	return &CrawlerWorker{
		Log:             log,
		RssClient:       rss,
		LlmClient:       llm,
		NominatimClient: nominatim,
		CrawlerUseCase:  usecase,
		Cron:            cron.New(),
	}
}

// StartScheduler memulai cron job di background
func (w *CrawlerWorker) StartScheduler() {
	// Menjalankan crawler setiap 2 jam
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
	ctx := context.Background()
	keywords := []string{"jalan rusak", "jembatan rusak", "sampah menumpuk", "bangunan terbengkalai", "drainase tersumbat"}

	for _, keyword := range keywords {
		articles, err := w.RssClient.FetchArticles(ctx, keyword)
		if err != nil {
			w.Log.Warnf("Crawler failed to fetch articles for keyword '%s': %+v", keyword, err)
			continue
		}

		w.processArticlesConcurrently(ctx, articles)
	}
}

func (w *CrawlerWorker) processArticlesConcurrently(ctx context.Context, articles []client.RSSArticle) {
	var wg sync.WaitGroup
	// Menggunakan channel sebagai semaphore untuk membatasi maksimal 5 goroutine berjalan bersamaan
	// agar tidak terkena rate limit dari API LLM atau Nominatim
	semaphore := make(chan struct{}, 5)

	for _, article := range articles {
		// 1. Cek duplikasi (Fase 2) - Eksekusi sinkron karena ini cuma 1 query DB cepat
		if w.CrawlerUseCase.IsArticleProcessed(ctx, article.URL) {
			continue
		}

		wg.Add(1)
		semaphore <- struct{}{} // Acquire token

		go func(art client.RSSArticle) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release token

			// 2. Ekstraksi LLM (Fase 3)
			extraction, err := w.LlmClient.ExtractNewsInfo(ctx, art.Title, art.Content)
			if err != nil || extraction == nil {
				w.Log.Warnf("Failed to extract info for article %s: %+v", art.URL, err)
				return
			}

			// Abaikan jika berita tidak relevan dengan infrastruktur
			if !extraction.IsRelevant {
				return
			}

			// 3. Geocoding (Fase 4)
			geocode, err := w.NominatimClient.Geocode(ctx, extraction.Location)
			if err != nil || geocode == nil {
				w.Log.Warnf("Failed to geocode location '%s': %+v", extraction.Location, err)
				return
			}

			// 4. Simpan ke database (Fase 5 & 6)
			req := &model.ProcessCrawledArticleRequest{
				URL:        art.URL,
				Title:      art.Title,
				Content:    art.Content,
				SourceName: art.SourceName,
				CrawledAt:  time.Now(),
				Extraction: extraction,
				Geocode:    geocode,
			}

			if err := w.CrawlerUseCase.SaveCrawledReport(ctx, req); err != nil {
				w.Log.Warnf("Failed to save crawled report for %s: %+v", art.URL, err)
			} else {
				w.Log.Infof("Successfully processed and saved crawled article: %s", art.Title)
			}

		}(article)
	}

	wg.Wait() // Tunggu semua artikel di keyword ini selesai sebelum lanjut ke keyword berikutnya
}

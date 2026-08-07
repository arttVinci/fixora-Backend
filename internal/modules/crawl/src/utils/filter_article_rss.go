package utils

import (
	"strings"
	"time"

	"github.com/arttVinci/fixora-Backend/internal/modules/crawl/src/infra"
)

func ArticleRssFilter(article infra.RSSArticle) bool {
	// Skip artikel lebih dari 3 hari
	if pubDate, err := time.Parse(time.RFC1123Z, article.PublishedAt); err == nil {
		if pubDate.Before(time.Now().AddDate(0, 0, -3)) {
			return true
		}
	}

	var excludeWords = []string{
    // Opini & sejenisnya — tulisan pandangan pribadi
    "opini", "kolom", "kolomnis", "editorial", "tajuk rencana",
    "sudut pandang", "perspektif", "esai", "komentar", "refleksi",
    "catatan redaksi", "suara pembaca", "surat pembaca",

    // Analisis mendalam — bukan laporan kejadian spesifik
    "analisis", "ulasan", "resensi",

    // Wawancara — subjektif, bukan laporan kondisi lapangan
    "wawancara", "bincang", "dialog eksklusif",

    // Feature/human interest — biasanya bukan tentang kondisi infrastruktur spesifik
    "feature", "kisah", "cerita di balik",

    // Fact-check/debunking — MEMBAHAS topik tapi isinya klarifikasi, bukan laporan baru
    "cek fakta", "hoaks", "hoax", "klarifikasi",

    // Retrospektif/nostalgia — bukan kondisi TERKINI
    "kilas balik", "flashback", "tahun lalu", "dulu",
}
	titleLower := strings.ToLower(article.Title)
	for _, w := range excludeWords {
		if strings.Contains(titleLower, w) {
			return true
		}
	}
	return false
}
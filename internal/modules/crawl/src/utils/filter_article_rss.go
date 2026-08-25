package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/arttVinci/fixora-Backend/internal/modules/crawl/src/infra"
)

func ArticleRssFilter(article infra.RSSArticle) bool {
	// Skip artikel lebih dari 3 hari.
	// Google News RSS mengirim tanggal format "Mon, 02 Jan 2006 15:04:05 GMT" —
	// time.RFC1123 (bukan RFC1123Z) yang sanggup parse suffix "GMT".
	if pubDate, err := parseRSSDate(article.PublishedAt); err == nil {
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

// parseRSSDate mencoba berbagai layout tanggal RSS.
// Layout pertama yang cocok menang; error dikembalikan bila semua gagal
// sehingga caller (ArticleRssFilter) bisa memutuskan fallback-nya.
func parseRSSDate(value string) (time.Time, error) {
	layouts := []string{
		time.RFC1123, // "Mon, 02 Jan 2006 15:04:05 GMT" — format Google News
		time.RFC1123Z,
		time.RFC3339,
		"Mon, 2 Jan 2006 15:04:05 MST",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %q", value)
}
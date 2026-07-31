package infra

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
	"time"
)

func GenerateArticleID(sourceName string) string {
	return "ART-" + slugify(sourceName) + "-" + time.Now().Format("20060102") + "-" + randomHex(2)
}

func GenerateReportID(categorySlug string) string {
	return "RPT-" + slugify(categorySlug) + "-" + time.Now().Format("20060102") + "-" + randomHex(2)
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	re := regexp.MustCompile(`[^a-z0-9]+`)
	value = re.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "unknown"
	}
	return value
}

func randomHex(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return time.Now().Format("150405")
	}
	return hex.EncodeToString(buf)
}

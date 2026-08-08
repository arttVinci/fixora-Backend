package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

type ExtractionResult struct {
	Location   string `json:"location"`
	Category   string `json:"category"`
	Severity   string `json:"severity"`
	IsRelevant bool   `json:"is_relevant"`
}

type LlmClient interface {
	ExtractNewsInfo(ctx context.Context, title, content string) (*ExtractionResult, error)
}

type llmClientImpl struct {
	Log   *logrus.Logger
	Genai *genai.Client
}

func NewLlmClient(log *logrus.Logger, genaiClient *genai.Client) LlmClient {
	return &llmClientImpl{
		Log:   log,
		Genai: genaiClient,
	}
}

const maxRetries = 3

var llmLimiter = rate.NewLimiter(rate.Every(15*time.Second), 1)

func Limitter(ctx context.Context) error {
	return llmLimiter.Wait(ctx)
}

func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
    return strings.Contains(err.Error(), "429") || 
           strings.Contains(err.Error(), "QuotaFailure")
}

func (c *llmClientImpl) ExtractNewsInfo(ctx context.Context, title, content string) (*ExtractionResult, error) {
	model := c.Genai.GenerativeModel("gemini-3.5-flash-lite")
	model.ResponseSchema = &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"location": {
				Type:        genai.TypeString,
				Description: "Lokasi spesifik tempat masalah infrastruktur terjadi. Kosongkan jika terlalu ambigu atau general.",
			},
			"category": {
				Type:        genai.TypeString,
				Description: "Slug kategori masalah: jalan-rusak, sampah, drainase-tersumbat, jembatan-rusak, bangunan-terbengkalai",
			},
			"severity": {
				Type:        genai.TypeString,
				Description: "Tingkat keparahan masalah: ringan, sedang, atau parah.",
			},
			"is_relevant": {
				Type:        genai.TypeBoolean,
				Description: "Apakah berita mendeskripsikan masalah infrastruktur spesifik saat ini.",
			},
		},
		Required: []string{"location", "category", "severity", "is_relevant"},
	}
	model.ResponseMIMEType = "application/json"

	prompt := fmt.Sprintf(`Tugas kamu adalah mengekstrak informasi masalah infrastruktur dari berita.

Judul Berita: %s
Isi Berita: %s

Panduan:
1. is_relevant harus true hanya jika berita melaporkan kerusakan nyata saat ini.
2. location harus alamat/area spesifik. Jika hanya provinsi/negara, is_relevant false.
3. category wajib salah satu slug: jalan-rusak, sampah, drainase-tersumbat, jembatan-rusak, bangunan-terbengkalai.
4. severity wajib salah satu: ringan, sedang, parah.

Balas JSON sesuai schema.`, title, content)
    var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {

		if err := Limitter(ctx); err != nil {
            return nil, fmt.Errorf("rate limiter error: %w", err)
        }

		resp, err := model.GenerateContent(ctx, genai.Text(prompt))
		if err != nil {
			lastErr = err

			// Cek apakah errornya karena Rate Limit (429)
			if isRateLimitError(err) {
				backoff := time.Duration(15*(attempt+1)) * time.Second
				c.Log.Warnf("LLM Rate limit hit: %v. Retrying attempt %d/%d after %v...", err, attempt+1, maxRetries, backoff)

				// Context-aware sleep — don't hang if context is already cancelled
				select {
				case <-ctx.Done():
					c.Log.Warnf("Context cancelled during rate limit backoff: %+v", ctx.Err())
					return nil, ctx.Err()
				case <-time.After(backoff):
				}
				continue
			}

			// Kalau error lain (bukan 429), langsung return
			c.Log.Warnf("Failed to generate content from LLM: %+v", err)
			return nil, err
		}

		if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
			return nil, fmt.Errorf("no content returned from LLM")
		}

		part := resp.Candidates[0].Content.Parts[0]
		text, ok := part.(genai.Text)
		if !ok {
			return nil, fmt.Errorf("unexpected LLM response format")
		}

		jsonStr := strings.TrimPrefix(string(text), "```json\n")
		jsonStr = strings.TrimSuffix(jsonStr, "\n```")
		jsonStr = strings.TrimSpace(jsonStr)

		var result ExtractionResult
		if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
			c.Log.Warnf("Failed to unmarshal JSON from LLM: %+v (String: %s)", err, jsonStr)
			return nil, err
		}
		
		c.Log.Infof("LLM extraction successful for '%s': relevant=%v", title, result.IsRelevant)
		return &result, nil
	}
	return nil, lastErr
}

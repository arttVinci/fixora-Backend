package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/sirupsen/logrus"
)

type ExtractionResult struct {
	Location   string `json:"location"`
	CategoryID string `json:"category_id"`
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

func (c *llmClientImpl) ExtractNewsInfo(ctx context.Context, title, content string) (*ExtractionResult, error) {
	// Configure the model
	model := c.Genai.GenerativeModel("gemini-1.5-pro")
	
	// Define structured output schema
	model.ResponseSchema = &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"location": {
				Type:        genai.TypeString,
				Description: "Lokasi jalan, jembatan, bangunan, atau area spesifik tempat masalah infrastruktur terjadi. Kosongkan jika terlalu ambigu atau general.",
			},
			"category_id": {
				Type:        genai.TypeString,
				Description: "ID kategori masalah (pilih salah satu dari: jalan, jembatan, drainase, sampah, bangunan, lainnya)",
			},
			"severity": {
				Type:        genai.TypeString,
				Description: "Tingkat keparahan masalah (ringan, sedang, parah)",
			},
			"is_relevant": {
				Type:        genai.TypeBoolean,
				Description: "Apakah berita ini mendeskripsikan masalah infrastruktur yang spesifik (bukan sekedar rencana, opini, wacana anggaran, atau politik)?",
			},
		},
		Required: []string{"location", "category_id", "severity", "is_relevant"},
	}
	model.ResponseMIMEType = "application/json"

	// Define the prompt
	prompt := fmt.Sprintf(`Tugas kamu adalah mengekstrak informasi masalah infrastruktur dari sebuah berita.
	
Judul Berita: %s
Isi Berita: %s

Panduan Ekstraksi:
1. is_relevant: Pastikan beritanya melaporkan kerusakan nyata saat ini (lubang, ambruk, menumpuk, dll), bukan wacana, politik, atau opini.
2. location: Cari alamat spesifik (misal "Jl. Gatot Subroto, Jakarta", "Jembatan Musi, Palembang"). Jika hanya menyebut provinsi atau negara, anggap tidak relevan.
3. category_id: Klasifikasikan ke salah satu dari:
   - "jalan" (Jalan berlubang, rusak, aspal mengelupas)
   - "jembatan" (Jembatan putus, rawan roboh, rusak)
   - "drainase" (Drainase tersumbat, banjir karena gorong-gorong)
   - "sampah" (Sampah menumpuk, TPS liar)
   - "bangunan" (Bangunan publik terbengkalai/rusak)
   - "lainnya"
4. severity: Evaluasi keparahan: "ringan", "sedang", atau "parah".

Berikan hasil dalam format JSON persis sesuai schema.`, title, content)

	// Execute LLM call
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		c.Log.Warnf("Failed to generate content from LLM: %+v", err)
		return nil, err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content returned from LLM")
	}

	// Parse JSON
	part := resp.Candidates[0].Content.Parts[0]
	text, ok := part.(genai.Text)
	if !ok {
		return nil, fmt.Errorf("unexpected LLM response format")
	}
	
	// Clean markdown json ticks if present
	jsonStr := strings.TrimPrefix(string(text), "```json\n")
	jsonStr = strings.TrimSuffix(jsonStr, "\n```")
	jsonStr = strings.TrimSpace(jsonStr)

	var result ExtractionResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		c.Log.Warnf("Failed to unmarshal JSON from LLM: %+v (String: %s)", err, jsonStr)
		return nil, err
	}

	c.Log.Infof("LLM Extraction successful for '%s': Relevant=%v", title, result.IsRelevant)
	return &result, nil
}

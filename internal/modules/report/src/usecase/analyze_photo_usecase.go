package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/model"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/generative-ai-go/genai"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type AnalyzePhotoUseCase struct {
	DB       *gorm.DB
	Log      *logrus.Logger
	Validate *validator.Validate
	Genai    *genai.Client
}

func NewAnalyzePhotoUseCase(
	db *gorm.DB,
	log *logrus.Logger,
	validate *validator.Validate,
	genai *genai.Client,
) *AnalyzePhotoUseCase {
	return &AnalyzePhotoUseCase{
		DB:       db,
		Log:      log,
		Validate: validate,
		Genai:    genai,
	}
}

func (i *AnalyzePhotoUseCase) AnalyzeIssueImage(ctx context.Context, image *multipart.FileHeader) (*model.IssueAnalysisResultResponse, error) {
	file, err := image.Open()
	if err != nil {
		i.Log.Warnf("Failed to open image file : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal membaca file gambar")
	}
	defer file.Close()

	imgBytes, err := io.ReadAll(file)
	if err != nil {
		i.Log.Warnf("Failed to read image file : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal membaca file gambar")
	}

	extension := strings.TrimPrefix(filepath.Ext(image.Filename), ".")

	response, err := i.AnalyzeImage(ctx, imgBytes, extension)
	if err != nil {
		i.Log.Warnf("Failed to analyze issue image : %+v", err)
		return nil, err
	}

	return response, nil
}

func (i *AnalyzePhotoUseCase) AnalyzeImage(ctx context.Context, image []byte, extension string) (*model.IssueAnalysisResultResponse, error) {
	llm := i.Genai.GenerativeModel("gemini-3.5-flash-lite")
	llm.ResponseSchema = &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"title": {
				Type:        genai.TypeString,
				Description: "Judul spesifik tempat masalah infrastruktur terjadi.",
			},
			"description": {
				Type:        genai.TypeString,
				Description: "Jelaskan deskripsi masalah dari gambar tersebut",
			},
			"category": {
				Type:        genai.TypeString,
				Description: "Slug kategori masalah: jalan-rusak, sampah, jembatan-rusak, bangunan-terbengkalai",
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
		Required: []string{"title", "description", "category", "severity", "is_relevant"},
	}
	llm.ResponseMIMEType = "application/json"

	prompt := `Kamu adalah sistem yang menganalisis foto laporan masalah infrastruktur publik di Indonesia.

		Dari foto yang diberikan, analisis dan hasilkan data sesuai skema yang ditentukan.

		Panduan kategori:
		- jalan-rusak: lubang, retak, aspal terkelupas, jalan ambles
		- sampah: tumpukan sampah, TPS liar, sampah berserakan
		- jembatan-rusak: kerusakan struktur jembatan, retak, korosi
		- bangunan-terbengkalai: bangunan tidak terawat, terbengkalai, rusak dan dibiarkan

		Panduan severity:
		- ringan: kerusakan kecil, belum mengganggu aktivitas secara signifikan
		- sedang: kerusakan cukup terlihat, mulai mengganggu aktivitas warga
		- parah: kerusakan signifikan, berpotensi membahayakan atau sangat mengganggu

		Jika foto tidak jelas menunjukkan salah satu dari 4 kategori di atas, atau foto tidak 
		relevan dengan masalah infrastruktur (misalnya foto orang, makanan, dokumen, atau 
		pemandangan tanpa kerusakan yang jelas), set is_relevant menjadi false, dan tetap isi 
		field lain dengan estimasi terbaik atau nilai default yang masuk akal.

		Jangan mengarang detail yang tidak terlihat di foto (nama lokasi, penyebab kerusakan, 
		durasi masalah) — foto ini TIDAK memuat informasi lokasi, itu akan didapat dari sumber lain.`

	response, err := llm.GenerateContent(ctx, genai.Text(prompt), genai.ImageData(extension, image))
	if err != nil {
		i.Log.Warnf("Failed to generate content from LLM: %+v", err)
		return nil, err
	}

	if len(response.Candidates) == 0 || len(response.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content returned from LLM")
	}

	part := response.Candidates[0].Content.Parts[0]
	text, ok := part.(genai.Text)
	if !ok {
		return nil, fmt.Errorf("unexpected LLM response format")
	}

	jsonStr := strings.TrimPrefix(string(text), "```json\n")
	jsonStr = strings.TrimSuffix(jsonStr, "\n```")
	jsonStr = strings.TrimSpace(jsonStr)

	result := new(model.IssueAnalysisResultResponse)

	if err := json.Unmarshal([]byte(jsonStr), result); err != nil {
		i.Log.Warnf("Failed to unmarshal JSON from LLM: %+v (String: %s)", err, jsonStr)
		return nil, err
	}

	return result, nil
}

package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/model"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/repository"
	"github.com/arttVinci/fixora-Backend/internal/shared/client"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/generative-ai-go/genai"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type AnalyzePhotoUseCase struct {
	DB                 *gorm.DB
	Log                *logrus.Logger
	Validate           *validator.Validate
	Genai              *genai.Client
	Cloudinary         *client.CloudinaryClient
	CategoryRepository *repository.CategoryRepository
}

func NewAnalyzePhotoUseCase(
	db *gorm.DB,
	log *logrus.Logger,
	validate *validator.Validate,
	genai *genai.Client,
	cloudinary *client.CloudinaryClient,
	categoryRepo *repository.CategoryRepository,
) *AnalyzePhotoUseCase {
	return &AnalyzePhotoUseCase{
		DB:                 db,
		Log:                log,
		Validate:           validate,
		Genai:              genai,
		Cloudinary:         cloudinary,
		CategoryRepository: categoryRepo,
	}
}

func (i *AnalyzePhotoUseCase) AnalyzeIssueImage(ctx context.Context, image *multipart.FileHeader) (*model.IssueAnalysisResultResponse, error) {
	sessionID := uuid.NewString()

	if i.Cloudinary == nil {
		i.Log.Warnf("Cloudinary is not configured, photo will not be staged")
		return nil, fiber.NewError(fiber.StatusServiceUnavailable, "Penyimpanan foto belum dikonfigurasi")
	}

	staged, err := i.Cloudinary.UploadStaged(ctx, image, client.StagingPublicID(sessionID, client.PrimarySlot))
	if err != nil {
		i.Log.Warnf("Failed to stage photo : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengunggah foto")
	}

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

	response.SessionID = sessionID
	response.PhotoURL = staged.SecureURL

	return response, nil
}

func (i *AnalyzePhotoUseCase) AnalyzeImage(ctx context.Context, image []byte, extension string) (*model.IssueAnalysisResultResponse, error) {
	llm := i.Genai.GenerativeModel("gemini-3.5-flash-lite")
	llm.ResponseSchema = &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"title": {
				Type:        genai.TypeString,
				Description: "Judul singkat masalah infrastruktur.",
			},
			"description": {
				Type:        genai.TypeString,
				Description: "Deskripsi singkat masalah dari foto.",
			},
			"category": {
				Type:        genai.TypeString,
				Description: "Slug kategori masalah: jalan-rusak, sampah, jembatan-rusak, bangunan-terbengkalai. Kosongkan jika is_relevant false.",
			},
			"severity": {
				Type:        genai.TypeString,
				Description: "Tingkat keparahan masalah: ringan, sedang, atau parah. Kosongkan jika is_relevant false.",
			},
			"is_relevant": {
				Type:        genai.TypeBoolean,
				Description: "Apakah foto menunjukkan kerusakan infrastruktur publik yang nyata dan termasuk salah satu dari 4 kategori.",
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

PENENTUAN is_relevant (WAJIB DIPATUHI, ini penentu utama):
- is_relevant = true HANYA jika foto secara jelas menunjukkan kerusakan fisik infrastruktur
  publik yang nyata dan cocok dengan salah satu dari 4 kategori di atas.
- is_relevant = false untuk SEMUA kondisi berikut:
  * Bukan infrastruktur publik: foto orang/selfie, hewan, makanan/minuman, barang pribadi,
    tangkapan layar (screenshot) aplikasi/chat/media sosial, dokumen/teks, struk, logo.
  * Infrastruktur tetapi TIDAK rusak/bermasalah (jalan mulus, gedung terawat, jalan layang
    normal, tiang listrik utuh, rambu utuh, dll).
  * Foto terlalu gelap, blur, terpotong, atau objek tidak dapat dikenali.
  * Pemandangan alam/gunung/pantai/langit tanpa objek infrastruktur yang rusak.
- Jika ragu antara relevant dan tidak, pilih is_relevant = false.
- Ketika is_relevant = false, category WAJIB diisi "" (string kosong) dan severity WAJIB
  diisi "" (string kosong). title dan description boleh diisi ringkas alasan foto ditolak.

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

	// Guard: a photo is only usable when it maps to an existing category.
	// A non-relevant photo, or one the LLM mislabels with an unknown slug,
	// must not leak a prefillable category/severity to the frontend.
	if !result.IsRelevant || result.Category == "" {
		result.IsRelevant = false
		result.Category = ""
		result.Severity = ""
		return result, nil
	}

	category := new(entity.Category)
	if err := i.CategoryRepository.FindBySlug(i.DB.WithContext(ctx), category, result.Category); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			i.Log.Warnf("Unknown category slug '%s' from LLM", result.Category)
			result.IsRelevant = false
			result.Category = ""
			result.Severity = ""
			return result, nil
		}
		i.Log.Warnf("Failed to lookup category slug '%s' : %+v", result.Category, err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal memvalidasi kategori foto")
	}

	return result, nil
}

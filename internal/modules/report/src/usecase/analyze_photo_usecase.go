package usecase

import (
	"context"
	"encoding/json"
	"errors"
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
	NominatimClient    client.NominatimClient
	CategoryRepository *repository.CategoryRepository
}

func NewAnalyzePhotoUseCase(
	db *gorm.DB,
	log *logrus.Logger,
	validate *validator.Validate,
	genai *genai.Client,
	cloudinary *client.CloudinaryClient,
	nominatimClient client.NominatimClient,
	categoryRepo *repository.CategoryRepository,
) *AnalyzePhotoUseCase {
	return &AnalyzePhotoUseCase{
		DB:                 db,
		Log:                log,
		Validate:           validate,
		Genai:              genai,
		Cloudinary:         cloudinary,
		NominatimClient:    nominatimClient,
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

// analyzePhotoLLMResult adalah kontrak JSON output dari vision LLM.
// Dipisah dari model.IssueAnalysisResultResponse karena membawa sinyal internal
// has_timestamp_overlay yang tidak diekspos ke frontend.
type analyzePhotoLLMResult struct {
	Location            string `json:"location"`
	Title               string `json:"title"`
	Description         string `json:"description"`
	Category            string `json:"category"`
	Severity            string `json:"severity"`
	IsRelevant          bool   `json:"is_relevant"`
	Reason              string `json:"reason"`
	HasTimestampOverlay bool   `json:"has_timestamp_overlay"`
}

func (i *AnalyzePhotoUseCase) AnalyzeImage(ctx context.Context, image []byte, extension string) (*model.IssueAnalysisResultResponse, error) {
	llm := i.Genai.GenerativeModel("gemini-3.5-flash-lite")
	llm.ResponseSchema = &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"has_timestamp_overlay": {
				Type:        genai.TypeBoolean,
				Description: "TRUE jika dan hanya jika foto memiliki stempel (overlay) berisi teks TANGGAL + WAKTU yang ter-burn di gambar (aplikasi kamera timestamp/CCTV/dashcam). FALSE jika tidak ada stempel tanggal/waktu sama sekali.",
			},
			"location": {
				Type:        genai.TypeString,
				Description: "Teks lokasi yang tercetak/terbaca di foto (alamat, nama tempat, koordinat GPS). Wajib string kosong jika tidak ada teks lokasi di foto.",
			},
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
				Description: "Slug kategori masalah: jalan-rusak, sampah, jembatan-rusak, bangunan-terbengkalai. Kosongkan jika foto tidak valid.",
			},
			"severity": {
				Type:        genai.TypeString,
				Description: "Tingkat keparahan masalah: ringan, sedang, atau parah. Kosongkan jika foto tidak valid.",
			},
			"is_relevant": {
				Type:        genai.TypeBoolean,
				Description: "Apakah foto menunjukkan kerusakan infrastruktur publik yang nyata dan termasuk salah satu dari 4 kategori.",
			},
			"reason": {
				Type:        genai.TypeString,
				Description: "Alasan penolakan singkat bila foto tidak valid. Kosongkan jika valid.",
			},
		},
		Required: []string{"has_timestamp_overlay", "location", "title", "description", "category", "severity", "is_relevant", "reason"},
	}
	llm.ResponseMIMEType = "application/json"

	prompt := `Kamu adalah sistem yang menganalisis foto laporan masalah infrastruktur publik di Indonesia.

Dari foto yang diberikan, analisis dan hasilkan data sesuai skema yang ditentukan.

ATURAN PALING PENTING — STAMPEL KAMERA (OVERLAY TANGGAL & WAKTU):
- Foto yang SAH hanya foto yang diambil dengan aplikasi kamera timestamp/CCTV/dashcam, yaitu
  foto yang di dalam gambarnya TER-BURN teks tanggal + waktu (mis. "2026-08-31 10:15:22" di pojok).
- Baca dengan teliti: apakah ada teks tanggal + waktu yang tercetak permanen di gambar?
  * ADA → has_timestamp_overlay = true.
  * TIDAK ADA / tidak terbaca / blur → has_timestamp_overlay = false.
- Jika has_timestamp_overlay = false → foto TIDAK VALID: is_relevant = false,
  reason = "bukan foto timestamp camera (tidak ada stempel tanggal & waktu)", category = "", severity = "".

LOKASI:
- Jika ada teks lokasi/koordinat GPS tercetak di foto, tulis PERSIS di field "location".
- Jika TIDAK ada teks lokasi, "location" WAJIB string kosong. JANGAN PERNAH mengarang atau
  menebak lokasi dari konteks foto. Lokasi hanya boleh diambil dari teks yang terlihat.

Panduan kategori:
- jalan-rusak: lubang, retak, aspal terkelupas, jalan ambles
- sampah: tumpukan sampah, TPS liar, sampah berserakan
- jembatan-rusak: kerusakan struktur jembatan, retak, korosi
- bangunan-terbengkalai: bangunan tidak terawat, terbengkalai, rusak dan dibiarkan

Panduan severity:
- ringan: kerusakan kecil, belum mengganggu aktivitas secara signifikan
- sedang: kerusakan cukup terlihat, mulai mengganggu aktivitas warga
- parah: kerusakan signifikan, berpotensi membahayakan atau sangat mengganggu

PENENTUAN is_relevant (WAJIB DIPATUHI):
- is_relevant = true HANYA jika (1) has_timestamp_overlay = true, DAN (2) ada teks lokasi terbaca,
  DAN (3) foto secara jelas menunjukkan kerusakan fisik infrastruktur publik yang nyata
  dan cocok dengan salah satu dari 4 kategori.
- is_relevant = false untuk SEMUA kondisi berikut:
  * Tidak ada stempel tanggal & waktu (has_timestamp_overlay = false).
  * Tidak ada teks lokasi di foto.
  * Bukan infrastruktur publik: foto orang/selfie, hewan, makanan/minuman, barang pribadi,
    tangkapan layar (screenshot), dokumen/teks, struk, logo.
  * Infrastruktur tetapi TIDAK rusak/bermasalah.
  * Foto terlalu gelap, blur, terpotong, atau objek tidak dapat dikenali.
  * Pemandangan alam tanpa objek infrastruktur rusak.
- Jika ragu antara relevant dan tidak, pilih is_relevant = false.
- Ketika is_relevant = false, category WAJIB "" dan severity WAJIB "". Isi "reason" dengan alasan
  penolakan singkat. title dan description boleh diisi ringkas alasan foto ditolak.

Jangan mengarang detail yang tidak terlihat di foto.`

	response, err := llm.GenerateContent(ctx, genai.Text(prompt), genai.ImageData(extension, image))
	if err != nil {
		i.Log.Warnf("Failed to generate content from LLM: %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal menganalisis foto")
	}

	if len(response.Candidates) == 0 || len(response.Candidates[0].Content.Parts) == 0 {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal menganalisis foto")
	}

	part := response.Candidates[0].Content.Parts[0]
	text, ok := part.(genai.Text)
	if !ok {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal menganalisis foto")
	}

	jsonStr := strings.TrimPrefix(string(text), "```json\n")
	jsonStr = strings.TrimSuffix(jsonStr, "\n```")
	jsonStr = strings.TrimSpace(jsonStr)

	llmResult := new(analyzePhotoLLMResult)
	if err := json.Unmarshal([]byte(jsonStr), llmResult); err != nil {
		i.Log.Warnf("Failed to unmarshal JSON from LLM: %+v (String: %s)", err, jsonStr)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal menganalisis foto")
	}

	result := &model.IssueAnalysisResultResponse{
		Title:       llmResult.Title,
		Description: llmResult.Description,
		Category:    llmResult.Category,
		Severity:    llmResult.Severity,
		Location:    llmResult.Location,
		Reason:      llmResult.Reason,
		IsRelevant:  llmResult.IsRelevant,
	}

	// Guard 0 (paling ketat): foto WAJIB punya stempel tanggal & waktu dari
	// aplikasi kamera timestamp/CCTV/dashcam. Ini penjaga utama supaya foto
	// biasa (tanpa overlay) tidak lolos, walau LLM sempat menghalusinasi lokasi.
	if !llmResult.HasTimestampOverlay {
		i.Log.Warnf("Photo rejected: no timestamp overlay detected")
		result.IsRelevant = false
		result.Category = ""
		result.Severity = ""
		result.Location = ""
		if result.Reason == "" {
			result.Reason = "bukan foto timestamp camera (tidak ada stempel tanggal & waktu)"
		}
		return result, nil
	}

	// Guard 1: foto harus relevan (kerusakan infrastruktur publik nyata).
	if !result.IsRelevant || result.Category == "" {
		result.IsRelevant = false
		result.Category = ""
		result.Severity = ""
		return result, nil
	}

	// Guard 2: kategori harus terdaftar (tidak boleh slug ngawur).
	category := new(entity.Category)
	if err := i.CategoryRepository.FindBySlug(i.DB.WithContext(ctx), category, result.Category); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			i.Log.Warnf("Unknown category slug '%s' from LLM", result.Category)
			result.IsRelevant = false
			result.Category = ""
			result.Severity = ""
			if result.Reason == "" {
				result.Reason = "kategori tidak dikenali"
			}
			return result, nil
		}
		i.Log.Warnf("Failed to lookup category slug '%s' : %+v", result.Category, err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal memvalidasi kategori foto")
	}

	// Guard 3: lokasi wajib terbaca dari foto lalu di-geocode menjadi lat/lng.
	result.Location = strings.TrimSpace(result.Location)
	if result.Location == "" {
		i.Log.Warnf("Location missing on photo")
		result.IsRelevant = false
		result.Category = ""
		result.Severity = ""
		if result.Reason == "" {
			result.Reason = "lokasi tidak terbaca"
		}
		return result, nil
	}

	geocode, err := i.NominatimClient.Geocode(ctx, result.Location)
	if err != nil || geocode == nil {
		i.Log.Warnf("Failed to geocode location %q : %+v", result.Location, err)
		result.IsRelevant = false
		result.Category = ""
		result.Severity = ""
		if result.Reason == "" {
			result.Reason = "lokasi tidak dapat diidentifikasi"
		}
		return result, nil
	}

	result.Latitude = geocode.Latitude
	result.Longitude = geocode.Longitude
	result.Address = geocode.Address

	return result, nil
}

# AGENTS.md

Dokumen ini adalah panduan wajib bagi siapa pun (manusia atau AI agent) yang menulis atau mengubah kode di backend ini. Referensi tambahan: `docs/fixora-prd`.

---

## 1. Tentang Proyek

**Fixora — Infrastructure Neglect Tracker** adalah platform open source untuk melacak akuntabilitas jangka panjang terhadap infrastruktur publik yang dibiarkan rusak (jalan berlubang, jembatan rawan roboh, bangunan terbengkalai, sampah menumpuk, drainase tersumbat), khususnya di kota-kota besar Indonesia seperti Jakarta dan Bekasi.

Berbeda dari platform pelaporan eksisting (Qlue, LAPOR!) yang bersifat reaktif dan "lapor sekali, selesai", Fixora:

- Secara **aktif mencari** isu infrastruktur lewat AI news crawler (tidak hanya menunggu laporan warga), sehingga punya data sejak hari pertama dan tidak terjebak cold-start problem.
- **Melacak durasi** masalah dibiarkan, bukan hanya mencatat satu laporan tunggal.
- Menyediakan **verifikasi berlapis** (AI classifier + crowdsource) agar data yang tampil kredibel.
- (Fase 2+) Menghubungkan laporan dengan data anggaran resmi pemerintah untuk bukti akuntabilitas.

Sumber data platform berasal dari dua jalur yang berjalan paralel — **laporan manual warga** (foto + lokasi + deskripsi) dan **entry otonom dari AI news crawler** (hasil ekstraksi berita) — keduanya ditampilkan di peta yang sama, dibedakan lewat badge sumber data ("Laporan Warga" vs "Terdeteksi AI/Media").

Backend ditulis dengan **Go (Golang)**, menggunakan **Fiber** sebagai HTTP framework, **GORM** sebagai ORM ke **PostgreSQL**, dengan struktur layer: `controller` → `usecase` → `repository` → `entity`, dilengkapi `model` (DTO) dan `converter` (entity ↔ DTO). Frontend (repo terpisah, `fixora-fe`) menggunakan React + TypeScript dan tidak dicakup oleh dokumen ini.

**Cakupan MVP** (ringkas, lihat PRD lengkap di `docs/fixora-prd` untuk detail): peta interaktif, pelaporan manual, detail & riwayat titik masalah, konfirmasi status "masih begini", AI news crawler, CV classifier & severity scoring dari foto, auto-flagging duplikat. Fitur seperti multi-agent verification kompleks, RAG cross-reference anggaran pemerintah, predictive risk scoring, satellite imagery analysis, dan agentic auto-generate surat pengaduan **belum masuk MVP** — jangan diimplementasikan tanpa approval eksplisit dari owner (lihat prinsip di Bab 3).

---

## 2. Arsitektur Sistem

### 2.1 Pola Arsitektur: Modular Monolith

Backend dibangun dengan pola **Modular Monolith** — satu aplikasi/deployment unit, namun kode dipecah menjadi modul-modul yang terisolasi secara data dan hanya saling berkomunikasi lewat interface (client) yang eksplisit. Pola ini dipilih karena tim backend masih kecil (solo/awal), sehingga kompleksitas operasional microservices belum diperlukan, tapi struktur modular tetap memudahkan pemisahan tanggung jawab dan migrasi ke microservices di masa depan bila dibutuhkan.

### 2.2 Prinsip Utama

- **Data Isolation**: setiap modul punya tabel sendiri. **Tidak boleh** ada foreign key GORM lintas modul — referensi antar modul hanya berupa ID (string/UUID) biasa, bukan FK database.
- **Inter-module Communication via Client Interface**: satu modul mengakses data modul lain **hanya** lewat interface publik (`*-client`) milik modul tersebut. **Dilarang** melakukan query/JOIN langsung ke tabel modul lain.
- **Independent Data Ownership**: setiap modul bertanggung jawab atas `AutoMigrate`/migration tabelnya sendiri.
- **Module Contract**: setiap modul mengimplementasikan interface modul generik `module.Module` (`Migrate()` dan `RegisterRoutes()`) sehingga wiring di `main.go` seragam untuk semua modul.

---

## 3. Proses Kerja Wajib

```
Task Intake → Reference Check → Planning → Approval Gate →
Implementation → Code Review → Verification → Documentation Sync
```

Dua prinsip inti yang tidak boleh dilanggar:

1. **PLAN FIRST, IMPLEMENT LATER** — tidak ada kode yang boleh ditulis tanpa rencana tertulis yang sudah disetujui owner.
2. **Jika ragu, TANYA. Jangan asumsikan** — setiap ambiguitas harus ditanyakan ke owner, bukan diputuskan sendiri.

---

## 4. Konvensi Layer `usecase`

Pola ini disimpulkan dari analisis usecase existing dan berlaku untuk semua modul.

### 4.1 Struct & Dependency Injection

Setiap usecase wajib punya struct dan constructor dengan pola berikut — field dan urutan parameter constructor **harus konsisten**:

```go
type XUseCase struct {
	DB           *gorm.DB
	Log          *logrus.Logger
	Validate     *validator.Validate
	XRepository  *repository.XRepository
	// tambahkan *-client di sini jika butuh data modul lain
}

func NewXUseCase(
	db *gorm.DB,
	log *logrus.Logger,
	validate *validator.Validate,
	xRepo *repository.XRepository,
) *XUseCase {
	return &XUseCase{
		DB:          db,
		Log:         log,
		Validate:    validate,
		XRepository: xRepo,
	}
}
```

### 4.2 Signature Method

- **Parameter pertama wajib `ctx context.Context`.**
- Parameter kedua (dan seterusnya) mengikuti kebutuhan operasi:
  - Create/Update → `request *model.XxxRequest` (struct request, bukan primitif lepas).
  - Get/Delete → primitif/identifier langsung, contoh `storeId string, id string` (tanpa perlu bikin struct request kalau cuma identifier sederhana).
  - Search/List → `request *model.SearchXxxRequest`.
- **Return value**:
  - Single object → `(*model.XxxResponse, error)`
  - List/paginated → `([]model.XxxResponse, int64, error)` — elemen kedua adalah `total`.
  - Aksi tanpa data balik (Delete) → `error` saja.

### 4.3 Transaksi Database

Setiap method yang menyentuh DB **wajib** membuka transaksi di awal dan rollback via `defer`, lalu commit eksplisit di akhir:

```go
tx := c.DB.WithContext(ctx).Begin()
defer tx.Rollback()

// ...logic...

if err := tx.Commit().Error; err != nil {
	c.Log.Warnf("Failed commit transaction : %+v", err)
	return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal <aksi> data <entity>")
}
```

`defer tx.Rollback()` aman dipanggil walau sudah commit (no-op), jadi selalu ditulis tanpa terkecuali.

### 4.4 Validasi Request

Jika method menerima `request` berupa struct, validasi **wajib** dilakukan di awal, sebelum logic lain, menggunakan `go-playground/validator`:

```go
if err := c.Validate.Struct(request); err != nil {
	c.Log.Warnf("Invalid request body : %+v", err)
	return nil, fiber.NewError(fiber.StatusBadRequest, "Format data request tidak valid")
}
```

Untuk validasi bisnis tambahan yang tidak bisa dihandle tag validator (misal field kondisional, aturan yang bergantung pada field lain, atau field yang wajib diisi hanya di konteks tertentu), lakukan manual check setelahnya. Pola umum: cek kondisi field apa pun yang relevan dengan bisnis proses tersebut, lalu balas `400 Bad Request` dengan pesan yang jelas menyebut field-nya:

```go
if request.<NamaField> == <zero value> {
	return nil, 0, fiber.NewError(fiber.StatusBadRequest, "<NamaField> wajib diisi")
}
```

Field yang divalidasi manual **bisa apa saja** sesuai kebutuhan modul masing-masing — bukan terbatas pada satu field tertentu.

### 4.5 Query Entity dari Repository

Sebelum memanggil repository yang butuh pointer tujuan (`FindById`, dsb), **wajib** siapkan wadah entity kosong terlebih dahulu dengan `new(entity.X)`. Pola ini berlaku untuk **entity apa pun di modul apa pun**, bukan hanya contoh di bawah:

```go
x := new(entity.X)
if err := c.XRepository.FindById(tx, x, id); err != nil {
	c.Log.Warnf("X not found : %+v", err)
	return nil, fiber.NewError(fiber.StatusNotFound, "<Nama entity> tidak ditemukan")
}
```

Alasan wajib bikin wadah (`new(entity.X)`) dulu: repository butuh pointer tujuan yang sudah teralokasi untuk di-scan hasil query GORM-nya (`db.Where(...).First(dest)`), jadi variabelnya harus ada sebelum dipanggil, bukan hasil balikan dari method.

### 4.6 Ownership / Otorisasi Data (Multi-tenant Guard)

Setiap kali entity yang diambil punya field kepemilikan/tenant (bisa `StoreID`, `UserID`, `OwnerID`, atau identifier lain tergantung konteks modul), **wajib** dicek kecocokan dengan konteks pemanggil sebelum lanjut. Jika tidak cocok → `403 Forbidden`. Pola ini generik, berlaku untuk field kepemilikan apa pun, tidak terbatas pada satu nama field tertentu:

```go
if x.<FieldKepemilikan> != <nilai dari konteks pemanggil> {
	c.Log.Warnf("Forbidden: <FieldKepemilikan> mismatch. Expected %s, got %s", x.<FieldKepemilikan>, <nilai dari konteks pemanggil>)
	return nil, fiber.NewError(fiber.StatusForbidden, "Akses ditolak")
}
```

Guard ini wajib diterapkan di **setiap operasi** (Get, Update, Delete, dll) yang mengambil satu entity berdasarkan ID, selama entity tersebut punya konsep kepemilikan/tenant — bukan hanya pada modul yang kebetulan punya field `StoreID`.

### 4.7 Error Handling (Pola Wajib)

Semua error dari layer bawah (repository, validator, commit, dll) mengikuti pola tiga langkah berikut, **tanpa terkecuali**:

```go
if err != nil {
	c.Log.Warnf("<pesan teknis bahasa Inggris> : %+v", err)
	return <zero values>, fiber.NewError(<http status>, "<pesan user-facing bahasa Indonesia>")
}
```

Aturan mapping status:

| Kondisi                               | HTTP Status | Contoh pesan Log (EN)            | Contoh pesan user (ID)                         |
| ------------------------------------- | ----------- | -------------------------------- | ---------------------------------------------- |
| Validasi request gagal                | 400         | `Invalid request body`           | `Format data request tidak valid`              |
| Validasi bisnis manual gagal          | 400         | -                                | pesan spesifik, mis. `<NamaField> wajib diisi` |
| Data tidak ditemukan                  | 404         | `<Entity> not found`             | `<Nama entity> tidak ditemukan`                |
| Kepemilikan/tenant tidak cocok        | 403         | `Forbidden: <field> mismatch`    | `Akses ditolak`                                |
| Gagal generate ID / proses internal   | 500         | `Failed to generate <entity> id` | `Gagal membuat ID <nama entity>`               |
| Gagal query/insert/update/delete repo | 500         | `Failed <aksi> <entity>`         | `Gagal <aksi> data <nama entity>`              |
| Gagal commit transaksi                | 500         | `Failed commit transaction`      | `Gagal <aksi> data <nama entity>`              |

`<Entity>`/`<entity>`/`<nama entity>` diisi sesuai domain modul yang sedang dikerjakan (mis. `Report`, `Category`, `Confirmation`, dst) — bukan literal satu nama tertentu.

Catatan:

- Log **selalu** dalam bahasa Inggris teknis dan **selalu** pakai `%+v` untuk error agar stack/context ikut tercetak.
- Pesan ke user **selalu** dalam Bahasa Indonesia, singkat, dan tidak membocorkan detail teknis/internal.
- **Tidak boleh** return raw `err` dari layer bawah ke controller — selalu dibungkus `fiber.NewError`.

### 4.8 Response ke Layer Atas

Usecase **tidak pernah** mengembalikan entity mentah ke controller. Selalu dikonversi lewat `converter`:

```go
return converter.XToResponse(x), nil
```

Untuk list, konversi per item lalu kembalikan slice + total:

```go
responses := make([]model.XResponse, len(items))
for i, item := range items {
	responses[i] = *converter.XToResponse(&item)
}
return responses, total, nil
```

---

## 5. Konvensi Layer `controller`

Pola ini disimpulkan dari analisis controller existing dan berlaku untuk semua modul.

### 5.1 Struct & Constructor

```go
type XController struct {
	Log     *logrus.Logger
	UseCase *usecase.XUseCase
}

func NewXController(useCase *usecase.XUseCase, logger *logrus.Logger) *XController {
	return &XController{
		Log:     logger,
		UseCase: useCase,
	}
}
```

### 5.2 Signature Handler

Semua handler **wajib** bertipe `func(ctx *fiber.Ctx) error`.

### 5.3 Parsing Input

- **Body** (POST/PUT) → wajib pakai `ctx.BodyParser(request)`, error ditangani sebelum apa pun:

```go
request := new(model.CreateXRequest)
if err := ctx.BodyParser(request); err != nil {
	c.Log.Warnf("Failed to parse request body : %+v", err)
	return fiber.NewError(fiber.StatusBadRequest, "Format data request tidak valid")
}
```

- **Path param** → `ctx.Params("id")`.
- **Query param wajib** → ambil dengan `ctx.Query("key")`, lalu validasi manual kalau kosong. Nama key/field menyesuaikan kebutuhan modul (bisa apa saja, bukan cuma satu nama tertentu):

```go
tenantId := ctx.Query("<key>")
if tenantId == "" {
	return fiber.NewError(fiber.StatusBadRequest, "<NamaField> tidak boleh kosong")
}
```

- **Query param opsional dengan default** → pakai default value langsung di pemanggilan, contoh `ctx.QueryInt("page", 1)`.

### 5.4 Memanggil Usecase

Selalu teruskan `ctx.UserContext()` (bukan `ctx` Fiber) sebagai `context.Context`:

```go
resp, err := c.UseCase.Create(ctx.UserContext(), request)
if err != nil {
	c.Log.Warnf("Failed to create <entity> : %+v", err)
	return err
}
```

Error dari usecase (sudah berupa `*fiber.Error`) **langsung** di-`return`, tidak dibungkus ulang.

### 5.5 Response Sukses

**Wajib** menggunakan `ctx.JSON` dengan `response.WebResponse[T]`, generic type eksplisit sesuai tipe data:

```go
return ctx.JSON(response.WebResponse[*model.XResponse]{
	Data:    resp,
	Message: "Berhasil menambahkan <nama entity>",
	Success: true,
})
```

Untuk endpoint tanpa data balik (Delete): `Data: nil` dengan `response.WebResponse[any]`.

Untuk endpoint list: sertakan `Paging`, dihitung dari `total`:

```go
paging := &response.PageMetadata{
	Page:      request.Page,
	Size:      request.Size,
	TotalItem: total,
	TotalPage: int64(math.Ceil(float64(total) / float64(request.Size))),
}

return ctx.JSON(response.WebResponse[[]model.XResponse]{
	Data:    responses,
	Paging:  paging,
	Message: "Berhasil menampilkan daftar <nama entity>",
	Success: true,
})
```

### 5.6 Dokumentasi Swagger

Setiap handler **wajib** didahului komentar godoc lengkap (`@Summary`, `@Description`, `@Tags`, `@Accept`, `@Produce`, `@Security`, `@Param`, `@Success`, `@Failure`, `@Router`), mengikuti format standar godoc/Swagger — konsisten di semua controller, apa pun modulnya.

---

## 6. Format Response API (Standar)

Seluruh endpoint backend mengembalikan response dengan format seragam melalui generic `WebResponse[T]` (di `internal/shared/response`):

```go
type WebResponse[T any] struct {
	Data    T             `json:"data"`
	Message string        `json:"message,omitempty"`
	Success bool          `json:"success,omitempty"`
	Paging  *PageMetadata `json:"paging,omitempty"`
}
```

## 7. Generic Repository

```go
// internal/shared/repository/repository.go
type Repository[T any] struct{}

// Methods: Create, Update, Delete, CountById, Count, FindById
```

Repository per-modul (mis. `ReportRepository`) meng-embed atau menggunakan generic `Repository[T]` ini untuk operasi CRUD dasar, dan menambahkan method spesifik modul (mis. `Search`) di atasnya.

---

## 8. Konvensi Lain

- **Validasi Input**: `go-playground/validator` via struct tags pada `model.*Request`.
- **Timestamp**: gunakan tipe `*time.Time` pada entity (bukan `int64` milli-epoch).
- **ID Generation**: gunakan util khusus per domain, contoh `utils.GenerateProductId()`, jangan generate ID manual di tempat lain.
- **Bahasa pesan**: log = English teknis (`c.Log.Warnf`, format `"<pesan> : %+v"`), pesan ke user = Bahasa Indonesia lewat `fiber.NewError`.
- **Tidak ada logic HTTP di usecase**: usecase tidak boleh tahu soal `fiber.Ctx`; ia hanya tahu `context.Context` dan tipe domain (`model`, `entity`).
- **Tidak ada akses DB langsung di controller**: controller hanya boleh memanggil `UseCase`, tidak pernah `DB`/`Repository` langsung.

---

## 9. Referensi

- `docs/fixora-prd` — Product Requirement Document lengkap, rujukan utama untuk requirement bisnis sebelum implementasi. Wajib dibaca ulang sebelum mengerjakan modul baru.
- **Out of Scope MVP (jangan dikerjakan tanpa approval owner)**: multi-agent verification kompleks (>2 agent/debate-consensus), RAG cross-reference anggaran pemerintah, predictive risk scoring, satellite/aerial imagery analysis, agentic auto-generate surat pengaduan, social media listening, Google Maps review mining, IoT/sensor integration, traffic anomaly detection, notifikasi push/email, aplikasi mobile native, multi-bahasa, integrasi pembayaran/donasi. Daftar lengkap ada di PRD Bab 7.
- **Success metrics MVP** (ringkas, lihat PRD Bab 6 untuk detail lengkap): jumlah entri masalah aktif di peta, unique visitor, jumlah laporan manual, jumlah konfirmasi "masih begini", akurasi klasifikasi CV, uptime news crawler.

---

## 10. Contoh Peta Modul Domain (Non-Eksklusif)

Contoh `Report`/`ReportUseCase`/`ReportController` di dokumen ini **murni ilustrasi pola layer & konvensi kode** (bab 4–5), bukan domain asli Fixora. Modul domain nyata di Fixora mengikuti kebutuhan fitur di PRD, misalnya (nama indikatif, bisa berubah sesuai desain final):

- **Report / Issue** — laporan masalah infrastruktur (manual dari warga maupun hasil ekstraksi AI news crawler), termasuk status (Mangkrak/Dalam Perbaikan/Selesai) dan riwayat konfirmasi.
- **Category** — kategori masalah (jalan, jembatan, sampah, bangunan, drainase).
- **Confirmation** — entri konfirmasi "masih begini" dari pengguna terhadap satu Report.
- **Crawler Source / News Entry** — hasil crawling berita sebelum/associated dengan Report ber-badge "AI-sourced".
- **Verification** — status verifikasi (pending/lolos/ditolak) terhadap Report baru, baik dari classifier maupun review manual.
- **User** — akun pengguna pelapor (jika autentikasi sudah masuk scope MVP sesuai keputusan final).

Setiap modul di atas **tetap wajib** mengikuti aturan arsitektur di Bab 2 (data isolation, komunikasi lewat `*-client`, migration independen) dan konvensi layer di Bab 4–5 — hanya nama entity/field yang berbeda, pola dan struktur kodenya tetap sama.

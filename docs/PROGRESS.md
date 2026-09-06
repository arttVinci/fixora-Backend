# Fixora — Progress & Roadmap

> **Dokumen ini mencatat status pengerjaan backend Fixora dibandingkan codebase aktual.**
> **Status:** In Progress
> **Branch strategy:** setiap aktivitas dikerjakan di branch terpisah (`feature/*`)
> **Source of truth:** codebase di `internal/modules/`, `cmd/web/main.go`, `internal/shared/`.

---

## Ringkasan Status

| Modul | Status | Keterangan |
|-------|--------|-----------|
| Foundation (model & migration) | ✅ Selesai | MySQL 8.0 via GORM `AutoMigrate` |
| Seed wilayah | ✅ Selesai | Seluruh Indonesia (37 provinsi) |
| Seed kategori | ✅ Selesai | 4 kategori |
| Region module | ⚠️ Sebagian | Migration + seeder + resolve client selesai; **REST endpoint kosong** |
| Category module | ✅ Selesai | `GET /api/categories` |
| Report module | ✅ Selesai | Create + map + detail + analyze-photo |
| News crawler | ✅ Selesai | Cron 2 jam + trigger manual |
| CV classifier | ✅ Selesai | Bagian dari `analyze-photo` |
| Duplicate detection | ✅ Selesai | Perceptual hash + radius + soft-merge |
| Multi-agent verification | ✅ Selesai | 3 agent (advocate/skeptic/manager) |
| Konfirmasi "masih begini" | ⚠️ Sebagian | Tabel + repository ada; **endpoint belum ada** |
| Gov data sync | ❌ Belum | Belum ada modul (lihat `SYSTEM-FLOW-GOV-SYNC.md`) |
| RAG cross-reference anggaran | ❌ Belum | Fase 2, belum ada pipeline |

---

## Yang Sudah Selesai

### 1. Foundation — Database Models & Migration

- Entity GORM: `Province`, `City`, `District`, `Village` (region), `Category`, `Report`, `ReportPhoto`, `Reporter`, `ReportConfirmation`, `DuplicateReport` (report), `CrawledArticle` (crawl), `VerificationSession`, `VerificationLog` (verification).
- Total 13 tabel, dimigrasikan per-modul via `module.Migrate()` → `AutoMigrate`.
- **Database: MySQL 8.0** (driver `gorm.io/driver/mysql`), bukan PostgreSQL.
- Wiring di `cmd/web/main.go`: region → report → verification → crawl (berurutan sesuai dependency).

### 2. Seed Data

- **Wilayah:** seeder parse SQL embedded (`database/seeders/regions`) berisi **seluruh Indonesia** (37 provinsi, 514 kabupaten/kota, dst.) — bukan hanya DKI Jakarta. PK memakai kode BPS (string).
- **Kategori:** 4 kategori — `Sampah`, `Jalan Rusak`, `Jembatan Rusak`, `Bangunan Terbengkalai` — masing-masing dengan `search_keywords` (JSON) untuk query RSS crawler.
- Seeder idempotent (`SeedIfEmpty`).

### 3. Report Module (Core)

Endpoint terdaftar di `report/route.go`:

- `GET /api/reports/map` — data peta (bounding box `min_lat/max_lat/min_lng/max_lng` + filter `category_id/status/severity/source_type`; auto-filter `merged_into_id IS NULL`; limit 500).
- `GET /api/reports/:id` — detail report + foto + konfirmasi + `related_reports`.
- `POST /api/reports/analyze-photo` — CV classifier (upload foto → draft AI).
- `POST /api/reports` — create report warga (reverse geocode → resolve village → promote foto → simpan).
- `GET /api/categories` — list kategori.

### 4. News Crawler Module

- Cron tiap 2 jam (`0 */2 * * *`) + langsung jalan saat startup + `POST /api/crawl/trigger` untuk manual.
- Fetch Google News RSS per keyword `search_keywords` × region Jabodetabek.
- LLM Gemini ekstraksi → geocode Nominatim → resolve village → validasi Jabodetabek → auto-create report `ai_news`.
- Dedup by URL; rejected article disimpan untuk audit.
- Detail lengkap: `SYSTEM-FLOW-CRAWLER.md`.

### 5. CV Classifier (US-06)

- Terintegrasi di `POST /api/reports/analyze-photo` (modul report, bukan modul terpisah).
- Vision LLM Gemini (`gemini-3.5-flash-lite`) + guard berlapis (stempel timestamp kamera, relevansi, kategori terdaftar, lokasi terbaca, geocoding).

### 6. Duplicate Detection (US-07)

- Perceptual hash (`goimagehash`) foto primary + radius 100m + kategori sama.
- Soft-merge via `reports.merged_into_id`; audit trail di `duplicate_reports`.

### 7. Multi-Agent Verification (di luar rencana awal)

- Modul `verification` dengan 3 agent: advocate → skeptic → manager (debate + consensus).
- Cron tiap 30 detik; endpoint `POST /api/crawl/verify/trigger/:reportId`, `POST /api/crawl/verify/retry/:sessionId`, `GET /api/crawl/verify/sessions/:reportId`.
- Hanya `user_report` yang diverifikasi via LLM; `ai_news`/`gov_data` auto-approve (`verified`).

---

## Yang Belum Selesai

### A. Region REST API (Step 3 rencana lama)

`region/route.go` **kosong** — belum ada endpoint:

- `GET /api/provinces`
- `GET /api/provinces/:id/cities`
- `GET /api/cities/:id/districts`
- `GET /api/districts/:id/villages`

Client `region-client.ResolveVillageByAddress` sudah ada dan dipakai report+crawler, tapi tidak ada controller/usecase untuk list wilayah.

### B. Konfirmasi "Masih Begini" (US-04)

Tabel `report_confirmations` + repository (`HasConfirmedByIP`, anti-spam 24 jam) + entity + `TotalConfirmations` di detail response sudah ada, tapi **belum ada endpoint/usecase** untuk membuat konfirmasi.

### C. Gov Data Sync (US-08)

Belum diimplementasikan sama sekali. `source_type: gov_data` didukung di enum/validasi tapi tidak ada pipeline penghasilnya. Lihat `SYSTEM-FLOW-GOV-SYNC.md`.

### D. RAG Cross-Reference Anggaran (Fase 2)

Belum ada. Tidak ada tabel `budget_items`, tidak ada pipeline RAG.

---

## Catatan Umum

- **Arsitektur:** Modular Monolith (setiap modul implement `module.Module`: `Migrate()` + `RegisterRoutes()`).
- **API versioning:** prefix `/api` (bukan `/api/v1`) — sesuai `main.go` `app.Group("/api")`.
- **Error handling:** standar `dto.WebResponse[T]` + `dto.ApiErrorResponse` (error handler di `shared/config/fiber.go`).
- **Reverse geocoding:** Nominatim (OpenStreetMap) — dipakai di report & crawler.
- **Foto:** Cloudinary (staging → promote) + cleanup worker untuk orphan staging.
- **LLM:** Gemini (`google_ai_studio`) untuk CV + ekstraksi berita; CommandCode (OpenAI-compatible) untuk verifikasi multi-agent.

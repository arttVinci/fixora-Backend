# Fixora — Progress & Roadmap

> **Dokumen ini mencatat step-by-step pengerjaan backend Fixora.**  
> **Status:** In Progress  
> **Branch strategy:** Setiap step dikerjakan di branch terpisah

---

## Step 1: Foundation — Database Models & Migration

**Branch:** `feat/database-models`

**Tujuan:** Setup semua GORM model sesuai DATABASE-SCHEMA.md dan jalankan auto-migrate.

- [ ] Buat GORM model: `Province`, `City`, `District`, `Village`
- [ ] Buat GORM model: `Category`
- [ ] Buat GORM model: `Report` (village_id NULLABLE)
- [ ] Buat GORM model: `ReportPhoto`
- [ ] Buat GORM model: `CrawledArticle`
- [ ] Pastikan semua relasi (FK, ON DELETE CASCADE) terdefinisi dengan benar
- [ ] Auto-migrate semua model saat aplikasi start
- [ ] Test: jalankan aplikasi, pastikan 8 tabel terbuat di PostgreSQL

**Catatan:** `village_id` di `reports` harus nullable — kalau reverse geocoding gagal matching, report tetap bisa tersimpan.

---

## Step 2: Seed Data — Wilayah Jakarta & Kategori

**Branch:** `feat/seed-data`

**Tujuan:** Isi tabel wilayah (DKI Jakarta) dan kategori awal agar modul lain punya data referensi.

- [ ] Seed wilayah DKI Jakarta dari data wilayah.id (1 provinsi, 6 kota, ~44 kecamatan, ~267 kelurahan)
- [ ] Seed 5 kategori awal:
  - Jalan Rusak
  - Jembatan Rusak
  - Sampah Menumpuk
  - Bangunan Terbengkalai
  - Drainase Tersumbat
- [ ] Buat mekanisme seed yang idempotent (bisa dijalankan berulang tanpa duplikat)
- [ ] Test: query tabel provinces/cities/districts/villages, pastikan data lengkap

---

## Step 3: Region Module — API Wilayah

**Branch:** `feat/region-module`

**Tujuan:** Endpoint API untuk frontend bisa fetch data wilayah (untuk filter dropdown, search, dll).

- [ ] `GET /api/v1/provinces` — list semua provinsi
- [ ] `GET /api/v1/provinces/:id/cities` — list kota by provinsi
- [ ] `GET /api/v1/cities/:id/districts` — list kecamatan by kota
- [ ] `GET /api/v1/districts/:id/villages` — list kelurahan by kecamatan
- [ ] Response format standar (JSON, pagination jika perlu)
- [ ] Test: hit semua endpoint, pastikan data sesuai seed

---

## Step 4: Category Module — API Kategori

**Branch:** `feat/category-module`

**Tujuan:** Endpoint API untuk frontend fetch daftar kategori (untuk filter di peta & form pelaporan).

- [ ] `GET /api/v1/categories` — list semua kategori (id, name, slug, icon, color)
- [ ] Test: hit endpoint, pastikan 5 kategori tampil

---

## Step 5: Report Module — API Laporan (Core)

**Branch:** `feat/report-module`

**Tujuan:** CRUD laporan — ini modul inti Fixora. Frontend bisa submit laporan baru dan menampilkan data di peta.

- [ ] `POST /api/v1/reports` — create report baru (dengan upload foto)
  - Input: foto (wajib), title, description (opsional), latitude, longitude, category_id
  - Backend: reverse geocoding (Nominatim) → resolve village_id
  - Backend: simpan foto ke storage → create record report_photos
  - Response: report yang baru dibuat
- [ ] `GET /api/v1/reports` — list reports (untuk peta)
  - Filter: category_id, status, source_type, village/district/city, severity
  - Response: list reports dengan foto primary, info wilayah
  - Support pagination & bounding box (lat/lng range) untuk peta viewport
- [ ] `GET /api/v1/reports/:id` — detail satu report
  - Response: report lengkap + semua foto + info wilayah (JOIN ke 4 tabel)
- [ ] Integrasi Nominatim reverse geocoding
  - lat/lng → province, city, district, village → match ke tabel wilayah → set village_id
  - Kalau gagal match → village_id NULL, simpan raw address text
- [ ] Upload foto ke local storage / MinIO (configurable)
- [ ] Test: create report via API, lalu GET dan pastikan tampil

---

## Step 6: News Crawler Module — AI Data Collector

**Branch:** `feat/news-crawler`

**Tujuan:** Cron job yang tarik berita infrastruktur dari RSS, LLM extract data, auto-create report. Ini yang bikin platform punya data tanpa nunggu warga lapor.

- [ ] Setup cron job scheduler (berjalan tiap beberapa jam)
- [ ] RSS feed fetcher — tarik artikel dari media (Detik, Kompas, Tempo, dll)
- [ ] Deduplikasi URL — cek `crawled_articles.url` UNIQUE sebelum proses
- [ ] LLM extraction — kirim konten berita ke LLM, minta structured output:
  - `{ location, category, severity, title, description }`
- [ ] Geocoding — convert lokasi teks → lat/lng (Nominatim)
- [ ] Reverse geocoding → match village_id (sama seperti flow report)
- [ ] Auto-create report dengan `source_type = 'ai_news'`
- [ ] Update `crawled_articles.status` (pending → processed/rejected)
- [ ] `GET /api/v1/crawled-articles` — list artikel (untuk monitoring/admin)
- [ ] Test: jalankan crawler manual, pastikan artikel → report terbuat

---

## Step 7: CV Classifier & Severity Scoring

**Branch:** `feat/cv-classifier`

**Tujuan:** Saat warga upload foto, AI otomatis klasifikasi kategori + skor severity.

- [ ] Integrasi multimodal LLM API (vision)
- [ ] Saat `POST /api/v1/reports`:
  - Kirim foto ke LLM → dapat `{ category, severity, confidence }`
  - Kalau confidence tinggi → auto-set category_id & severity
  - Kalau confidence rendah → flag `status = 'pending_verification'` + perlu review
- [ ] Test: upload foto jalan rusak, pastikan auto-classify benar

---

## Step 8: Duplicate Detection

**Branch:** `feat/duplicate-detection`

**Tujuan:** Deteksi & merge laporan duplikat agar peta tidak penuh entri yang sama.

- [ ] Generate perceptual hash saat foto diupload
- [ ] Saat report baru masuk, cek:
  - Perceptual hash similarity dengan foto existing
  - Radius GPS (misal < 100m) + kategori sama + rentang waktu dekat
- [ ] Kalau terdeteksi duplikat → set `merged_into_id` ke report induk (soft-merge)
- [ ] Report yang di-merge tidak tampil di `GET /reports` (filter `WHERE merged_into_id IS NULL`)
- [ ] Test: submit 2 report di lokasi sama dengan foto mirip, pastikan ter-merge

---

## Catatan Umum

- **Arsitektur:** Clean Architecture / Modular Monolith (sesuai existing codebase)
- **Setiap module** implement interface `Module` (`RegisterRoutes` + `Migrate`)
- **API versioning:** `/api/v1/...`
- **Error handling:** Standard response format dari `shared/response`
- **Reverse geocoding:** Nominatim (OpenStreetMap, gratis) — dipakai di Step 5 & Step 6

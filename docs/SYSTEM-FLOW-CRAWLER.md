# System Flow: AI News Crawler

> Dokumen ini menjelaskan alur kerja AI News Crawler — proses otomatis yang berjalan di background untuk mengumpulkan data masalah infrastruktur dari media berita, tanpa bergantung pada laporan warga.  
> Referensi: **US-05 (AI News Crawler)** di PRD.

---

## Gambaran Umum

News Crawler adalah **cron job** yang berjalan tiap 2 jam secara otonom. Tugasnya: tarik berita infrastruktur dari RSS feed berdasarkan keyword dari tabel `categories` (bukan hardcode), ekstrak informasi terstruktur (kategori slug, lokasi, severity) menggunakan LLM, geocode lokasi via Nominatim, resolve village via `region-client`, lalu otomatis membuat entry di tabel `reports` dengan badge sumber `ai_news`. Artikel yang gagal diproses tetap disimpan sebagai `rejected` agar URL-nya tidak dicrawl ulang.

---

## Alur Lengkap

### Fase 1 — Trigger & Iterasi Keyword dari DB

| Step | Proses | Detail |
|------|--------|--------|
| 1 | Cron job aktif | Scheduler men-trigger crawl tiap 2 jam |
| 2 | Ambil categories dari DB | Keyword diambil dari tabel `categories` lewat `report-client.GetAllCategories()` |
| 3 | Fetch RSS per keyword | Untuk setiap category name, fetch hasil dari Google News RSS |
| 4 | Dedup in-memory | Artikel dari berbagai keyword digabung, dedup by URL agar tidak diproses ganda |

---

### Fase 2 — Deduplikasi Artikel (DB)

Untuk setiap artikel unik dari RSS:

| Step | Proses | Detail |
|------|--------|--------|
| 5 | Cek duplikat URL di DB | Query `crawled_articles` — apakah URL artikel ini sudah pernah diproses? |

**Percabangan:**
- **URL sudah ada di database** → Skip, lanjut ke artikel berikutnya
- **URL belum ada** → Lanjut ke Fase 3

> Kolom `url` di tabel `crawled_articles` di-set UNIQUE.

---

### Fase 3 — Ekstraksi LLM

| Step | Proses | Detail |
|------|--------|--------|
| 6 | Kirim ke LLM | Judul dan isi berita dikirim ke LLM (Gemini) dengan prompt structured output (JSON mode) |
| 7 | LLM ekstrak | LLM mengembalikan: `category` (slug), `location` (teks), `severity`, `is_relevant` (boolean) |

**Percabangan setelah ekstraksi:**
- **LLM gagal ekstrak** → Simpan sebagai `rejected` dengan reason `llm_extraction_failed`
- **`is_relevant: false`** → Simpan sebagai `rejected` dengan reason `not_relevant`
- **LLM berhasil dan relevan** → Lanjut ke Fase 4

> Hasil extraction bersifat transient — hanya dipakai saat pipeline, **tidak disimpan** di `crawled_articles`.

---

### Fase 4 — Resolve Category

| Step | Proses | Detail |
|------|--------|--------|
| 8 | Resolve category slug | Slug dari LLM (misal `jalan-rusak`) di-resolve ke UUID via `report-client.GetCategoryBySlug()` |

**Percabangan:**
- **Category tidak ditemukan** → Simpan sebagai `rejected` dengan reason `category_not_found`
- **Category ditemukan** → `category_id` tersedia → Lanjut ke Fase 5

---

### Fase 5 — Geocoding & Resolve Wilayah

| Step | Proses | Detail |
|------|--------|--------|
| 9 | Forward geocoding | Teks lokasi hasil LLM dikonversi menjadi koordinat `lat/lng` + `display_name` via Nominatim |
| 10 | Resolve village | Dari `display_name` Nominatim, resolve ke `village_id` via `region-client.ResolveVillageByName()` |

**Percabangan:**
- **Geocoding gagal** → Simpan sebagai `rejected` dengan reason `geocoding_failed`
- **Village tidak ditemukan** → Simpan sebagai `rejected` dengan reason `village_not_resolved`
- **Semua berhasil** → Koordinat, address, dan `village_id` tersedia → Lanjut ke Fase 6

> Geocode dan village resolution adalah dua step terpisah. Nominatim hanya return lat/lon/address (generic). Village resolution adalah domain region module.

---

### Fase 6 — Buat Report Otomatis

| Step | Proses | Detail |
|------|--------|--------|
| 11 | Insert crawled_article | Simpan artikel ke `crawled_articles` dengan `status: processed` |
| 12 | Insert report | Buat entry baru di tabel `reports` via `report-client.CreateReport()` |
| 13 | Link article ke report | Update `crawled_articles.report_id` dengan ID report yang baru dibuat |

**Data yang diisi saat insert report:**

| Field | Nilai |
|-------|-------|
| `id` | Special ID format `RPT-<category_slug>-<YYYYMMDD>-<random>` |
| `category_id` | Hasil resolve dari slug LLM |
| `village_id` | Hasil resolve dari region-client |
| `title` | Judul berita |
| `description` | Isi/ringkasan berita |
| `latitude` / `longitude` | Hasil geocoding Nominatim |
| `address` | Display name dari Nominatim |
| `severity` | Hasil ekstraksi LLM (`ringan`/`sedang`/`parah`) |
| `status` | `pending_verification` |
| `source_type` | `ai_news` |
| `reporter_id` | `NULL` (tidak ada pelapor manusia) |
| `first_reported_at` | Tanggal publikasi berita (bukan waktu crawl) |

---

### Fase 7 — Rejected Article

Artikel yang gagal di fase mana pun tetap disimpan ke `crawled_articles` dengan:
- `status: rejected`
- `reject_reason`: salah satu dari `llm_extraction_failed`, `not_relevant`, `category_not_found`, `geocoding_failed`, `village_not_resolved`

Tujuan: URL tercatat sehingga tidak dicrawl ulang, dan tersedia untuk audit/debugging.

---

## Concurrency & Timeout

- Artikel diproses **concurrent** dengan semaphore (max 5 goroutine).
- Setiap goroutine punya **timeout context** (2 menit per artikel).
- Keseluruhan crawl cycle punya timeout 30 menit.

---

## Ringkasan Mapping ke Backend

| Fase | Proses Backend | Dependency |
|------|----------------|------------|
| Fase 1 (Keyword & RSS) | `report-client.GetAllCategories()` + RSS client (crawl infra) | `report-client`, `crawl/infra` |
| Fase 2 (Dedup URL) | `CrawledRepository.FindByURL()` | `crawl/repository` |
| Fase 3 (LLM) | LLM client (crawl infra) | `crawl/infra` |
| Fase 4 (Category) | `report-client.GetCategoryBySlug()` | `report-client` |
| Fase 5 (Geocoding + Village) | Nominatim client (shared) + `region-client.ResolveVillageByName()` | `shared/client`, `region-client` |
| Fase 6 (Save) | `CrawlerUseCase.SaveCrawledReport()` → `report-client.CreateReport()` | `crawl/usecase`, `report-client` |
| Fase 7 (Rejected) | `CrawlerUseCase.SaveRejectedArticle()` | `crawl/usecase` |

---

## Catatan Desain

- Crawler ini berjalan **sepenuhnya di background** — tidak ada interaksi user, tidak ada REST endpoint.
- Semua report hasil crawler masuk dengan `source_type: ai_news` dan ditampilkan di peta dengan badge *"Sumber: Media"*.
- **Keyword dari DB**, bukan hardcode — sehingga menambah category otomatis menambah keyword crawler.
- **Rejected article tetap tersimpan** untuk audit trail dan dedup, tapi tidak menghasilkan report.
- **Data extraction bersifat transient** — `crawled_articles` lean, tidak menyimpan `extracted_*` fields.
- **RSS dan LLM client** ada di `crawl/infra`, bukan di `shared/client`, karena domain-specific.
- **Nominatim tetap di `shared/client`** karena generic geocoding, bisa dipakai module lain.
- Flow ini selaras dengan **US-05 (AI News Crawler)** di PRD.

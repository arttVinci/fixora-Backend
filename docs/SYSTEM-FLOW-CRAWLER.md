# System Flow: AI News Crawler

> Dokumen ini menjelaskan alur kerja AI News Crawler — proses otomatis yang berjalan di background untuk mengumpulkan data masalah infrastruktur dari media berita, tanpa bergantung pada laporan warga.
> Referensi: **US-05 (AI News Crawler)** di PRD.
> Source of truth: `internal/modules/crawl/src/worker/crawler_worker.go`, `internal/modules/crawl/src/usecase/crawler_usecase.go`, `internal/modules/crawl/src/infra/`.

---

## Gambaran Umum

News Crawler adalah **cron job** yang berjalan tiap 2 jam secara otonom. Tugasnya: tarik berita infrastruktur dari Google News RSS berdasarkan keyword dari kolom `categories.search_keywords` (bukan hardcode), ekstrak informasi terstruktur (title, kategori slug, lokasi, severity) menggunakan LLM Gemini, geocode lokasi via Nominatim, resolve village via `region-client`, validasi scope wilayah Jabodetabek, lalu otomatis membuat entry di tabel `reports` dengan `source_type: ai_news`. Artikel yang gagal diproses (bukan kegagalan transien LLM) disimpan sebagai `rejected` agar URL-nya tidak dicrawl ulang.

---

## Alur Lengkap

### Fase 1 — Trigger & Iterasi Keyword dari DB

| Step | Proses | Detail |
|------|--------|--------|
| 1 | Cron job aktif | Scheduler men-trigger crawl tiap 2 jam (`0 */2 * * *`), plus langsung dijalankan sekali saat startup |
| 2 | Ambil categories dari DB | Kategori diambil dari tabel `categories` lewat `report-client.GetAllCategories()` |
| 3 | Ambil `search_keywords` per kategori | Setiap kategori punya array `search_keywords` (JSON) |
| 4 | Query RSS per keyword × region | Untuk setiap keyword, query Google News RSS per kota di scope Jabodetabek: `{keyword} {region}` |
| 5 | Dedup in-memory | Artikel dari berbagai keyword digabung, dedup by URL agar tidak diproses ganda |

---

### Fase 2 — Deduplikasi Artikel (DB)

Untuk setiap artikel unik dari RSS:

| Step | Proses | Detail |
|------|--------|--------|
| 6 | Cek duplikat URL di DB | Query `crawled_articles` — apakah URL artikel ini sudah pernah diproses? |

**Percabangan:**
- **URL sudah ada di database** → Skip, lanjut ke artikel berikutnya
- **URL belum ada** → Lanjut ke Fase 3

> Kolom `url` di tabel `crawled_articles` di-set UNIQUE. Baris dengan `reject_reason = 'llm_extraction_failed'` dianggap belum selesai (transien) dan **tetap bisa di-retry** — lihat `CrawledRepository.FindByURL`.

---

### Fase 3 — Ekstraksi LLM

| Step | Proses | Detail |
|------|--------|--------|
| 7 | Kirim ke LLM | Judul dan isi berita dikirim ke LLM Gemini (`gemini-3.5-flash-lite`) dengan structured output (JSON mode) |
| 8 | LLM ekstrak | LLM mengembalikan: `title`, `category` (slug), `location` (teks), `severity`, `is_relevant` (boolean) |

**Percabangan setelah ekstraksi:**
- **LLM gagal ekstrak** → **TIDAK disimpan sebagai rejected**. Kegagalan LLM (rate limit, deadline, network) bersifat transien — dibiarkan agar URL di-retry pada run berikutnya.
- **`is_relevant: false`** → Simpan sebagai `rejected` dengan reason `not_relevant`
- **LLM berhasil dan relevan** → Lanjut ke Fase 4

> Hasil extraction bersifat transient — hanya dipakai saat pipeline, **tidak disimpan** di `crawled_articles` (tidak ada kolom `extracted_*`).

---

### Fase 4 — Resolve Category

| Step | Proses | Detail |
|------|--------|--------|
| 9 | Resolve category slug | Slug dari LLM (misal `jalan-rusak`) di-resolve ke UUID via `report-client.GetCategoryBySlug()` |

**Percabangan:**
- **Category tidak ditemukan** → Simpan sebagai `rejected` dengan reason `category_not_found`
- **Category ditemukan** → `category_id` tersedia → Lanjut ke Fase 5

---

### Fase 5 — Geocoding & Resolve Wilayah

| Step | Proses | Detail |
|------|--------|--------|
| 10 | Forward geocoding | Teks lokasi hasil LLM dikonversi menjadi koordinat `lat/lng` via Nominatim |
| 11 | Reverse geocoding | Dari `lat/lng`, ambil `display_name` + komponen wilayah (village/district/city/province) via Nominatim |
| 12 | Resolve village | Dari komponen wilayah, resolve ke `village_id` via `region-client.ResolveVillageByAddress()` |
| 13 | Validasi scope wilayah | Cek kode BPS kota (`village.CityCode`) dengan `isInJabodetabek()` |

**Percabangan:**
- **Forward geocoding gagal** → Simpan sebagai `rejected` dengan reason `geocoding_failed`
- **Reverse geocoding gagal** → Simpan sebagai `rejected` dengan reason `reverse_geocoding_failed`
- **Village tidak ditemukan** → Simpan sebagai `rejected` dengan reason `village_not_resolved`
- **Di luar Jabodetabek** → Simpan sebagai `rejected` dengan reason `outside_jabodetabek`
- **Semua berhasil** → Koordinat, address, dan `village_id` tersedia → Lanjut ke Fase 6

> Geocode dan village resolution adalah step terpisah. Nominatim hanya return lat/lon/address (generic). Village resolution adalah domain region module.
> Scope wilayah: **Jabodetabek** (DKI Jakarta + Bogor + Depok + Tangerang + Bekasi), berdasarkan kode BPS kota — bukan hanya DKI Jakarta.

---

### Fase 6 — Buat Report Otomatis

| Step | Proses | Detail |
|------|--------|--------|
| 14 | Insert crawled_article | Simpan artikel ke `crawled_articles` dengan `status: success` (transient) |
| 15 | Insert report | Buat entry baru di tabel `reports` via `report-client.CreateReport()` |
| 16 | Update article | Update `crawled_articles` ke `status: processed` + `report_id` |

**Data yang diisi saat insert report:**

| Field | Nilai |
|-------|-------|
| `id` | `RPT-<category_slug>-<YYYYMMDD>-<random>` |
| `category_id` | Hasil resolve dari slug LLM |
| `village_id` | Hasil resolve dari region-client |
| `title` | `clean_title` (judul hasil LLM, fallback ke judul berita) |
| `description` | Isi/ringkasan berita |
| `latitude` / `longitude` | Hasil geocoding Nominatim |
| `address` | Display name dari Nominatim |
| `severity` | Hasil ekstraksi LLM (`ringan`/`sedang`/`parah`) |
| `status` | `verified` |
| `source_type` | `ai_news` |
| `source_url` | URL artikel asli |
| `confidence_score` | `0.8` |
| `reporter_id` | `NULL` (tidak ada pelapor manusia) |
| `first_reported_at` | Tanggal publikasi berita (bukan waktu crawl) |

---

### Fase 7 — Rejected Article

Artikel yang gagal di fase mana pun (kecuali kegagalan transien LLM) disimpan ke `crawled_articles` dengan:
- `status: rejected`
- `reject_reason`: salah satu dari `not_relevant`, `category_not_found`, `geocoding_failed`, `reverse_geocoding_failed`, `village_not_resolved`, `outside_jabodetabek`

Tujuan: URL tercatat sehingga tidak dicrawl ulang, dan tersedia untuk audit/debugging.

---

## Concurrency & Timeout

- Artikel diproses **secara sequential** (satu per satu), bukan concurrent — disesuaikan dengan rate limit Gemini free tier (±15 RPM + daily quota).
- Setiap artikel punya **timeout context** 5 menit.
- Keseluruhan crawl cycle punya timeout **3 jam** (karena pemrosesan sekuensial ratusan artikel × 15 detik rate limiter).
- Versi concurrent (`processArticlesConcurrent`, semaphore + goroutine) tersedia sebagai komentar di `crawler_worker.go` untuk dipakai setelah upgrade ke paid tier.

---

## Ringkasan Mapping ke Backend

| Fase | Proses Backend | Dependency |
|------|----------------|------------|
| Fase 1 (Keyword & RSS) | `report-client.GetAllCategories()` + RSS client (crawl infra) + `utils.ArticleRssFilter` | `report-client`, `crawl/infra`, `crawl/utils` |
| Fase 2 (Dedup URL) | `CrawledRepository.FindByURL()` | `crawl/repository` |
| Fase 3 (LLM) | LLM client Gemini (crawl infra) | `crawl/infra` |
| Fase 4 (Category) | `report-client.GetCategoryBySlug()` | `report-client` |
| Fase 5 (Geocoding + Village) | Nominatim client (shared) + `region-client.ResolveVillageByName()` + `isInJabodetabek()` | `shared/client`, `region-client`, `crawl/worker` |
| Fase 6 (Save) | `CrawlerUseCase.SaveCrawledReport()` → `report-client.CreateReport()` | `crawl/usecase`, `report-client` |
| Fase 7 (Rejected) | `CrawlerUseCase.SaveRejectedArticle()` | `crawl/usecase` |

---

## Catatan Desain

- Crawler ini berjalan **sepenuhnya di background** — tidak ada interaksi user. Endpoint `POST /api/crawl/trigger` tersedia untuk trigger manual.
- Semua report hasil crawler masuk dengan `source_type: ai_news` dan status `verified` (langsung tayang, tanpa melewati multi-agent verification — verifikasi hanya untuk `user_report`).
- **Keyword dari DB** (`search_keywords`), bukan hardcode — menambah/ubah keyword kategori otomatis mengubah query crawler.
- **Rejected article tetap tersimpan** untuk audit trail dan dedup, tapi tidak menghasilkan report.
- **Data extraction bersifat transient** — `crawled_articles` lean, tidak menyimpan `extracted_*` fields.
- **RSS dan LLM client** ada di `crawl/infra`, bukan di `shared/client`, karena domain-specific.
- **Nominatim tetap di `shared/client`** karena generic geocoding, bisa dipakai module lain.
- Flow ini selaras dengan **US-05 (AI News Crawler)** di PRD.

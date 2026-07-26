# System Flow: AI News Crawler

> Dokumen ini menjelaskan alur kerja AI News Crawler — proses otomatis yang berjalan di background untuk mengumpulkan data masalah infrastruktur dari media berita, tanpa bergantung pada laporan warga.  
> Referensi: **US-05 (AI News Crawler)** di PRD.

---

## Gambaran Umum

News Crawler adalah **cron job** yang berjalan tiap beberapa jam secara otonom. Tugasnya: tarik berita infrastruktur dari RSS feed, ekstrak informasi terstruktur (kategori, lokasi, severity) menggunakan LLM, lalu otomatis membuat entry di tabel `reports` dengan badge sumber `ai_news`. Tujuannya mengatasi *cold-start problem* — platform punya data sejak hari pertama tanpa menunggu warga melapor.

---

## Alur Lengkap

### Fase 1 — Trigger & Iterasi Keyword

| Step | Proses | Detail |
|------|--------|--------|
| 1 | Cron job aktif | Scheduler men-trigger crawl tiap beberapa jam (interval configurable) |
| 2 | Loop keyword | Sistem iterasi semua keyword pencarian: *"jalan rusak"*, *"jembatan rusak"*, *"sampah menumpuk"*, *"bangunan terbengkalai"*, *"drainase tersumbat"*, dll — sesuai daftar `categories` |
| 3 | Fetch RSS | Untuk setiap keyword, fetch hasil dari Google News RSS |

---

### Fase 2 — Deduplikasi Artikel

Untuk setiap artikel berita yang ditemukan dari RSS:

| Step | Proses | Detail |
|------|--------|--------|
| 4 | Cek duplikat URL | Query `crawled_articles` — apakah URL artikel ini sudah pernah diproses? |

**Percabangan:**
- **URL sudah ada di database** → Skip, lanjut ke artikel berikutnya
- **URL belum ada** → Lanjut ke Fase 3

> Kolom `url` di tabel `crawled_articles` di-set UNIQUE, jadi pengecekan ini mencegah crawl ulang artikel yang sama.

---

### Fase 3 — Simpan & Ekstraksi LLM

| Step | Proses | Detail |
|------|--------|--------|
| 5 | Simpan artikel | Insert ke `crawled_articles` dengan `status: pending` |
| 6 | Kirim ke LLM | Judul dan/atau isi berita dikirim ke LLM dengan prompt structured output (JSON mode) |
| 7 | LLM ekstrak | LLM mengembalikan: kategori masalah, lokasi (teks), tingkat keparahan (`ringan`/`sedang`/`parah`) |

**Percabangan setelah ekstraksi:**
- **LLM gagal ekstrak / lokasi terlalu umum / berita tidak relevan** → Tandai `crawled_articles.status: rejected` → lanjut ke artikel berikutnya
- **LLM berhasil ekstrak dengan lokasi yang cukup spesifik** → Lanjut ke Fase 4

> Hasil ekstraksi LLM disimpan ke field `extracted_location`, `extracted_category_id`, `extracted_severity` di `crawled_articles` — terlepas dari berhasil atau tidaknya proses selanjutnya, untuk keperluan audit trail dan re-processing.

---

### Fase 4 — Geocoding & Resolve Wilayah

| Step | Proses | Detail |
|------|--------|--------|
| 8 | Forward geocoding | Teks lokasi hasil LLM (misal: *"Jl. Gatot Subroto, Jakarta Selatan"*) dikonversi menjadi koordinat lat/lng via Nominatim |
| 9 | Reverse geocoding | Dari koordinat lat/lng, resolve ke hierarki wilayah (`villages` → `districts` → `cities` → `provinces`) untuk mendapatkan `village_id` |

**Percabangan:**
- **Geocoding gagal** (lokasi terlalu ambigu, tidak ditemukan) → Tandai `crawled_articles.status: rejected`
- **Geocoding berhasil** → Koordinat dan `village_id` tersedia → Lanjut ke Fase 5

> Proses reverse geocoding ini identik dengan yang dijalankan di flow pelaporan manual (User Flow Fase 4).

---

### Fase 5 — Buat Report Otomatis

| Step | Proses | Detail |
|------|--------|--------|
| 10 | Insert report | Buat entry baru di tabel `reports` dengan data hasil ekstraksi |

**Data yang diisi saat insert:**

| Field | Nilai |
|-------|-------|
| `category_id` | Hasil matching dari `extracted_category_id` |
| `village_id` | Hasil reverse geocoding (Fase 4) |
| `title` | Judul yang di-generate LLM dari konteks berita |
| `description` | Ringkasan berita (opsional) |
| `latitude` / `longitude` | Hasil geocoding |
| `address` | Alamat dari reverse geocoding |
| `severity` | Hasil ekstraksi LLM (`ringan`/`sedang`/`parah`) |
| `status` | `pending_verification` |
| `source_type` | `ai_news` |
| `reporter_id` | `NULL` (tidak ada pelapor manusia) |
| `first_reported_at` | Tanggal publikasi berita (bukan waktu crawl) |

---

### Fase 6 — Update Status & Lanjut ke Pipeline Gabungan

| Step | Proses | Detail |
|------|--------|--------|
| 11 | Update crawled_articles | Tandai `status: processed` dan isi `report_id` dengan ID report yang baru dibuat |
| 12 | Trigger pipeline gabungan | Report baru ini masuk ke pipeline yang sama dengan laporan manual: |

**Pipeline Gabungan (berlaku untuk semua report, baik `user_report` maupun `ai_news`):**
- **Duplicate Detection (US-07):** Cek radius GPS + kategori + rentang waktu terhadap report existing. Jika terdeteksi duplikat → soft-merge (`merged_into_id`), catat di `merge_logs`
- **RAG Cross-Reference Anggaran (Async):** Trigger `CrossReferenceBudget(reportID)` untuk mencocokkan dengan data anggaran pemerintah → hasil disimpan ke `reports.budget_info`

Setelah semua artikel dalam satu keyword selesai diproses, lanjut ke keyword berikutnya. Jika semua keyword sudah habis, satu putaran crawl selesai dan menunggu jadwal berikutnya.

---

## Ringkasan Mapping ke Backend

| Fase | Proses Backend | Tabel Terkait |
|------|----------------|---------------|
| Fase 1 (Fetch RSS) | Cron scheduler + HTTP client | — |
| Fase 2 (Dedup URL) | Query `WHERE url = ?` | `crawled_articles` |
| Fase 3 (Simpan & LLM) | Insert + LLM API call | `crawled_articles`, `categories` |
| Fase 4 (Geocoding) | Nominatim forward + reverse | `villages`, `districts`, `cities`, `provinces` |
| Fase 5 (Buat Report) | Insert report | `reports` |
| Fase 6 (Update & Pipeline) | Update status + trigger async | `crawled_articles`, `reports`, `merge_logs`, `budget_items` |

---

## Catatan Desain

- Crawler ini berjalan **sepenuhnya di background** — tidak ada interaksi user.
- Semua report hasil crawler masuk dengan `source_type: ai_news` dan ditampilkan di peta dengan badge *"Sumber: Media"* untuk transparansi asal data.
- Artikel yang `rejected` tetap tersimpan di `crawled_articles` untuk audit trail, tapi tidak menghasilkan report.
- Pipeline gabungan (dedup + RAG) berlaku universal untuk semua report — tidak peduli sumbernya `user_report` atau `ai_news`.
- Flow ini selaras dengan **US-05 (AI News Crawler)** dan **US-07 (Auto-Flagging Duplikat)** di PRD.

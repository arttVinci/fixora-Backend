# System Flow: Government Data Sync (Kondisi Infrastruktur)

> Dokumen ini menjelaskan alur sinkronisasi data kondisi infrastruktur dari portal open data pemerintah (prioritas: SatuData Jakarta) ke tabel `reports`.  
> Referensi: **US-08 (Sumber Data Pemerintah)** di PRD.

---

## Gambaran Umum

Government Data Sync adalah **cron job** yang berjalan dengan frekuensi lebih jarang dari News Crawler (harian atau mingguan, configurable). Tugasnya: tarik dataset kondisi infrastruktur dari API pemerintah, mapping kondisi ke severity, lalu insert/update entry di tabel `reports` dengan badge sumber `gov_data`. Berbeda dari sumber lain, data pemerintah langsung mendapat status `verified` tanpa perlu melewati Multi-Agent Verification — karena sumbernya sudah resmi dan terstruktur.

> Flow ini khusus untuk `dataset_type: condition` (kondisi infrastruktur → `reports`). Sinkronisasi dataset anggaran (`dataset_type: budget` → `budget_items`) adalah flow terpisah.

---

## Alur Lengkap

### Fase 1 — Trigger & Cek Riwayat Sync

| Step | Proses | Detail |
|------|--------|--------|
| 1 | Cron job aktif | Scheduler men-trigger sync sesuai jadwal (harian/mingguan, configurable) |
| 2 | Cek riwayat | Query `gov_datasets_sync_log` — kapan terakhir sync dataset ini? Digunakan untuk menentukan apakah perlu fetch ulang atau hanya ambil data yang lebih baru |
| 3 | Fetch dataset | Panggil API SatuData Jakarta (atau sumber open data pemerintah lainnya) untuk dataset kondisi infrastruktur |

---

### Fase 2 — Mapping Kondisi ke Severity

Untuk setiap item dalam dataset yang diterima:

| Step | Proses | Detail |
|------|--------|--------|
| 4 | Mapping kondisi | Konversi terminologi pemerintah ke severity Fixora |

**Tabel mapping:**

| Kondisi di Dataset Pemerintah | Severity Fixora | Aksi |
|-------------------------------|-----------------|------|
| Rusak Berat | `parah` | Proses sebagai masalah |
| Rusak Ringan / Sedang | `sedang` | Proses sebagai masalah |
| Baik | — | Skip, bukan masalah infrastruktur |

**Percabangan:**
- **Kondisi "Baik"** → Skip item ini, lanjut ke item berikutnya
- **Kondisi mengindikasikan masalah** → Lanjut ke Fase 3

---

### Fase 3 — Resolve Wilayah & Cek Duplikat

| Step | Proses | Detail |
|------|--------|--------|
| 5 | Match wilayah | Cari kelurahan/kecamatan di tabel `villages` berdasarkan nama atau kode BPS yang ada di dataset pemerintah. Kode BPS di tabel wilayah memungkinkan pencocokan langsung (JOIN) tanpa geocoding |
| 6 | Cek report existing | Apakah sudah ada report untuk ruas/lokasi yang sama dengan `source_type: gov_data`? (berdasarkan lokasi + kategori) |

**Percabangan:**
- **Sudah ada report existing** → Lanjut ke Fase 4a (Update)
- **Belum ada** → Lanjut ke Fase 4b (Insert)

> Pencocokan wilayah di sini tidak perlu geocoding (Nominatim) karena dataset pemerintah biasanya sudah menyertakan kode wilayah BPS yang bisa langsung di-match ke tabel `villages.code`.

---

### Fase 4a — Update Report Existing

| Step | Proses | Detail |
|------|--------|--------|
| 7a | Update report | Update severity dan status terbaru berdasarkan data terkini dari pemerintah |

Lanjut ke Fase 5 (Pipeline Gabungan).

---

### Fase 4b — Insert Report Baru

| Step | Proses | Detail |
|------|--------|--------|
| 7b | Insert report | Buat entry baru di tabel `reports` |

**Data yang diisi saat insert:**

| Field | Nilai |
|-------|-------|
| `category_id` | Hasil mapping dari jenis infrastruktur di dataset |
| `village_id` | Hasil match dari kode BPS ke tabel `villages` |
| `title` | Nama ruas/lokasi dari dataset pemerintah |
| `description` | Detail kondisi dari dataset (jika tersedia) |
| `latitude` / `longitude` | Koordinat dari dataset pemerintah (jika tersedia) |
| `address` | Alamat dari dataset |
| `severity` | Hasil mapping kondisi (Fase 2) |
| `status` | `verified` — langsung, tanpa perlu Multi-Agent Verification |
| `source_type` | `gov_data` |
| `reporter_id` | `NULL` (tidak ada pelapor manusia) |
| `first_reported_at` | Tanggal survei/pencatatan di dataset pemerintah |

> Status langsung `verified` karena data berasal dari sumber resmi pemerintah yang sudah terverifikasi — tidak perlu melewati pipeline verifikasi seperti `user_report` atau `ai_news`.

Lanjut ke Fase 5 (Pipeline Gabungan).

---

### Fase 5 — Pipeline Gabungan & Sync Log

| Step | Proses | Detail |
|------|--------|--------|
| 8 | Pipeline gabungan | Report baru/updated masuk ke pipeline yang sama dengan sumber lain: |

**Pipeline Gabungan (berlaku universal untuk semua report):**
- **Duplicate Detection (US-07):** Cek apakah ada report dari sumber lain (`user_report` / `ai_news`) yang merujuk ke lokasi yang sama — jika ya, pertimbangkan soft-merge
- **RAG Cross-Reference Anggaran (Async):** Trigger `CrossReferenceBudget(reportID)` untuk mencocokkan dengan data di `budget_items`

| Step | Proses | Detail |
|------|--------|--------|
| 9 | Iterasi selesai? | Jika masih ada item lain dalam dataset → kembali ke Fase 2. Jika semua item sudah diproses → lanjut step 10 |
| 10 | Update sync log | Insert/update `gov_datasets_sync_log` dengan: waktu sync terakhir, jumlah record yang diproses, status (`success` / `failed`) |
| 11 | Selesai | Satu putaran sync selesai, menunggu jadwal berikutnya |

---

## Ringkasan Mapping ke Backend

| Fase | Proses Backend | Tabel Terkait |
|------|----------------|---------------|
| Fase 1 (Fetch Dataset) | Cron scheduler + HTTP client ke API pemerintah | `gov_datasets_sync_log` |
| Fase 2 (Mapping) | Logic internal | `categories` |
| Fase 3 (Resolve Wilayah) | Query by kode BPS | `villages`, `districts`, `cities`, `provinces` |
| Fase 3 (Cek Duplikat) | Query `WHERE source_type = 'gov_data'` | `reports` |
| Fase 4a (Update) | Update report | `reports` |
| Fase 4b (Insert) | Insert report | `reports` |
| Fase 5 (Pipeline) | Trigger async | `reports`, `merge_logs`, `budget_items` |
| Fase 5 (Sync Log) | Insert/update log | `gov_datasets_sync_log` |

---

## Catatan Desain

- Flow ini berjalan **sepenuhnya di background** — tidak ada interaksi user.
- Report dari `gov_data` langsung `verified` — tidak perlu Multi-Agent Verification karena sumber resmi.
- Mapping wilayah menggunakan **kode BPS langsung** (JOIN ke `villages.code`), bukan geocoding — lebih akurat dan cepat dibanding Nominatim.
- Duplikat lintas-sumber (misal: ruas jalan yang sama dilaporkan warga DAN tercatat di dataset pemerintah) di-handle oleh Pipeline Gabungan, bukan di sini.
- `gov_datasets_sync_log` mencatat riwayat sync agar proses berikutnya bisa incremental (hanya ambil data yang berubah sejak sync terakhir).
- Flow ini selaras dengan **US-08 (Sumber Data Pemerintah)** di PRD. Dataset anggaran (`budget` → `budget_items`) di-handle oleh flow terpisah.

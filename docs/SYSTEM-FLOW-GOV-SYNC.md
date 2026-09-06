# System Flow: Government Data Sync (Kondisi Infrastruktur)

> **Status implementasi: BELUM DIIMPLEMENTASIKAN.** Fitur ini adalah rancangan Fase 2 dan **tidak ada di codebase saat ini** (tidak ada modul gov sync, tidak ada tabel `budget_items`/`gov_datasets_sync_log`).
> Referensi: **US-08 (Sumber Data Pemerintah)** di PRD — yang saat ini juga belum diimplementasikan.

---

## Ringkasan

Dokumen ini menyimpan rancangan alur sinkronisasi data kondisi infrastruktur dari portal open data pemerintah (prioritas: SatuData Jakarta) ke tabel `reports`. Karena fitur belum dibangun, seluruh isi di bawah adalah **spesifikasi target**, bukan deskripsi perilaku codebase yang ada.

> **Catatan kontras dengan codebase:** `source_type` `gov_data` sudah didukung oleh enum entity `reports.source_type` (varchar, tanpa enum DB) dan oleh validasi request, tetapi **belum ada pipeline** yang menghasilkan report dengan sumber ini. Satu-satunya penanganan `gov_data` di codebase adalah: verifikasi multi-agent langsung meng-approve report `gov_data` (auto `verified`) tanpa menjalankan agent LLM — sama seperti `ai_news`.

---

## Gambaran Umum (Target)

Government Data Sync direncanakan sebagai **cron job** dengan frekuensi lebih jarang dari News Crawler (harian/mingguan, configurable). Tugasnya: tarik dataset kondisi infrastruktur dari API pemerintah, mapping kondisi ke severity, lalu insert/update entry di tabel `reports` dengan badge sumber `gov_data`. Karena sumbernya resmi dan terstruktur, data pemerintah direncanakan langsung berstatus `verified` tanpa melewati multi-agent verification.

> Flow ini khusus untuk `dataset_type: condition` (kondisi infrastruktur → `reports`). Sinkronisasi dataset anggaran (`dataset_type: budget` → `budget_items`) adalah flow terpisah yang juga belum diimplementasikan.

---

## Alur (Target)

### Fase 1 — Trigger & Cek Riwayat Sync

| Step | Proses | Detail |
|------|--------|--------|
| 1 | Cron job aktif | Scheduler men-trigger sync sesuai jadwal (harian/mingguan, configurable) |
| 2 | Cek riwayat | Query `gov_datasets_sync_log` — kapan terakhir sync dataset ini? |
| 3 | Fetch dataset | Panggil API SatuData Jakarta (atau sumber open data pemerintah lainnya) |

### Fase 2 — Mapping Kondisi ke Severity

| Kondisi di Dataset Pemerintah | Severity Fixora | Aksi |
|-------------------------------|-----------------|------|
| Rusak Berat | `parah` | Proses sebagai masalah |
| Rusak Ringan / Sedang | `sedang` | Proses sebagai masalah |
| Baik | — | Skip, bukan masalah infrastruktur |

### Fase 3 — Resolve Wilayah & Cek Duplikat

| Step | Proses | Detail |
|------|--------|--------|
| 4 | Match wilayah | Cari kelurahan/kecamatan berdasarkan kode BPS di `villages.code` (tanpa geocoding) |
| 5 | Cek report existing | Apakah sudah ada report `gov_data` untuk lokasi + kategori yang sama? |

### Fase 4 — Update / Insert

- **Update existing:** perbarui `severity` dan `status` berdasarkan data terkini.
- **Insert baru:** buat entry dengan `status: verified`, `source_type: gov_data`, `reporter_id: NULL`, `first_reported_at` = tanggal survei dataset.

### Fase 5 — Pipeline Gabungan & Sync Log

- Duplicate detection lintas-sumber (US-07) — pertimbangkan soft-merge.
- RAG cross-reference anggaran (async).
- Update `gov_datasets_sync_log`.

---

## Ringkasan Mapping ke Backend (Target)

| Fase | Proses Backend | Tabel Terkait |
|------|----------------|---------------|
| Fase 1 (Fetch) | Cron scheduler + HTTP client | `gov_datasets_sync_log` |
| Fase 2 (Mapping) | Logic internal | `categories` |
| Fase 3 (Resolve) | Query by kode BPS | `villages`, `districts`, `cities`, `provinces` |
| Fase 3 (Cek Duplikat) | Query `WHERE source_type = 'gov_data'` | `reports` |
| Fase 4 (Update/Insert) | Update/insert report | `reports` |
| Fase 5 (Sync Log) | Insert/update log | `gov_datasets_sync_log` |

---

## Catatan Desain (Target)

- Flow ini direncanakan berjalan **sepenuhnya di background**.
- Report `gov_data` direncanakan langsung `verified` — tanpa multi-agent verification (konsisten dengan perilaku codebase yang sudah ada untuk `ai_news`/`gov_data`).
- Mapping wilayah memakai **kode BPS langsung** (JOIN ke `villages.code`), bukan geocoding.
- `gov_datasets_sync_log` mencatat riwayat sync agar proses berikutnya bisa incremental.

> **Kesimpulan:** dokumen ini adalah backlog spesifikasi. Sebelum diimplementasikan, perlu dibuat modul gov-sync baru + tabel `budget_items`/`gov_datasets_sync_log`, karena keduanya saat ini belum ada di codebase.

# User Flow: Pelaporan Masalah Infrastruktur

> Dokumen ini menjelaskan alur pelaporan masalah infrastruktur oleh warga (sumber `user_report`).
> Referensi: **US-02 (Pelaporan Manual)**, **US-06 (CV Classifier)** di PRD.
> Source of truth: `internal/modules/report/src/usecase/analyze_photo_usecase.go`, `internal/modules/report/src/usecase/report_usecase.go`, `internal/modules/report/src/controller/report_controller.go`.

---

## Gambaran Umum

Pelaporan di Fixora terdiri dari **dua langkah API terpisah**:

1. **Analisis foto** (`POST /api/reports/analyze-photo`) — foto diunggah, disimpan ke staging Cloudinary, lalu dianalisis oleh AI (vision) untuk menghasilkan draft otomatis (title, deskripsi, kategori, severity, lokasi).
2. **Submit laporan** (`POST /api/reports`) — frontend mengirim draft final (yang sudah direview/dikoreksi user) beserta `staging_session_id`, lalu backend mempromosikan foto staging ke folder permanen dan membuat record report.

Tidak ada form panjang: foto + AI menghasilkan draft, user hanya review & koreksi sebelum submit.

---

## Langkah 1 — Upload & Analisis Foto (CV Classifier)

**Endpoint:** `POST /api/reports/analyze-photo` (multipart/form-data, field `photo`)

| Step | Proses | Detail |
|------|--------|--------|
| 1 | Upload foto | User mengunggah satu file foto (jpg/png) |
| 2 | Staging | Foto di-upload ke Cloudinary folder staging dengan public ID deterministik berdasarkan `session_id` (UUID baru) |
| 3 | Analisis AI | Foto dikirim ke Gemini vision (`gemini-3.5-flash-lite`) dengan JSON schema |
| 4 | Hasil draft | AI mengembalikan `title`, `description`, `category` (slug), `severity`, `location`, `reason`, `has_timestamp_overlay` |

**Penjaga validitas (guard, dieksekusi backend secara berurutan):**

| Guard | Syarat | Jika gagal |
|-------|--------|-----------|
| Guard 0 — stempel kamera | Foto **wajib** punya overlay teks tanggal+waktu ter-burn di gambar (dari aplikasi kamera timestamp/CCTV/dashcam) | `is_relevant: false`, reason `"bukan foto timestamp camera (tidak ada stempel tanggal & waktu)"` |
| Guard 1 — relevansi | Foto harus menunjukkan kerusakan infrastruktur publik nyata & masuk salah satu dari 4 kategori | `is_relevant: false` |
| Guard 2 — kategori terdaftar | Slug kategori harus terdaftar di `categories` | `is_relevant: false`, reason `"kategori tidak dikenali"` |
| Guard 3 — lokasi terbaca | Harus ada teks lokasi/koordinat tercetak di foto | `is_relevant: false`, reason `"lokasi tidak terbaca"` |
| Guard 4 — geocoding | Teks lokasi di-geocode via Nominatim menjadi `lat/lng` | `is_relevant: false`, reason `"lokasi tidak dapat diidentifikasi"` |

**Response (selalu HTTP 200, walau tidak relevan):**

```json
{
  "data": {
    "session_id": "<uuid>",
    "photo_url": "<url staging>",
    "title": "...",
    "description": "...",
    "category": "jalan-rusak",
    "severity": "sedang",
    "location": "...",
    "latitude": -6.2,
    "longitude": 106.8,
    "address": "...",
    "reason": "",
    "is_relevant": true
  }
}
```

> Jika `is_relevant: false`, field `category`, `severity`, `location` kosong dan `reason` berisi alasan penolakan. Frontend **wajib memblokir submit** ketika `is_relevant: false`.

---

## Langkah 2 — Review & Submit Laporan

**Endpoint:** `POST /api/reports` (application/json)

| Step | Proses | Detail |
|------|--------|--------|
| 5 | Review draft | User mereview/mengoreksi field hasil AI (title, deskripsi, kategori, severity, lokasi) |
| 6 | Submit | Frontend mengirim body JSON berisi field final + `staging_session_id` |
| 7 | Reverse geocoding | Backend meng-geocode `latitude`/`longitude` via Nominatim untuk mendapatkan komponen wilayah + alamat |
| 8 | Resolve village | Komponen wilayah di-resolve ke `village_id` via `region-client`. Gagal → 400 "Lokasi tidak teridentifikasi" |
| 9 | Promote foto | Foto staging dipindah ke folder permanen Cloudinary (satu rename, tanpa re-upload) |
| 10 | Simpan reporter | Jika `reporter_email` diisi: cari/create `reporters` berdasarkan email |
| 11 | Simpan report | Buat record `reports` dengan `status: pending_verification`, `source_type: user_report` |
| 12 | Simpan foto | Buat record `report_photos` (primary) |
| 13 | Trigger async | Setelah commit: trigger `CheckDuplicate` (deteksi duplikat) + `CreateVerification` (multi-agent) |

**Request body:**

| Field | Tipe | Wajib | Keterangan |
|-------|------|-------|-----------|
| `category_id` | string | Ya | UUID kategori |
| `title` | string | Ya | Maks 200 karakter |
| `description` | string | Tidak | |
| `latitude` | float | Ya | -90..90 |
| `longitude` | float | Ya | -180..180 |
| `address` | string | Tidak | **Diabaikan backend** — alamat di-override hasil Nominatim |
| `severity` | string | Ya | `ringan` / `sedang` / `parah` |
| `staging_session_id` | string | Ya | UUID dari endpoint `analyze-photo` |
| `reporter_email` | string | Tidak | Email pelapor (opsional) |

---

## Langkah 3 — Verifikasi & Penayangan

| Step | Proses | Detail |
|------|--------|--------|
| 14 | Verifikasi | Multi-agent verification (advocate/skeptic/manager) berjalan di background (cron tiap 30 detik) |
| 15 | Status | `verified` jika lolos, `rejected` (+ `reject_reason`) jika ditolak |
| 16 | Tayang | Report tayang di peta publik setelah `status: verified` |

> **Catatan penting:** status awal laporan warga adalah `pending_verification`. Hanya `user_report` yang melewati verifikasi multi-agent — `ai_news` dan `gov_data` langsung `verified` (lihat `verification_usecase.go`).

---

## Ringkasan Mapping ke Backend

| Langkah | Endpoint / Proses | Tabel Terkait |
|---------|-------------------|---------------|
| Analisis foto | `POST /api/reports/analyze-photo` → `AnalyzePhotoUseCase` | — (foto di Cloudinary staging) |
| Reverse geocoding | Nominatim (shared client) | — |
| Resolve village | `region-client.ResolveVillageByAddress` | `villages`, `districts`, `cities`, `provinces` |
| Submit laporan | `POST /api/reports` → `ReportUseCase.CreateReport` | `reporters`, `reports`, `report_photos` |
| Duplikat | `DuplicateUseCase.CheckDuplicate` (async) | `reports`, `report_photos`, `duplicate_reports` |
| Verifikasi | `VerificationUseCase` (async, cron) | `verification_sessions`, `verification_logs`, `reports` |

---

## Catatan Desain

- **Dua langkah, bukan chat.** Codebase tidak mengimplementasikan "asisten AI percakapan" — hanya dua endpoint stateless: analisis foto lalu submit.
- **Foto wajib punya stempel timestamp kamera.** Ini penjaga utama anti-manipulasi (foto lama/screenshot ditolak).
- **Alamat dari reverse geocode.** Field `address` pada request di-override oleh hasil Nominatim (`reverseResult.FullAddress`).
- **Tanpa login.** Pelapor tidak perlu registrasi; `reporter_email` opsional dan hanya untuk follow-up internal.
- **Tidak ada pengiriman email** di codebase saat ini.
- **Tidak ada pipeline RAG** di codebase (lihat `SYSTEM-FLOW-GOV-SYNC.md` untuk status fitur anggaran).

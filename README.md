# Fixora — Backend

> **Fixora** adalah platform open source untuk melacak akuntabilitas jangka panjang terhadap infrastruktur publik yang dibiarkan rusak (jalan berlubang, jembatan rawan roboh, bangunan terbengkalai, sampah menumpuk).
>
> Backend ini dibangun dengan **Go (Golang)**, **Fiber**, **GORM**, dan **MySQL 8.0**, memakai pola **Modular Monolith** — satu aplikasi, tapi kode dipecah per modul yang terisolasi datanya dan hanya berkomunikasi lewat interface client.

---

## Daftar Isi

1. [Apa yang Dikerjakan Backend Ini](#apa-yang-dikerjakan-backend-ini)
2. [Teknologi](#teknologi)
3. [Arsitektur](#arsitektur)
4. [Persyaratan](#persyaratan)
5. [Cara Menjalankan](#cara-menjalankan)
6. [Konfigurasi](#konfigurasi)
7. [Dokumentasi API (Swagger)](#dokumentasi-api-swagger)
8. [Daftar Endpoint](#daftar-endpoint)
9. [Cara Kerja (Flow)](#cara-kerja-flow)
10. [Struktur Direktori](#struktur-direktori)

---

## Apa yang Dikerjakan Backend Ini

Fixora punya dua jalur data yang berjalan paralel, keduanya tampil di peta yang sama:

| Jalur | Sumber | Badge |
|-------|--------|-------|
| **Laporan manual warga** | Upload foto + lokasi, dianalisis AI, lalu diverifikasi | `user_report` |
| **AI News Crawler** | Cron otonom menarik berita infrastruktur dari media, diekstrak LLM, dibuat jadi laporan otomatis | `ai_news` |

Selain itu backend juga mengelola: kategori masalah, hierarki wilayah Indonesia, deteksi duplikat, dan verifikasi multi-agent untuk menjaga data tetap kredibel.

### Modul yang sudah ada

- **`region`** — hierarki wilayah (provinsi → kota → kecamatan → kelurahan) + resolve lokasi berdasarkan nama/alamat.
- **`report`** — laporan warga (create/map/detail), kategori, analisis foto AI (CV classifier), deteksi duplikat.
- **`crawl`** — AI News Crawler (RSS + LLM + geocoding + auto-create report).
- **`verification`** — verifikasi multi-agent (advocate → skeptic → manager).

### Belum diimplementasikan (roadmap)

- REST API list wilayah (endpoint provinsi/kota/kecamatan/kelurahan).
- Endpoint konfirmasi "masih begini" (tabel sudah ada, endpoint belum).
- Government data sync (`gov_data`) & RAG cross-reference anggaran (Fase 2).

Detail lengkap ada di `docs/PROGRESS.md` dan `docs/`.

---

## Teknologi

| Layer | Teknologi |
|-------|-----------|
| Bahasa | Go 1.25 |
| HTTP framework | Fiber v2 |
| ORM | GORM (driver MySQL) |
| Database | MySQL 8.0 (lokal) / TiDB Cloud (produksi) |
| Config | Viper (`config.json` + env override) |
| Validasi | go-playground/validator |
| Logging | Logrus |
| Scheduler | robfig/cron v3 |
| RSS parsing | gofeed |
| AI (CV + ekstraksi berita) | Google Gemini (`gemini-3.5-flash-lite`) |
| AI (verifikasi multi-agent) | CommandCode (OpenAI-compatible, model `qwen/qwen3.7-flash`) |
| Geocoding | Nominatim (OpenStreetMap, gratis) |
| Penyimpanan foto | Cloudinary |
| Deteksi duplikat foto | goimagehash (perceptual hash) |

---

## Arsitektur

Pola **Modular Monolith**: satu binary/deployment unit, kode dipecah per modul.

- **Data isolation** — tiap modul punya tabel sendiri, tanpa foreign key GORM lintas modul.
- **Inter-module communication** — modul mengakses data modul lain hanya lewat interface `*-client` (mis. `report-client`, `region-client`, `verification-client`).
- **Module contract** — tiap modul implement `module.Module` (`Migrate()` + `RegisterRoutes()`), di-wiring seragam di `cmd/web/main.go`.
- **Layer** — `controller → usecase → repository → entity` + `model` (DTO) + `converter`.

Urutan inisialisasi modul di `main.go` (sesuai dependency):

```
region → report → verification → crawl
```

---

## Persyaratan

- **Go 1.25+** (jika jalan tanpa Docker)
- **Docker & Docker Compose** (cara yang direkomendasikan)
- **Git**

---

## Cara Menjalankan

### Cara 1: Docker Compose (rekomendasi)

```bash
git clone https://github.com/arttVinci/fixora-Backend.git
cd fixora-Backend
```

1. **Siapkan `.env`** (untuk variabel MySQL container):

```bash
cp .env.example .env
```

Isi default `.env`:

```env
MYSQL_ROOT_PASSWORD=rootpassword
MYSQL_DATABASE=database_name
MYSQL_USER=db_user
MYSQL_PASSWORD=database_password
DB_PORT_EXTERNAL=3306
```

2. **Siapkan `config.json`** (untuk koneksi DB + API key AI):

```bash
cp config.json.example config.json
```

3. **Jalankan**:

```bash
docker compose -f docker-compose.dev.yml up --build -d
```

Tunggu sampai MySQL siap (`ready for connections`) dan backend berjalan. Base URL default: **`http://127.0.0.1:8080`**.

### Cara 2: Tanpa Docker (Go langsung)

```bash
# Siapkan config.json terlebih dahulu (lihat bagian Konfigurasi)
go mod download
go run ./cmd/web
```

> Pastikan MySQL sudah berjalan dan nilai `database.host` di `config.json` menunjuk ke host MySQL yang benar (`localhost` jika MySQL lokal).

---

## Konfigurasi

Konfigurasi utama ada di `config.json` (dibaca Viper). Struktur lengkap:

```json
{
  "app": { "name": "fixora" },
  "web": { "prefork": false, "port": 8080 },
  "log": { "level": 6 },
  "database": {
    "username": "db_user",
    "password": "db_password",
    "host": "fixora_mysql",
    "port": 3306,
    "name": "fixora_db",
    "pool": { "idle": 10, "max": 100, "lifetime": 300 }
  },
  "jwt": { "secret": "your_jwt_secret" },
  "google_ai_studio": { "api_key": "YOUR_GEMINI_API_KEY" },
  "llm_provider": {
    "base_url": "https://api.commandcode.ai/provider/v1/chat/completions",
    "api_key": "YOUR_LLM_PROVIDER_API_KEY"
  },
  "cloudinary": {
    "cloud_name": "YOUR_CLOUD_NAME",
    "api_key": "YOUR_CLOUD_API_KEY",
    "api_secret": "YOUR_CLOUD_API_SECRET"
  }
}
```

### Penjelasan tiap blok

| Blok | Fungsi |
|------|--------|
| `web` | Port + prefork Fiber |
| `database` | Koneksi MySQL/TiDB (username, password, host, port, name, pool) |
| `jwt` | Secret (disediakan untuk kebutuhan auth mendatang) |
| `google_ai_studio` | API key Gemini — untuk **CV classifier** (analisis foto) & **ekstraksi berita** (crawler) |
| `llm_provider` | Base URL + API key CommandCode (OpenAI-compatible) — untuk **verifikasi multi-agent** |
| `cloudinary` | Kredensial penyimpanan foto (staging → permanent) |

### Override via environment variable

Koneksi database bisa di-override lewat env var (diprioritaskan di atas `config.json`):

| Env var | Menimpa |
|---------|---------|
| `DB_HOST` | `database.host` |
| `DB_USER` | `database.username` |
| `DB_PASSWORD` | `database.password` |
| `DB_NAME` | `database.name` |
| `DB_PORT` | `database.port` |

> **Penting saat pakai Docker Compose:** nilai `username`/`password`/`name` di `config.json` harus sama dengan `MYSQL_USER`/`MYSQL_PASSWORD`/`MYSQL_DATABASE` di `.env`, dan `database.host` harus `fixora_mysql` (nama container), bukan `localhost`.

### Geocoding (Nominatim)

Nominatim (OpenStreetMap) dipakai untuk geocoding/reverse geocoding **tanpa API key**. Tidak perlu konfigurasi tambahan.

---

## Dokumentasi API (Swagger)

Swagger UI dapat diakses di:

👉 **https://api.portofy.net/swagger/index.html**

Secara lokal (saat `docker compose up`), Swagger tersedia di:

👉 **http://127.0.0.1:8080/swagger/index.html**

---

## Daftar Endpoint

Base URL: `/api`

### Reports

| Method | Path | Deskripsi |
|--------|------|-----------|
| `GET` | `/reports/map` | Data peta berdasarkan bounding box (`min_lat`, `max_lat`, `min_lng`, `max_lng`) + filter `category_id`, `status`, `severity`, `source_type` |
| `GET` | `/reports/:id` | Detail satu laporan (+ foto, konfirmasi, `related_reports`) |
| `POST` | `/reports/analyze-photo` | Analisis foto (CV classifier) → draft otomatis (title, deskripsi, kategori, severity, lokasi) |
| `POST` | `/reports` | Submit laporan warga (dengan `staging_session_id` dari `analyze-photo`) |

### Categories

| Method | Path | Deskripsi |
|--------|------|-----------|
| `GET` | `/categories` | Daftar kategori masalah |

### Crawl

| Method | Path | Deskripsi |
|--------|------|-----------|
| `POST` | `/crawl/trigger` | Trigger crawler manual (jalan di background) |

### Verification

| Method | Path | Deskripsi |
|--------|------|-----------|
| `POST` | `/crawl/verify/trigger/:reportId` | Trigger verifikasi untuk satu laporan |
| `POST` | `/crawl/verify/retry/:sessionId` | Ulang sesi verifikasi yang error |
| `GET` | `/crawl/verify/sessions/:reportId` | Daftar sesi verifikasi (+ log agent) untuk satu laporan |

### Region

> Belum ada endpoint REST (hanya client internal untuk resolve village). Lihat `docs/PROGRESS.md`.

### Format response standar

Setiap endpoint mengembalikan envelope seragam `WebResponse[T]`:

```json
{
  "data": { },
  "message": "Pesan opsional",
  "success": true
}
```

Error dikembalikan sebagai `ApiErrorResponse` (`message` + `statusCode`).

---

## Cara Kerja (Flow)

### 1. Laporan Warga (`user_report`)

```
User upload foto
  → POST /reports/analyze-photo
  → Foto disimpan ke Cloudinary (staging)
  → Gemini vision menganalisis → draft (title, deskripsi, kategori, severity, lokasi)
  → Guard berlapis: wajib ada stempel timestamp kamera + lokasi terbaca + kategori terdaftar
  → Response berisi draft + session_id

User review/koreksi draft
  → POST /reports (dengan staging_session_id)
  → Reverse geocode lat/lng via Nominatim → resolve village_id
  → Foto staging dipromosikan ke folder permanen Cloudinary
  → Report disimpan dengan status pending_verification
  → (async) deteksi duplikat + verifikasi multi-agent

Verifikasi multi-agent (cron tiap 30 detik)
  → advocate → skeptic → manager (LLM CommandCode)
  → approved → status verified ; rejected → status rejected + alasan

Report tayang di peta publik setelah status verified
```

### 2. AI News Crawler (`ai_news`)

```
Cron tiap 2 jam (+ langsung jalan saat startup, + POST /crawl/trigger manual)
  → Ambil kategori + search_keywords dari DB
  → Fetch Google News RSS per keyword × region Jabodetabek
  → Filter artikel (buang opini/analisis/lama >30 hari)
  → Dedup by URL (crawled_articles.url UNIQUE)

Untuk tiap artikel unik:
  → Ekstraksi LLM Gemini (title, kategori, lokasi, severity, is_relevant)
  → Reject jika tidak relevan / kategori tidak dikenal / geocode gagal / di luar Jabodetabek
  → Resolve village_id via region-client
  → Auto-create report source_type=ai_news, status=verified (langsung tayang)

Rejected article tetap disimpan (audit trail + mencegah re-crawl)
```

Detail lengkap: `docs/SYSTEM-FLOW-CRAWLER.md`, `docs/USER-FLOW-REPORT.md`.

### 3. Deteksi Duplikat (soft-merge)

```
Setelah report dibuat:
  → Cari report nearby (radius 100m, kategori sama, belum di-merge)
  → Bandingkan perceptual hash foto primary
  → Catat similarity ke duplicate_reports (audit trail)
  → Merge hanya jika foto identik + sumber sama → set merged_into_id
  → Report yang di-merge tidak tampil di /reports/map (filter merged_into_id IS NULL)
```

### 4. Background workers (cron)

| Worker | Frekuensi | Tugas |
|--------|-----------|-------|
| AI News Crawler | tiap 2 jam | Tarik + proses berita |
| Verifikasi | tiap 30 detik | Jalankan sesi verifikasi pending |
| Staging cleanup | tiap 1 jam | Hapus foto staging orphan > TTL |

---

## Struktur Direktori

```
.
├── cmd/web/main.go          # Entrypoint + wiring modul
├── internal/
│   ├── modules/
│   │   ├── region/          # Hierarki wilayah + resolve village
│   │   ├── region-client/   # Interface client region
│   │   ├── report/          # Laporan, kategori, analisis foto, duplikat
│   │   ├── report-client/   # Interface client report
│   │   ├── crawl/           # AI News Crawler
│   │   ├── crawl-client/    # Interface client crawl
│   │   ├── verification/    # Verifikasi multi-agent
│   │   └── verification-client/ # Interface client verifikasi
│   └── shared/
│       ├── config/          # Inisialisasi DB, Fiber, Viper, LLM, Cloudinary, dll.
│       ├── client/          # Nominatim, Cloudinary, LLM (shared)
│       ├── dto/             # WebResponse, LLM DTO
│       ├── repository/      # Generic repository
│       └── modules/         # Interface Module
├── database/
│   ├── seeders/             # Seed wilayah (SQL embedded)
│   └── migrations/          # (placeholder)
├── docs/                    # PRD, schema, flow, progress, swagger
├── config.json.example      # Template konfigurasi
├── .env.example             # Template env MySQL
├── docker-compose.dev.yml   # Compose untuk development
├── docker-compose.prod.yml  # Compose untuk production
└── Dockerfile
```

---

## Dokumentasi Tambahan

- `docs/FIXORA-PRD.md` — Product Requirement Document.
- `docs/DATABASE-SCHEMA.md` — Skema database aktual (13 tabel).
- `docs/PROGRESS.md` — Status pengerjaan vs codebase.
- `docs/SYSTEM-FLOW-CRAWLER.md` — Flow AI News Crawler.
- `docs/USER-FLOW-REPORT.md` — Flow pelaporan warga.
- `docs/SYSTEM-FLOW-GOV-SYNC.md` — Rancangan gov data sync (belum diimplementasikan).
- `docs/git-convetional.md` — Aturan commit & branch workflow.

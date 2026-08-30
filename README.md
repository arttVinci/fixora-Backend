# Fixora — Backend

Fixora (Infrastructure Neglect Tracker) adalah platform pelacakan akuntabilitas jangka panjang terhadap infrastruktur publik yang dibiarkan rusak. Backend ini dibangun menggunakan **Go (Golang)**, **Fiber**, **GORM**, dan **MySQL**.

---

## Persyaratan

- **Docker & Docker Compose**
- **Git**

---

## Cara Setup Lokal

Untuk menjalankan backend ini secara lokal di mesin Anda, ikuti langkah-langkah berikut:

### 1. Clone Repositori

```bash
git clone https://github.com/arttVinci/fixora-Backend.git
cd fixora-Backend
```

### 2. Siapkan Konfigurasi `.env`

Salin template `.env.example` menjadi `.env` dan sesuaikan nilainya jika perlu:

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

### 3. Siapkan Konfigurasi `config.json`

Salin template `config.json.example` menjadi `config.json`:

```bash
cp config.json.example config.json
```

Untuk integrasi dengan Docker Compose, atur `config.json` pada bagian database host ke `fixora_mysql` serta sesuaikan kredensial dan API key:

```json
{
  "app": {
    "name": "fixora"
  },
  "web": {
    "prefork": false,
    "port": 8080
  },
  "log": {
    "level": 6
  },
  "database": {
    "username": "db_user",
    "password": "database_password",
    "host": "fixora_mysql",
    "port": 3306,
    "name": "database_name",
    "pool": {
      "idle": 10,
      "max": 100,
      "lifetime": 300
    }
  },
  "jwt": {
    "secret": "your_jwt_secret_here"
  },
  "group": {
    "id": "fixora"
  },
  "google_ai_studio": {
    "api_key": "YOUR_GEMINI_API_KEY"
  }
}
```

> [!IMPORTANT]
> **Penting:** Pastikan nilai `username`, `password`, dan `name` di dalam `config.json` **selalu sama dan sesuai** dengan nilai `MYSQL_USER`, `MYSQL_PASSWORD`, dan `MYSQL_DATABASE` pada file `.env`.
>
> Selain itu, pastikan `database.host` di `config.json` diatur ke **`fixora_mysql`** (bukan `localhost`). Jika tidak sesuai, aplikasi backend di container tidak akan bisa terhubung ke container database.
>
> Isi juga `google_ai_studio.api_key` dengan API Key dari Google AI Studio untuk mengaktifkan fitur AI News Crawler.

### 4. Jalankan dengan Docker Compose

Gunakan perintah berikut untuk melakukan build dan menyalakan container:

```bash
docker compose up --build -d
```

Tunggu beberapa saat hingga container database MySQL siap (_ready for connections_) dan backend berjalan.

---

## Informasi Endpoint

### Base URL

Bila dijalankan secara lokal dengan konfigurasi default, Base URL API adalah:
👉 **`http://127.0.0.1:8080`**

### Daftar Endpoint

#### Reports

##### 1. Search Map Reports

Mengambil data titik laporan infrastruktur untuk tampilan peta interaktif berdasarkan _bounding box_ koordinat.

- **Method:** `GET`
- **Path:** `/api/reports/map`
- **Query Parameters:**

| Parameter     | Tipe   | Wajib | Deskripsi                                                  |
| ------------- | ------ | ----- | ---------------------------------------------------------- |
| `min_lat`     | float  | Ya    | Latitude minimal                                           |
| `max_lat`     | float  | Ya    | Latitude maksimal                                          |
| `min_lng`     | float  | Ya    | Longitude minimal                                          |
| `max_lng`     | float  | Ya    | Longitude maksimal                                         |
| `category_id` | string | Tidak | Filter UUID Kategori                                       |
| `status`      | string | Tidak | Filter Status (`pending_verification`, `verified`, `rejected`) |
| `severity`    | string | Tidak | Filter Severity (`ringan`, `sedang`, `parah`)              |
| `source_type` | string | Tidak | Filter Sumber (`user_report`, `ai_news`, `gov_data`)       |

**Contoh URL Request:**

```http
GET http://127.0.0.1:8080/api/reports/map?min_lat=-7.8&max_lat=-6.2&min_lng=106.4&max_lng=108.8
```

**Contoh Response Payload:**

```json
{
  "data": [
    {
      "id": "c9a7e2f1-4b6d-4e8a-9f3c-1a2b3c4d5e6f",
      "title": "Jalan Rusak parah di Kota Bekasi",
      "latitude": -6.2349858,
      "longitude": 106.9945444,
      "severity": "sedang",
      "category_slug": "jalan-rusak",
      "status": "pending_verification",
      "photo_url": "https://res.cloudinary.com/fixora/image/upload/v1/reports/c9a7e2f1/primary.jpg",
      "source": "ai_news"
    }
  ],
  "message": "Berhasil menampilkan data peta",
  "success": true
}
```

##### 2. Get Report Detail

Mengambil detail lengkap satu laporan infrastruktur berdasarkan ID.

- **Method:** `GET`
- **Path:** `/api/reports/:id`
- **Path Parameters:**

| Parameter | Tipe   | Wajib | Deskripsi        |
| --------- | ------ | ----- | ---------------- |
| `id`      | string | Ya    | Report ID (UUID) |

**Struktur Response Data (`ReportDetailResponse`):**

| Field | Tipe | Deskripsi |
| ----- | ---- | --------- |
| `id` | string | UUID unik laporan |
| `title` | string | Judul laporan kerusakan infrastruktur |
| `description` | string (opsional) | Deskripsi rinci kondisi masalah |
| `latitude` | float | Titik koordinat garis lintang (-90 s/d 90) |
| `longitude` | float | Titik koordinat garis bujur (-180 s/d 180) |
| `address` | string (opsional) | Alamat lokasi masalah |
| `severity` | string | Tingkat keparahan (`ringan`, `sedang`, `parah`) |
| `status` | string | Status verifikasi (`pending_verification`, `verified`, `rejected`) |
| `source` | string | Asal sumber data (`user_report`, `ai_news`, `gov_data`) |
| `source_url` | string (opsional) | Tautan URL sumber berita asli (hadir jika `source` = `ai_news`) |
| `category_name` | string | Nama kategori masalah (mis. `Jalan Rusak`, `Sampah`) |
| `category_slug` | string | Slug kategori masalah (mis. `jalan-rusak`, `sampah`) |
| `photo_url` | string (opsional) | URL foto utama masalah infrastruktur |
| `additional_photos` | array of string (opsional) | Daftar URL foto tambahan pendukung |
| `total_confirmations` | integer | Total konfirmasi "masih begini" dari pengguna |
| `merged_into_id` | string (opsional) | UUID laporan induk jika laporan ini ditandai duplikat dan di-merge |
| `first_reported_at` | string (ISO 8601) | Waktu pertama kali laporan dibuat atau berita dideteksi |
| `last_confirmed_at` | string (ISO 8601, opsional) | Waktu konfirmasi kondisi terakhir dari pengguna |

**Contoh URL Request:**

```http
GET http://127.0.0.1:8080/api/reports/c9a7e2f1-4b6d-4e8a-9f3c-1a2b3c4d5e6f
```

**Contoh Response Payload (Laporan Warga — `source: user_report`):**

```json
{
  "data": {
    "id": "c9a7e2f1-4b6d-4e8a-9f3c-1a2b3c4d5e6f",
    "title": "Jalan Berlubang di Jl. Ahmad Yani Bekasi",
    "description": "Lubang berdiameter 1 meter di jalur utama.",
    "latitude": -6.2349858,
    "longitude": 106.9945444,
    "address": "Jl. Ahmad Yani, Bekasi Selatan",
    "severity": "sedang",
    "status": "pending_verification",
    "source": "user_report",
    "category_name": "Jalan Rusak",
    "category_slug": "jalan-rusak",
    "photo_url": "https://res.cloudinary.com/fixora/image/upload/v1/reports/c9a7e2f1/primary.jpg",
    "additional_photos": [],
    "total_confirmations": 3,
    "first_reported_at": "2026-08-08T10:00:00Z",
    "last_confirmed_at": "2026-08-15T14:30:00Z"
  },
  "message": "Berhasil menampilkan detail laporan",
  "success": true
}
```

**Contoh Response Payload (Deteksi Berita AI — `source: ai_news` dengan `source_url`):**

```json
{
  "data": {
    "id": "e7b1a2f3-5c8d-4e9a-9f1c-2a3b4c5d6e7f",
    "title": "Jembatan Rusak dan Amblas di Jalur Penghubung",
    "description": "Sebagian badan jembatan amblas akibat tergerus aliran sungai deras.",
    "latitude": -6.2412345,
    "longitude": 106.9987654,
    "address": "Jl. Raya Narogong, Rawalumbu, Kota Bekasi",
    "severity": "parah",
    "status": "verified",
    "source": "ai_news",
    "source_url": "https://megapolitan.kompas.com/read/2026/08/20/jembatan-amblas-bekasi",
    "category_name": "Jembatan Rusak",
    "category_slug": "jembatan-rusak",
    "photo_url": "https://res.cloudinary.com/fixora/image/upload/v1/reports/e7b1a2f3/primary.jpg",
    "additional_photos": [],
    "total_confirmations": 5,
    "first_reported_at": "2026-08-20T08:00:00Z",
    "last_confirmed_at": "2026-08-22T09:15:00Z"
  },
  "message": "Berhasil menampilkan detail laporan",
  "success": true
}
```

##### 3. Analyze Photo (CV Classifier)

Upload foto masalah infrastruktur untuk mendapatkan draft otomatis (judul, deskripsi, kategori, severity) dari AI classifier. Endpoint ini digunakan sebelum membuat laporan agar form bisa di-prefill.

- **Method:** `POST`
- **Path:** `/api/reports/analyze-photo`
- **Content-Type:** `multipart/form-data`
- **Form Data:**

| Field   | Tipe | Wajib | Deskripsi           |
| ------- | ---- | ----- | ------------------- |
| `photo` | file | Ya    | File foto (jpg/png) |

**Contoh Response Payload:**

```json
{
  "data": {
    "session_id": "e4b6a2c8-9d1f-4c3e-8a7b-2f1d5c6e9a4b",
    "photo_url": "https://res.cloudinary.com/fixora/image/upload/v1/staging/e4b6a2c8/primary.jpg",
    "title": "Jalan Berlubang Besar di Area Perumahan",
    "description": "Terlihat lubang jalan berdiameter sekitar 1 meter dengan kedalaman cukup signifikan di area jalan perumahan.",
    "category": "jalan-rusak",
    "severity": "sedang"
  },
  "message": "Berhasil menganalisis foto",
  "success": true
}
```

##### 4. Create Report (Laporan Warga)

- **Method:** `POST`
- **Path:** `/api/reports/`
- **Content-Type:** `application/json`
- **Request Body:**

| Field               | Tipe   | Wajib | Deskripsi                                       |
| ------------------- | ------ | ----- | ----------------------------------------------- |
| `category_id`       | string | Ya    | UUID kategori masalah                           |
| `title`             | string | Ya    | Judul laporan (maks. 200 karakter)              |
| `description`       | string | Tidak | Deskripsi detail masalah                        |
| `latitude`          | float  | Ya    | Latitude lokasi (-90 s/d 90)                    |
| `longitude`         | float  | Ya    | Longitude lokasi (-180 s/d 180)                 |
| `address`           | string | Tidak | Alamat lokasi (maks. 500 karakter)              |
| `severity`          | string | Ya    | Tingkat keparahan (`ringan`, `sedang`, `parah`) |
| `staging_session_id` | string | Ya    | Session ID foto staging (dari endpoint `analyze-photo`) |
| `reporter_email`    | string | Tidak | Email pelapor (opsional)                        |

**Contoh Request Body:**

```json
{
  "category_id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Jalan Berlubang di Jl. Ahmad Yani Bekasi",
  "description": "Lubang berdiameter 1 meter di jalur utama, membahayakan pengendara motor.",
  "latitude": -6.2349858,
  "longitude": 106.9945444,
  "address": "Jl. Ahmad Yani, Bekasi Selatan",
  "severity": "sedang",
  "staging_session_id": "e4b6a2c8-9d1f-4c3e-8a7b-2f1d5c6e9a4b",
  "reporter_email": "warga@example.com"
}
```

**Contoh Response Payload (201 Created):**

```json
{
  "data": {
    "id": "d2f8b4a1-7c3e-4f9b-8a5d-6e2c9b0f1a3d",
    "title": "Jalan Berlubang di Jl. Ahmad Yani Bekasi",
    "description": "Lubang berdiameter 1 meter di jalur utama, membahayakan pengendara motor.",
    "latitude": -6.2349858,
    "longitude": 106.9945444,
    "address": "Jl. Ahmad Yani, Bekasi Selatan",
    "severity": "sedang",
    "status": "pending_verification",
    "source": "user_report",
    "category_name": "Jalan Rusak",
    "category_slug": "jalan-rusak",
    "photo_url": "https://res.cloudinary.com/fixora/image/upload/v1/reports/d2f8b4a1/primary.jpg",
    "additional_photos": [],
    "total_confirmations": 0,
    "first_reported_at": "2026-08-26T06:15:00Z"
  },
  "message": "Berhasil membuat laporan",
  "success": true
}
```

---

#### Categories

##### 5. Get List Categories

Mengambil daftar seluruh kategori masalah infrastruktur yang tersedia di platform.

- **Method:** `GET`
- **Path:** `/api/categories/`

**Contoh URL Request:**

```http
GET http://127.0.0.1:8080/api/categories/
```

**Contoh Response Payload:**

```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Sampah",
      "slug": "sampah"
    },
    {
      "id": "660f9500-f3ac-52e5-b827-557766550111",
      "name": "Jalan Rusak",
      "slug": "jalan-rusak"
    },
    {
      "id": "770a0600-a4bd-63f6-c938-668877660222",
      "name": "Jembatan Rusak",
      "slug": "jembatan-rusak"
    },
    {
      "id": "880b1700-b5ce-74a7-d049-779988770333",
      "name": "Bangunan Terbengkalai",
      "slug": "bangunan-terbengkalai"
    }
  ],
  "message": "Berhasil menampilkan daftar kategori",
  "success": true
}
```

---

#### Crawl

##### 6. Trigger AI News Crawler (Manual)

Memicu proses AI News Crawler secara manual untuk mencari berita kerusakan infrastruktur di background.

- **Method:** `POST`
- **Path:** `/api/crawl/trigger`

**Contoh Response Payload:**

```json
{
  "data": null,
  "message": "Crawler berhasil di-trigger, berjalan di background",
  "success": true
}
```

---

#### Verification

##### 7. Trigger Verification

Memicu proses verifikasi berlapis (multi-agent) untuk satu laporan. Mengembalikan sesi verifikasi yang aktif bila sudah ada, atau membuat sesi baru dengan status `pending`.

- **Method:** `POST`
- **Path:** `/api/crawl/verify/trigger/:reportId`
- **Path Parameters:**

| Parameter  | Tipe   | Wajib | Deskripsi |
| ---------- | ------ | ----- | --------- |
| `reportId` | string | Ya    | Report ID |

**Contoh URL Request:**

```http
POST http://127.0.0.1:8080/api/crawl/verify/trigger/d2f8b4a1-7c3e-4f9b-8a5d-6e2c9b0f1a3d
```

**Contoh Response Payload:**

```json
{
  "data": {
    "id": "f3c9a5b2-8d4f-4a0c-9b6e-7f3d0c1a2b4e",
    "report_id": "d2f8b4a1-7c3e-4f9b-8a5d-6e2c9b0f1a3d",
    "status": "pending",
    "logs": []
  },
  "message": "Berhasil memicu verifikasi",
  "success": true
}
```

##### 8. Retry Verification Session

Mengulang sesi verifikasi yang gagal (status `error`). Sesi dikembalikan ke status `pending` dan field final di-reset.

- **Method:** `POST`
- **Path:** `/api/crawl/verify/retry/:sessionId`
- **Path Parameters:**

| Parameter   | Tipe   | Wajib | Deskripsi            |
| ----------- | ------ | ----- | -------------------- |
| `sessionId` | string | Ya    | Verification Session ID |

**Contoh Response Payload:**

```json
{
  "data": {
    "id": "f3c9a5b2-8d4f-4a0c-9b6e-7f3d0c1a2b4e",
    "report_id": "d2f8b4a1-7c3e-4f9b-8a5d-6e2c9b0f1a3d",
    "status": "pending"
  },
  "message": "Berhasil mengulang verifikasi",
  "success": true
}
```

##### 9. Get Verification Sessions by Report

Mengambil seluruh sesi verifikasi (beserta log agen) untuk satu laporan.

- **Method:** `GET`
- **Path:** `/api/crawl/verify/sessions/:reportId`
- **Path Parameters:**

| Parameter  | Tipe   | Wajib | Deskripsi |
| ---------- | ------ | ----- | --------- |
| `reportId` | string | Ya    | Report ID |

**Contoh Response Payload:**

```json
{
  "data": [
    {
      "id": "f3c9a5b2-8d4f-4a0c-9b6e-7f3d0c1a2b4e",
      "report_id": "d2f8b4a1-7c3e-4f9b-8a5d-6e2c9b0f1a3d",
      "status": "approved",
      "final_verdict": true,
      "final_category_slug": "jalan-rusak",
      "final_severity": "sedang",
      "final_reasoning": "Foto dan deskripsi konsisten dengan kerusakan jalan.",
      "decided_by": "skeptic",
      "started_at": "2026-08-26T06:15:05Z",
      "completed_at": "2026-08-26T06:16:02Z",
      "created_at": "2026-08-26T06:15:00Z",
      "updated_at": "2026-08-26T06:16:02Z",
      "logs": [
        {
          "id": "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d",
          "session_id": "f3c9a5b2-8d4f-4a0c-9b6e-7f3d0c1a2b4e",
          "agent_role": "advocate",
          "llm_provider": "gemini",
          "llm_model": "gemini-3.5-flash-lite",
          "verdict": true,
          "confidence": 0.94,
          "category_slug": "jalan-rusak",
          "severity": "sedang",
          "raw_argument": "Foto menunjukkan lubang jalan yang nyata.",
          "prompt_used": "Kamu adalah agen advokat...",
          "latency_ms": 8400,
          "created_at": "2026-08-26T06:15:11Z"
        }
      ]
    }
  ],
  "message": "Berhasil menampilkan sesi verifikasi",
  "success": true
}
```

---

## Standar Response API

Setiap endpoint API selalu mengembalikan format JSON standar berikut:

### Response Sukses:

```json
{
  "data": { ... },
  "message": "Pesan sukses opsional",
  "success": true
}
```

### Response dengan Pagination:

```json
{
  "data": [ ... ],
  "message": "Pesan sukses opsional",
  "success": true,
  "paging": {
    "page": 1,
    "size": 10,
    "total_item": 25,
    "total_page": 3
  }
}
```

### Response Error:

```json
{
  "data": null,
  "message": "Pesan error yang jelas",
  "success": false
}
```

---

## Autentikasi

Endpoint yang diproteksi memerlukan token JWT. Kirimkan token pada header request:

```http
Authorization: Bearer <token_anda_dari_login>
```

Informasi teknis dan arsitektur lebih lanjut mengenai backend dapat dilihat pada dokumen di dalam folder `docs/` (`docs/fixora-prd`).

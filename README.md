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
| `status`      | string | Tidak | Filter Status (`pending_verification`, `in_progress`, dll) |
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
      "id": "RPT-jalan-rusak-20260808-ba00",
      "title": "Jalan Rusak parah di Kota Bekasi",
      "latitude": -6.2349858,
      "longitude": 106.9945444,
      "severity": "sedang",
      "category_slug": "jalan-rusak",
      "status": "pending_verification",
      "photo_url": "",
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

| Parameter | Tipe   | Wajib | Deskripsi |
| --------- | ------ | ----- | --------- |
| `id`      | string | Ya    | Report ID |

**Contoh URL Request:**

```http
GET http://127.0.0.1:8080/api/reports/RPT-jalan-rusak-20260808-ba00
```

**Contoh Response Payload:**

```json
{
  "data": {
    "id": "RPT-jalan-rusak-20260808-ba00",
    "title": "Jalan Berlubang di Jl. Ahmad Yani Bekasi",
    "description": "Lubang berdiameter 1 meter di jalur utama.",
    "latitude": -6.2349858,
    "longitude": 106.9945444,
    "address": "Jl. Ahmad Yani, Bekasi Selatan",
    "severity": "sedang",
    "status": "pending_verification",
    "source": "user_report",
    "source_url": null,
    "category_name": "Jalan Rusak",
    "category_slug": "jalan-rusak",
    "photo_url": "https://example.com/photo.jpg",
    "additional_photos": null,
    "total_confirmations": 3,
    "merged_into_id": null,
    "first_reported_at": "2026-08-08T10:00:00Z",
    "last_confirmed_at": "2026-08-20T14:30:00Z"
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

Submit laporan baru masalah infrastruktur beserta foto, lokasi, dan detail lainnya.

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
| `primary_photo_url` | string | Ya    | URL foto utama (harus berformat URL valid)      |
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
  "primary_photo_url": "https://example.com/photo.jpg",
  "reporter_email": "warga@example.com"
}
```

**Contoh Response Payload (201 Created):**

```json
{
  "data": {
    "id": "RPT-jalan-rusak-20260826-a1b2",
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
    "photo_url": "https://example.com/photo.jpg",
    "total_confirmations": 0,
    "first_reported_at": "2026-08-26T06:15:00Z"
  },
  "message": "Berhasil membuat laporan",
  "success": true
}
```

---

#### Crawl

##### 5. Trigger AI News Crawler (Manual)

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

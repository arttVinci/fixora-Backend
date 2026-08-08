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

#### 1. Search Map Reports

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
      "title": "Pemkot Bekasi Alokasikan Rp200 Miliar untuk Perbaikan Jalan Rusak - MetroTVNews.com",
      "latitude": -6.2349858,
      "longitude": 106.9945444,
      "severity": "sedang",
      "category_slug": "jalan-rusak",
      "status": "pending_verification",
      "primary_photo_url": "",
      "source_type": "ai_news"
    }
  ],
  "message": "Berhasil menampilkan data peta",
  "success": true
}
```

#### 2. Trigger AI News Crawler (Manual)

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

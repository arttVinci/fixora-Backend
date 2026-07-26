# User Flow: Pelaporan Masalah Infrastruktur

> Dokumen ini menjelaskan alur interaksi pengguna saat melaporkan masalah infrastruktur di Fixora.  
> Pendekatan: **Conversational AI-Assisted** — pelaporan dipandu oleh Asisten AI, bukan form statis.

---

## Gambaran Umum

Pelaporan di Fixora dirancang **minim hambatan (frictionless)**. User tidak langsung disodori form panjang — melainkan dipandu step-by-step oleh Asisten AI. Langkah pertama hanya upload foto, sisanya dibantu AI (klasifikasi kategori, deteksi lokasi). User tinggal review dan konfirmasi.

---

## Alur Lengkap

### Fase 1 — Entry Point & Sambutan

| Step | Aktor | Aksi |
|------|-------|------|
| 1 | User | Membuka aplikasi Fixora |
| 2 | Asisten | Menyapa sesuai waktu: *"Selamat siang! Ada yang bisa saya bantu hari ini?"* |
| 3 | User | Memilih salah satu opsi: |

**Percabangan:**
- **"Lihat Terkini"** → Diarahkan langsung ke Peta Besar (flow selesai)
- **"Melapor"** → Lanjut ke Fase 2

---

### Fase 2 — Upload Foto & Analisis AI

| Step | Aktor | Aksi |
|------|-------|------|
| 4 | Asisten | Menjelaskan proses: *"Nanti foto Anda akan dianalisis AI, lokasi dideteksi otomatis, lalu diverifikasi sebelum tayang."* |
| 5 | Asisten | *"Silakan masukkan foto masalahnya."* |
| 6 | User | Upload foto masalah infrastruktur |
| 7 | Sistem | **CV Classifier berjalan (US-06)** — foto dianalisis untuk menentukan kategori masalah dan tingkat keparahan (*severity scoring*) |
| 8 | Asisten | Menampilkan hasil analisis dalam **form terstruktur**: Title, Deskripsi, Kategori, Tingkat Kerusakan (sudah terisi otomatis oleh AI) |

---

### Fase 3 — Review & Edit Data

| Step | Aktor | Aksi |
|------|-------|------|
| 9 | User | Mereview form hasil AI |
| 10a | User | *(Jika ada yang salah)* → Edit field yang perlu dikoreksi via modal/form → kembali ke review |
| 10b | User | *(Jika sudah oke)* → Konfirmasi data benar → lanjut ke Fase 4 |

> **Human-in-the-loop**: AI memberikan draft, tapi keputusan akhir tetap di tangan user. Jika AI salah menebak kategori atau severity, user bisa mengoreksi.

---

### Fase 4 — Deteksi & Verifikasi Lokasi

| Step | Aktor | Aksi |
|------|-------|------|
| 11 | Sistem | Mengambil titik lokasi GPS *(sudah dicatat otomatis di background sejak awal, bukan diminta ke user baru di sini)* |
| 12 | Asisten | *"Titik lokasi Anda saat ini di [alamat hasil deteksi]."* |
| 13a | User | *(Jika lokasi meleset)* → Geser pin di peta interaktif → lokasi terkonfirmasi |
| 13b | User | *(Jika sudah benar)* → Konfirmasi → lanjut ke Fase 5 |

> **Kenapa GPS diambil di background?** Supaya user tidak perlu memikirkan lokasi di awal — fokus upload foto dulu. GPS sudah berjalan diam-diam sejak user membuka halaman pelaporan.

---

### Fase 5 — Identitas Pelapor (Opsional)

| Step | Aktor | Aksi |
|------|-------|------|
| 14 | Sistem | Multi-Agent Verification (MVP-simple) — Agent Verifier melakukan pengecekan ringan terhadap foto + lokasi sebelum tayang publik |
| 15 | Asisten | *"Apakah Anda ingin mengisi email untuk konfirmasi & update laporan ini? (opsional)"* |
| 16a | User | *(Isi email)* → Email tersimpan ke tabel `reporters`, `reporter_id` terhubung ke laporan |
| 16b | User | *(Skip)* → `reporter_id` tetap `NULL`, laporan tetap anonim |
| 17 | User | Konfirmasi submit laporan |

> **Tanpa login, tanpa registrasi.** Pelapor tidak perlu membuat akun. Email bersifat opsional — hanya untuk yang ingin dapat notifikasi update status perbaikan di kemudian hari.

---

### Fase 6 — Submit & Konfirmasi

| Step | Aktor | Aksi |
|------|-------|------|
| 18 | Sistem | `POST` laporan ke server → status awal: `pending_verification` |
| 19 | Asisten | *"Terima kasih atas laporan Anda! [apresiasi]"* |
| 20 | Sistem | *(Jika email diisi)* → Kirim email apresiasi + info bahwa user akan mendapat update laporan |
| 21 | Asisten | Mengirim link: *"Lihat detail laporan Anda di sini: [link]"* |
| 22 | Sistem | Laporan tayang di peta publik **setelah lolos verifikasi** |

---

## Ringkasan Mapping ke Backend

| Fase | Endpoint / Proses Backend | Tabel Terkait |
|------|---------------------------|---------------|
| Fase 2 (CV Classifier) | `POST /api/v1/reports/analyze-photo` | — (stateless, return JSON) |
| Fase 4 (Reverse Geocoding) | Internal call ke Nominatim | `villages`, `districts`, `cities`, `provinces` |
| Fase 5 (Simpan Reporter) | Bagian dari `POST /api/v1/reports` | `reporters` |
| Fase 6 (Submit Laporan) | `POST /api/v1/reports` | `reports`, `report_photos`, `reporters` |
| Fase 6 (Kirim Email) | Background job / queue | — |
| Post-submit (Verifikasi) | Internal / cron | `reports` (update status) |

---

## Catatan Desain

- **Warna kuning (⚙️)** pada flowchart asli menandai proses backend/AI yang berjalan di belakang layar.
- **Warna biru** menandai respons Asisten AI yang ditampilkan ke user.
- Flow ini selaras dengan **US-02 (Pelaporan Manual)**, **US-04 (Konfirmasi Status)**, dan **US-06 (CV Classifier)** di PRD.

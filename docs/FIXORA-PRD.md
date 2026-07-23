# Product Requirements Document (PRD)

# Fixora — Infrastructure Neglect Tracker

**Versi:** 1.0 (Draft MVP)
**Status:** Draft
**Tipe Project:** Open Source Software (OSS)
**Tech Stack:** Go (Backend) + React/TypeScript (Frontend)

---

## 1. Latar Belakang & Tujuan (Objective)

Di kota-kota besar Indonesia seperti Jakarta dan Bekasi, masalah infrastruktur publik yang dibiarkan rusak dalam jangka waktu lama — jalan berlubang, jembatan rawan roboh, bangunan terbengkalai, sampah menumpuk, drainase tersumbat — adalah persoalan yang berulang dan jarang mendapat akuntabilitas jangka panjang. Platform pelaporan yang ada saat ini (Qlue, LAPOR!) sudah ada, namun memiliki dua kelemahan utama: Qlue sudah tidak terawat (domain resminya bahkan sudah dialihkan ke situs tidak relevan), sementara LAPOR! masih berjalan tapi teknologinya tertinggal dan modelnya bersifat "lapor sekali, selesai" tanpa mekanisme pelacakan durasi masalah dibiarkan.

Fixora dibangun sebagai platform open source yang mengisi celah ini: bukan sekadar tempat lapor, tapi sistem **pelacakan akuntabilitas jangka panjang** terhadap infrastruktur yang mangkrak, didukung oleh AI yang secara aktif mencari data (bukan hanya menunggu laporan warga).

**Tujuan utama:**

- Membangun platform crowdsourced + AI-driven yang memetakan masalah infrastruktur publik secara transparan
- Mengurangi ketergantungan pada laporan manual warga dengan sumber data otonom (AI news crawler)
- Menyediakan bukti akuntabilitas dengan cross-reference ke data anggaran resmi pemerintah
- Menjadi portofolio teknis yang mendalam untuk pengembangan skill AI engineering (RAG, computer vision, multi-agent systems)

---

## 2. Problem Statement

1. **Tidak ada visibilitas publik terhadap durasi masalah dibiarkan.** Warga tahu ada jalan rusak, tapi tidak ada cara mudah untuk melihat "sudah berapa lama ini dibiarkan" secara terdokumentasi dan dapat diverifikasi.
2. **Platform pelaporan eksisting bersifat reaktif dan "sekali pakai".** LAPOR!/Qlue dirancang untuk laporan tunggal, bukan pelacakan status berkelanjutan.
3. **Tidak ada korelasi antara laporan kerusakan dan data anggaran resmi.** Warga tidak tahu apakah suatu titik kerusakan sebenarnya sudah dianggarkan untuk perbaikan atau belum, sehingga sulit menuntut akuntabilitas berbasis data.
4. **Ketergantungan penuh pada partisipasi warga menyebabkan cold-start problem.** Platform crowdsourced murni sulit berkembang di awal karena data kosong sebelum ada massa pengguna.
5. **Data infrastruktur UMKM/wilayah kecil seringkali tidak terdokumentasi** di platform manapun, termasuk peta digital besar sekalipun.

---

## 3. Goals & Value Proposition

### Goals

- Menyediakan peta interaktif yang menampilkan titik-titik masalah infrastruktur beserta durasi mangkrak
- Mengotomasi sebagian besar proses pengumpulan data awal melalui AI (news crawling), sehingga platform punya data sejak hari pertama tanpa bergantung pada laporan warga
- Membangun mekanisme verifikasi berlapis (AI + crowdsource) agar data yang tampil kredibel
- Menghubungkan laporan dengan data anggaran resmi pemerintah untuk memberi bukti akuntabilitas

### Value Proposition

| Untuk             | Fixora memberikan                                                                                        |
| ----------------- | -------------------------------------------------------------------------------------------------------- |
| Warga umum        | Transparansi kondisi infrastruktur di sekitar mereka, dengan bukti historis                              |
| Media & aktivis   | Data siap pakai untuk investigasi/liputan soal proyek mangkrak                                           |
| Pemerintah daerah | Sinyal awal (early signal) titik masalah yang perlu ditindaklanjuti, dilengkapi cross-reference anggaran |
| Kontributor OSS   | Codebase nyata untuk belajar penerapan AI (RAG, CV, multi-agent) dalam konteks sistem produksi           |

**Diferensiasi vs kompetitor (Qlue/LAPOR!):**
Fixora tidak hanya menerima laporan, tapi **secara aktif mencari** isu infrastruktur lewat AI news crawler, melacak durasi masalah dibiarkan, dan mencocokkan lokasi laporan dengan data anggaran resmi — sesuatu yang tidak dilakukan kompetitor eksisting.

---

## 4. Target Pengguna (User Personas) & User Flow

### Persona 1 — Warga Pelapor ("Rina, 29, karyawan swasta")

- Tinggal di area urban (Jakarta/Bekasi), sering menemukan jalan rusak/sampah di rute hariannya
- Ingin melaporkan masalah dengan cepat tanpa proses rumit
- Tidak terlalu peduli teknis, hanya ingin tahu "apakah laporan saya didengar"

**User Flow — Pelaporan:**

```
Buka aplikasi → Lihat peta → Tekan tombol "Lapor Masalah"
→ Ambil/upload foto → AI otomatis deteksi kategori & severity
→ Konfirmasi lokasi (auto-detect GPS / pin manual)
→ Isi deskripsi singkat (opsional) → Submit
→ Laporan masuk antrian verifikasi (Agent Verifier)
→ Setelah lolos verifikasi, laporan tayang di peta publik
```

### Persona 2 — Warga Pemantau ("Budi, 34, aktif di komunitas lokal")

- Tidak melapor, tapi rutin cek kondisi wilayah tempat tinggalnya
- Ingin tahu titik mana yang paling lama dibiarkan, untuk bahan advokasi ke RT/RW atau media sosial

**User Flow — Eksplorasi Data:**

```
Buka aplikasi → Lihat peta → Filter berdasarkan kategori/durasi mangkrak
→ Klik titik masalah → Lihat detail (foto, riwayat, status anggaran)
→ Konfirmasi "masih begini" jika relevan → Share ke media sosial (opsional)
```

### Persona 3 — Jurnalis/Peneliti ("Sari, 27, jurnalis data")

- Butuh data terverifikasi & bisa dijadikan bahan investigasi
- Tertarik pada fitur cross-reference anggaran (bukti proyek dianggarkan tapi tidak selesai)

**User Flow — Riset Data:**

```
Buka aplikasi → Cari wilayah spesifik → Lihat daftar titik mangkrak terlama
→ Buka detail → Lihat cross-reference data SIRUP/APBD
→ Ekspor/screenshot data sebagai bahan liputan
```

### Persona 4 — Sistem AI (Non-manusia, tapi krusial untuk flow)

- News Crawler yang berjalan otonom, mencari isu infrastruktur dari media
- Berperan mengisi data platform tanpa menunggu laporan warga

**Flow Otomatis:**

```
Cron job jalan berkala → Tarik berita dari RSS/NewsAPI
→ LLM ekstraksi (lokasi, kategori, severity) dalam format JSON
→ Simpan sebagai entry "AI-sourced" (beda label dari laporan warga)
→ Tampil di peta dengan badge "Sumber: Media" untuk transparansi asal data
```

---

## 5. Kebutuhan Fitur (Requirements & User Stories)

### 5.1 Fitur Inti (Core)

**US-01 — Peta Interaktif**

> Sebagai pengguna, saya ingin melihat peta dengan marker per kategori masalah, agar saya bisa cepat mengenali jenis dan lokasi masalah di sekitar saya.

- Marker dengan warna/icon berbeda per kategori (jalan, jembatan, sampah, bangunan, drainase)
- Marker clustering saat zoom out
- Filter by kategori, durasi mangkrak, wilayah

**US-02 — Pelaporan Masalah (Manual)**

> Sebagai warga, saya ingin melaporkan masalah infrastruktur lewat foto & lokasi, agar masalah tersebut terdokumentasi secara publik.

- Upload foto (wajib), deskripsi (opsional)
- Auto-detect lokasi GPS, dengan opsi koreksi manual pin
- Submit laporan masuk status "pending verification"

**US-03 — Detail & Riwayat Titik Masalah**

> Sebagai pengguna, saya ingin melihat detail satu titik masalah, agar saya tahu sejak kapan masalah ini dibiarkan.

- Foto, lokasi, deskripsi
- Tanggal pertama dilaporkan/terdeteksi
- Tanggal terakhir dikonfirmasi masih bermasalah
- Status: Mangkrak / Dalam Perbaikan / Selesai
- Badge sumber data: "Laporan Warga" atau "Terdeteksi AI (Media)"

**US-04 — Konfirmasi Status ("Masih Begini")**

> Sebagai pengguna, saya ingin mengkonfirmasi apakah suatu titik masih bermasalah, agar data tetap akurat/update.

- Tombol konfirmasi pada halaman detail
- Sistem menghitung ulang "confidence score" data berdasarkan waktu sejak konfirmasi terakhir

### 5.2 Fitur AI (MVP Scope)

**US-05 — AI News Crawler**

> Sebagai sistem, saya ingin mengambil berita terkait infrastruktur mangkrak secara berkala, agar platform punya data tanpa bergantung pada laporan warga.

- Cron job berjalan tiap beberapa jam
- Ekstraksi lokasi, kategori, dan severity dari teks berita via LLM (structured output/JSON mode)
- Entry baru otomatis masuk sebagai "AI-sourced", menunggu validasi ringan sebelum tayang

**US-06 — Computer Vision Classifier & Severity Scoring**

> Sebagai sistem, saya ingin otomatis mengklasifikasi kategori masalah dan tingkat keparahannya dari foto yang diupload, agar warga tidak perlu memilih kategori secara manual.

- Menggunakan multimodal LLM (vision) untuk klasifikasi kategori + skor keparahan
- Fallback: jika confidence rendah, kategori masuk status "perlu review manual"

**US-07 — Auto-Flagging Duplikat/Mencurigakan**

> Sebagai sistem, saya ingin mendeteksi laporan duplikat atau mencurigakan, agar data di peta tidak penuh entri yang sama/spam.

- Perceptual hashing untuk foto identik
- Pengecekan radius GPS + kategori + rentang waktu untuk deteksi kemungkinan duplikat
- Laporan yang terdeteksi duplikat digabung (merge), bukan dihapus

### 5.3 Fitur Lanjutan (Fase 2+, referensi — lihat Out of Scope)

- Multi-agent verification (Classifier vs Verifier)
- RAG cross-reference SIRUP/APBD
- Predictive risk scoring
- Agentic workflow otomatis (generate surat pengaduan)
- Satellite imagery change detection

---

## 6. Metrik Kesuksesan (Success Metrics)

| Kategori            | Metrik                                                              | Target (3 bulan pasca-rilis MVP)                     |
| ------------------- | ------------------------------------------------------------------- | ---------------------------------------------------- |
| Adopsi              | Jumlah entri masalah aktif di peta                                  | Min. 200 titik (gabungan AI-sourced + laporan warga) |
| Adopsi              | Jumlah pengguna unik yang membuka peta                              | Min. 500 unique visitors/bulan                       |
| Engagement          | Jumlah laporan manual dari warga                                    | Min. 30 laporan/bulan                                |
| Engagement          | Jumlah konfirmasi "masih begini"                                    | Min. 50 konfirmasi/bulan                             |
| Kualitas Data       | Persentase entri AI-sourced yang lolos validasi tanpa revisi manual | ≥ 70%                                                |
| Kualitas Data       | Tingkat akurasi klasifikasi CV (dibanding label manual sampel)      | ≥ 80%                                                |
| Reliabilitas Sistem | Uptime News Crawler (cron job berjalan sesuai jadwal)               | ≥ 95%                                                |
| Komunitas OSS       | Jumlah star/fork repository GitHub                                  | Min. 50 star dalam 3 bulan pertama                   |
| Komunitas OSS       | Jumlah kontributor eksternal (non-founder)                          | Min. 2 kontributor                                   |

_Catatan: angka target di atas adalah baseline awal dan dapat disesuaikan setelah data traksi riil tersedia pasca-rilis._

---

## 7. Out of Scope (Di Luar Cakupan MVP)

Fitur/ide berikut **tidak** akan dikerjakan pada fase rilis MVP, untuk mencegah scope creep:

1. **Multi-Agent Verification kompleks** (lebih dari 2 agent, sistem debate/consensus) — MVP hanya menggunakan 1 lapis klasifikasi + validasi dasar (bukan multi-agent penuh)
2. **RAG cross-reference SIRUP/APBD** — membutuhkan effort besar untuk data cleaning; ditunda ke Fase 2 setelah fondasi platform stabil. _Catatan: pengambilan raw data SIRUP/APBD (open government data) bisa dimulai lebih awal sebagai referensi manual/statis, namun pipeline RAG penuh (chunking, embedding, vector DB, retrieval otomatis) tetap masuk Fase 2, bukan MVP._
3. **Predictive Risk Scoring (forecasting ML)** — membutuhkan dataset historis yang belum tersedia; ditunda ke Fase 3
4. **Satellite/Aerial Imagery Analysis** — kompleksitas & kebutuhan expertise GIS terlalu tinggi untuk MVP; bersifat opsional jangka panjang
5. **Agentic Workflow otomatis** (auto-generate surat pengaduan, notifikasi ke instansi) — ditunda hingga sistem verifikasi dasar terbukti akurat
6. **Social Media Listening** (X/Twitter, Instagram, TikTok) — kendala biaya API & pembatasan scraping platform
7. **Google Maps Review Mining** — kendala kuota API berbayar
8. **IoT/Sensor Data Integration** (BMKG/BNPB) — membutuhkan kerjasama resmi instansi, di luar kapasitas tim saat ini
9. **Traffic Anomaly Detection** — data traffic real-time berbiaya tinggi, tidak feasible untuk MVP
10. **Sistem notifikasi push/email ke pengguna** — akan dipertimbangkan setelah basis pengguna aktif terbentuk
11. **Aplikasi mobile native (iOS/Android)** — MVP berbasis web responsive terlebih dahulu
12. **Multi-bahasa (selain Bahasa Indonesia)** — fokus pasar domestik di tahap awal
13. **Integrasi pembayaran/donasi** — tidak relevan dengan tujuan MVP saat ini

---

## Lampiran: Tech Stack Ringkas

| Layer              | Teknologi                                                                          |
| ------------------ | ---------------------------------------------------------------------------------- |
| Backend            | Go (Fiber, GORM, Logrus, Viper, Validator) — Clean Architecture / Modular Monolith |
| Frontend           | React + TypeScript (Hooks, Service Layer, Axios config, API Error Handler)         |
| Database           | PostgreSQL                                                                         |
| AI/LLM             | Multimodal LLM API (klasifikasi foto + ekstraksi berita)                           |
| Peta               | Leaflet.js / Mapbox + OpenStreetMap                                                |
| Infrastruktur Repo | 2 repository terpisah: `fixora-be` & `fixora-fe`                                   |
| Lisensi            | MIT (rekomendasi)                                                                  |

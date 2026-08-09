# Git Commit Convention & Branch Workflow — Fixora

Dokumen ini adalah aturan wajib untuk semua kontributor (termasuk AI coding agent) saat bekerja di repository Fixora. Tujuannya menjaga histori commit tetap bersih, mudah ditelusuri, dan branch tidak tercampur aktivitas dari fitur lain.

---

## 1. Prinsip Dasar: 1 Aktivitas = 1 Commit

**Jangan menumpuk beberapa aktivitas berbeda dalam satu commit besar.** Satu commit harus merepresentasikan satu perubahan logis yang bisa dijelaskan dalam satu kalimat singkat.

### ❌ Salah — Commit menumpuk banyak hal

```
commit: "update crawler, fix bug report, tambah field baru, refactor service"
```

Ini menyulitkan proses review, membuat `git blame`/`git log` tidak berguna untuk menelusuri kapan sebuah perubahan spesifik terjadi, dan menyulitkan revert jika salah satu bagian ternyata bermasalah.

### ✅ Benar — Dipecah per aktivitas

```
commit 1: "feat(crawl): tambah rate limiter untuk panggilan LLM"
commit 2: "fix(report): perbaiki query bounding box yang salah filter status"
commit 3: "feat(report): tambah field search_keywords di category entity"
commit 4: "refactor(crawl): pisahkan filter pre-LLM ke fungsi terpisah"
```

**Aturan praktis:** begitu 1 aktivitas di 1 file (atau kumpulan file yang berkaitan erat) selesai dan sudah diverifikasi jalan, **langsung commit** — jangan ditunda sampai mengerjakan hal lain dulu.

---

## 2. Format Commit Message — Conventional Commits

Menggunakan standar [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <deskripsi singkat>

[body opsional — jelaskan detail/alasan jika perlu]

[footer opsional — misal: Closes #12]
```

### Type yang dipakai

| Type       | Kapan Dipakai                                                               |
| ---------- | --------------------------------------------------------------------------- |
| `feat`     | Menambah fitur/fungsi baru                                                  |
| `fix`      | Memperbaiki bug                                                             |
| `refactor` | Mengubah struktur kode tanpa mengubah perilaku (misal pisah fungsi, rename) |
| `docs`     | Perubahan dokumentasi (README, docs/, komentar kode)                        |
| `test`     | Menambah/mengubah test                                                      |
| `chore`    | Perubahan non-kode (dependency, config, `.gitignore`, dll)                  |
| `perf`     | Perubahan yang murni untuk peningkatan performa                             |
| `style`    | Perubahan format kode (spasi, indentasi) tanpa mengubah logika              |

### Scope

Scope diisi nama modul yang terdampak, sesuai struktur folder `internal/modules/`:

```
feat(crawl): ...
fix(report): ...
feat(intake): ...
refactor(pipeline): ...
docs(schema): ...
chore(deps): ...
```

Kalau perubahan lintas-modul dan tidak bisa dipisah scope-nya, boleh tanpa scope atau pakai scope umum:

```
chore: update go.mod dependencies
```

### Contoh Lengkap

```
feat(crawl): tambah pre-filter berdasarkan path URL sebelum LLM

Menambahkan fungsi shouldSkipBeforeLLM() yang mengecek path URL
artikel (/opini/, /kolom/, dll) sebelum artikel dikirim ke LLM
untuk ekstraksi. Ini mengurangi jumlah panggilan LLM yang terbuang
untuk artikel yang jelas bukan laporan kondisi lapangan.
```

```
fix(report): perbaiki query bounding box tidak filter status verified

Sebelumnya GET /reports mengembalikan semua status termasuk
pending_verification dan rejected. Sekarang wajib filter
WHERE status = 'verified' untuk endpoint publik.
```

---

## 3. Kebersihan Branch — Wajib Commit + Push Sebelum Pindah Branch

**Aturan mutlak:** sebelum berpindah branch untuk mengerjakan aktivitas lain, **commit DAN push** dulu semua perubahan di branch saat ini — jangan cuma commit lokal, jangan cuma `git stash`.

### Skenario Contoh

Posisi saat ini: branch `feature/crawl-module`, sedang mengerjakan crawler. Tiba-tiba perlu mengerjakan sesuatu di modul `report`.

**Langkah yang benar:**

```bash
# 1. Selesaikan aktivitas terakhir di branch ini dulu
git add .
git commit -m "feat(crawl): tambah dedup check sebelum insert crawled_articles"

# 2. WAJIB push ke remote — bukan cuma commit lokal
git push origin feature/crawl-module

# 3. Baru pindah branch
git checkout -b feature/report-module
# atau kalau branch-nya sudah ada:
git checkout feature/report-module

# 4. Kerjakan aktivitas report di branch yang baru
```

### Kenapa WAJIB Push, Bukan Cuma Commit Lokal

Kalau hanya commit lokal tanpa push, lalu pindah branch, ada risiko:

1. **Commit "nyangkut" di branch yang salah** — kalau tidak sengaja membuat branch baru dari state yang belum ter-push, commit yang harusnya milik `feature/crawl-module` bisa ikut terbawa ke branch baru, bikin histori bercampur.
2. **Branch jadi kotor** — commit yang seharusnya sudah "selesai" di satu branch tapi belum ter-push berarti remote tidak tahu progress itu ada, rawan hilang kalau terjadi masalah di local (misal harus reset/reinstall).
3. **Kolaborasi terganggu** — kalau ada kontributor lain, mereka tidak akan lihat progress branch itu sampai di-push, padahal kelihatannya sudah "commit".

**Push itu bukan opsional "nanti aja sekalian banyak"** — push setiap kali selesai 1 commit yang menandakan 1 aktivitas selesai, terutama sebelum pindah context ke branch/modul lain.

---

## 4. Format Nama Branch

```
feature/<nama-modul-atau-fitur>
fix/<deskripsi-singkat-bug>
refactor/<deskripsi-singkat>
docs/<deskripsi-singkat>
```

Contoh:

```
feature/crawl-module
feature/report-module
feature/multi-agent-verification
feature/auto-flagging-duplicate
fix/rate-limit-crawler
docs/database-schema
```

---

## 5. Checklist Sebelum Pindah Branch

Sebelum meninggalkan sebuah branch untuk mengerjakan hal lain, pastikan:

- [ ] Semua perubahan yang berkaitan dengan 1 aktivitas terakhir sudah di-commit (bukan setengah-setengah)
- [ ] Commit message mengikuti format Conventional Commits di atas
- [ ] Sudah `git push origin <nama-branch>` — bukan cuma commit lokal
- [ ] `git status` menunjukkan working tree bersih (tidak ada perubahan uncommitted yang tertinggal)
- [ ] Baru boleh `git checkout` ke branch lain

---

## 6. Ringkasan Alur Kerja

```
Mulai kerjakan 1 aktivitas
        ↓
Selesai & sudah diverifikasi jalan
        ↓
git add [file terkait aktivitas ini saja]
git commit -m "type(scope): deskripsi"
        ↓
git push origin [branch saat ini]
        ↓
Mau kerjakan aktivitas lain di modul/branch berbeda?
        ↓
   YA → pindah branch (checkout), ulangi dari awal
   TIDAK → lanjut aktivitas berikutnya di branch yang sama,
           ulangi dari commit berikutnya
```

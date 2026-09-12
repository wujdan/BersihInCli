# Bersihin — Storage Optimizer (CLI)

> **Bersihin** — "bersihkan" penyimpanan laptop/komputer dengan aman.

Pembersih penyimpanan berbasis terminal (CLI) untuk **Windows** (juga berjalan di
Linux & macOS). Sistem memindai drive atau folder, mengidentifikasi file yang
tidak perlu — cache, temp, log lama, duplikat, installer bekas, dan file besar
yang sudah lama tidak diakses — lalu memindahkannya ke **karantina** yang bisa
dipulihkan. Bukan penghapusan langsung, sehingga data tetap aman jika terjadi
kesalahan.

![Go](https://img.shields.io/badge/Go-1.27-blue)
![Platform](https://img.shields.io/badge/Windows%20%7C%20Linux%20%7C%20macOS-lightgrey)
![CLI](https://img.shields.io/badge/CLI-Cobra%20%2B%20Bubble%20Tea-9cf)

> **Versi GUI desktop** (`Bersihin.exe`) berada di repository terpisah dan
> memakai **backend Go yang sama persis** dengan CLI ini. Artinya file yang
> dikarantina dari GUI bisa dipulihkan/dihapus dari CLI dan sebaliknya.

---

## Daftar Isi

- [Fitur](#fitur)
- [Prinsip Desain](#prinsip-desain)
- [Persyaratan](#persyaratan)
- [Instalasi (Build)](#instalasi-build)
- [Mulai Cepat](#mulai-cepat)
- [Panduan Perintah](#panduan-perintah)
  - [`scan`](#scan)
  - [`review`](#review)
  - [`restore`](#restore)
  - [`purge`](#purge)
  - [`config`](#config)
- [Navigasi Dashboard TUI](#navigasi-dashboard-tui)
- [Mode Scan](#mode-scan)
- [Klasifikasi File & Confidence Score](#klasifikasi-file--confidence-score)
- [Quarantine & Masa Retensi](#quarantine--masa-retensi)
- [Auto-Purge & Penjadwalan](#auto-purge--penjadwalan)
- [Konfigurasi](#konfigurasi)
- [Struktur Data & Penyimpanan](#struktur-data--penyimpanan)
- [Audit Trail](#audit-trail)
- [Mekanisme Keamanan](#mekanisme-keamanan)
- [Exit Codes](#exit-codes)
- [Struktur Kode](#struktur-kode)
- [Pengembangan & Test](#pengembangan--test)
- [Keterbatasan](#keterbatasan)
- [Disclaimer](#disclaimer)

---

## Fitur

- **Scan multi-drive** — memindai semua drive/partisi lokal (`full`), folder umum
  (`quick`), atau folder spesifik (`custom`). Drive eksternal/net di-skip
  secara default.
- **Deteksi duplikat** — pengelompokan file berdasarkan ukuran, lalu hash
  **SHA-256** untuk file dengan ukuran sama (>= 1 MB; file raksasa hanya
  di-hash 100 MB pertama untuk kecepatan).
- **Klasifikasi + confidence score** — 5 kategori kandidat dengan tingkat
  keyakinan, lengkap dengan alasan ringkas kenapa sebuah file layak dihapus.
- **Dashboard TUI interaktif** — pilih per kategori atau per file, ringkasan
  total langsung ter-update.
- **Karantina wajib** — file **tidak pernah** langsung dihancurkan; selalu
  dipindahkan dulu ke folder karantina yang bisa dipulihkan.
- **Validasi ulang 7 langkah** — dijalankan otomatis sesaat sebelum file
  dipindahkan; file yang berubah, terkunci, atau symlink otomatis dilewati.
- **Audit trail** — semua aksi tercatat dalam log JSONL berotasi harian.
- **Mode non-interaktif (`--yes`)** — cocok untuk Task Scheduler / cron.
- **Progress live** — penghitungan jumlah file tampil real-time saat scan.
- **Auto-purge daemon (`--every`)** — hapus file yang lewat retensi secara
  berkala.
- **Export laporan** — hasil scan bisa diekspor ke CSV / TSV / JSON.

---

## Prinsip Desain

1. **Tidak ada penghapusan tanpa persetujuan eksplisit dari user.**
2. Setiap penghapusan melalui tahap **karantina** sebelum dihapus permanen.
3. Seluruh proses tercatat dalam **log/audit trail**.
4. Mendukung mode non-interaktif untuk automasi dengan flag eksplisit (`--yes`).

---

## Persyaratan

- **Go 1.27** atau lebih baru (untuk build dari sumber)
- Sistem operasi: Windows (target utama), Linux, macOS
- Terminal yang mendukung ANSI/UTF-8 (Windows Terminal, PowerShell, cmd, dll.)

## Instalasi (Build)

```powershell
# dari folder proyek
go build -o storage-optimizer.exe .
```

Cara pakai tanpa build:

```powershell
.\storage-optimizer.exe scan --mode=full
```

> Anda juga dapat memakai `storage-optimizer.exe` yang sudah ter-bundle di
> root repo.

---

## Mulai Cepat

```powershell
# 1. Scan semua drive lokal dan review hasilnya
storage-optimizer scan

# 2. Lihat kandidat & dashboard interaktif
#    (navigasi: spasi = centang, y = approve, q = batal)

# 3. Pulihkan file yang tidak jadi dihapus
storage-optimizer restore

# 4. Hapus permanen file yang sudah lewat masa retensi
storage-optimizer purge
```

---

## Panduan Perintah

### `scan`

Menjalankan pemindaian penyimpanan. Mode interaktif menampilkan dashboard TUI
setelah scan selesai.

```bash
storage-optimizer scan --mode=full
storage-optimizer scan --mode=quick
storage-optimizer scan --mode=custom --path=D:\Data
storage-optimizer scan --mode=full --dry-run        # simulasi, tidak mengubah apa pun
storage-optimizer scan --mode=full --yes            # non-interaktif (auto-approve)
storage-optimizer scan --mode=full --yes --min-confidence=0.8
storage-optimizer scan --mode=full --include-external  # ikutkan drive eksternal
storage-optimizer scan --mode=quick --export=hasil.csv # ekspor daftar kandidat
```

Rangkuman setelah scan (termasuk kapasitas tersisa):

```text
SISTEM PEMBERSIHAN PENYIMPANAN
  Cakupan scan:
    - testdata

  HASIL PEMINDAIAN
  --------------------------------------------------------------------------
  File dipindai : 5
  Direktori     : 5
  Total ukuran  : 1.0 MB
  Kandidat hapus: 2 file (40 B)
  Waktu         : 6ms
  Kapasitas    : 57.6 GB tersisa dari 195.3 GB
  --------------------------------------------------------------------------
```

**Flag `scan`**

| Flag | Fungsi | Default |
|---|---|---|
| `--mode` | `full` \| `quick` \| `custom` | `full` |
| `--path` | Path target (wajib untuk `--mode=custom`) | — |
| `--dry-run` | Simulasi scan, tanpa opsi hapus | `false` |
| `-y, --yes` | Auto-approve, tanpa dashboard (untuk cron/Task Scheduler) | `false` |
| `--min-confidence` | Ambang confidence untuk auto-pilih (0–1) | `0.9` |
| `--include-external` | Ikutkan drive eksternal/network | `false` |
| `--export` | Tulis daftar kandidat ke `.csv`, `.tsv`, atau `.json` | — |

### `review`

Membuka dashboard hasil scan terakhir tanpa melakukan scan baru.

```bash
storage-optimizer review
```

### `restore`

Memulihkan file dari karantina kembali ke lokasi asli.

```bash
storage-optimizer restore          # daftar file di karantina
storage-optimizer restore <id>     # pulihkan satu file
storage-optimizer restore all      # pulihkan semua file
```

Contoh daftar:

```text
Manifest ~/.storage-optimizer/manifest.json
  3 file di quarantine
  [20260911_203253-97d138a9] app.log   475.3 KB  Log Files   ret s.d. 2026-10-11
  [20260911_203253-79dbd546] file1.tmp  18 B      Cache       ret s.d. 2026-09-25
  [20260911_203253-0a07079b] file2.tmp  18 B      Cache       ret s.d. 2026-09-25
```

> Restore lintas-drive (mis. karantina di `C:` → asli di `D:`) otomatis memakai
> *copy + delete* karena Windows menolak rename antar-disk.

### `purge`

Menghapus permanen isi karantina. **Wajib konfirmasi** dengan mengetik `yes`
kecuali memakai `--force`.

```bash
storage-optimizer purge               # hapus hanya yang LEWAT masa retensi
storage-optimizer purge <id>          # hapus satu file tertentu
storage-optimizer purge --now         # hapus SEMUA segera (hati-hati!)
storage-optimizer purge --force       # lewati konfirmasi (untuk skrip)
storage-optimizer purge --every=24h   # auto-purge daemon (lihat bagian Auto-Purge)
```

Prompt konfirmasi:

```text
  PERINGATAN: tindakan ini menghapus file secara PERMANEN dari karantina.
  File yang dihapus tidak dapat dipulihkan lagi.
  Sasaran  : file yang sudah lewat masa retensi
  Ketik 'yes' untuk melanjutkan, 'no' untuk membatalkan:
```

### `config`

Menampilkan atau membuat konfigurasi aktif.

```bash
storage-optimizer config               # tampilkan konfigurasi aktif
storage-optimizer config --generate    # buat file konfigurasi
```

---

## Navigasi Dashboard TUI

Saat hasil scan tampil (mode interaktif):

| Tombol | Aksi |
|---|---|
| `↑` / `↓` atau `j` / `k` | Pindah antar baris |
| `space` | Centang / batal centang |
| `enter` | Expand kategori → lihat daftar file |
| `a` | Pilih semua |
| `n` | Kosongkan semua |
| `y` | Approve & lanjut ke karantina |
| `q` | Keluar tanpa mengubah apa pun |

---

## Mode Scan

| Mode | Sasaran |
|---|---|
| `full` | Semua drive/partisi lokal (drive eksternal & network dilewati) |
| `quick` | Folder umum panas: `Downloads`, `Temp`, `Explorer`, `INetCache` |
| `custom` | Path spesifik via `--path` |

Root scan selalu di-izinkan walau masuk exclude list (agar `quick` tetap
berfungsi); daftar exclude tetap berlaku untuk anak-anak folder di bawahnya.

---

## Klasifikasi File & Confidence Score

Urutan keputusan classifier (file yang lebih dulu cocok menang):

```
File masuk
   │
   ├─ ① File penting? ────────────────→ SELAMATKAN (BUKAN kandidat)
   ├─ ② Cache / Temp? ────────────────→ Kandidat 96% (AMAN)
   ├─ ③ Log berumur ≥90 hari? ─────────→ Kandidat 94% (AMAN)
   ├─ ④ Installer berumur ≥90 hari? ───→ Kandidat 68% (REVIEW)
   ├─ ⑤ Duplikat (hash sama)? ─────────→ Kandidat 88% (REVIEW)
   ├─ ⑥ Diubah ≤30 hari? ──────────────→ SELAMATKAN (diduga dipakai)
   └─ ⑦ Besar ≥100MB & lama ≥365 hari? → Kandidat 55% (REVIEW)
```

### Kategori Kandidat

| Kategori | Indikator | Confidence | Label | Retensi |
|---|---|---|---|---|
| **Cache & Temp** | `.tmp` `.temp` `.cache` `.swp` `.lock`; folder berisi `cache/temp/tmp/trash/preview`; `thumbs.db`; file `~...` | **96%** | AMAN | 14 hari |
| **Log Files** | `.log` `.logs` `.logt` berumur ≥ 90 hari | **94%** | AMAN | 30 hari |
| **Installer Lama** | `.msi` `.msix` `.dmg` `.pkg` `.deb` `.rpm` berumur ≥ 90 hari | **68%** | REVIEW | 90 hari |
| **Duplikat** | Hash SHA-256 sama & ukuran ≥ 1 MB | **88%** | REVIEW | 30 hari |
| **File Besar Tidak Terpakai** | ≥ 100 MB dan tidak diakses ≥ 365 hari | **55%** | REVIEW | 90 hari |

> Rule ⑦ memakai **atime asli** (last access real dari OS) dengan fallback ke
> `ModTime`, dan mengambil nilai yang *paling baru* — file yang baru saja dibaca
> tidak akan dianggap "terlantar".

### Label

| Label | Arti | Perlakuan `--yes` |
|---|---|---|
| **AMAN** | Confidence ≥ 90% | Tercentang / diproses otomatis |
| **REVIEW** | Confidence < 90% | Tidak tercentang; perlu keputusan manual |

### File yang TIDAK PERNAH Ditawarkan untuk Dihapus

- **Kode sumber:** `.go` `.rs` `.py` `.js` `.ts` `.java` `.c` `.cpp` `.php` `.sql` `.sh` dll.
- **Dokumen:** `.pdf` `.docx` `.xlsx` `.pptx` `.txt` `.md` `.epub` dll.
- **Media pribadi:** `.jpg` `.png` `.mp4` `.mkv` `.mp3` `.wav` `.flac` dll.
- **Data pribadi:** `.vcf` `.ics` `.eml` `.pst` `.ost`.
- **Database / proyek:** `.db` `.sqlite` `.dwg` `.psd` `.ai`.
- **Dotfiles:** `.gitignore` `.gitkeep` `.env`.

---

## Quarantine & Masa Retensi

Lokasi: `~/.storage-optimizer/quarantine/`

```
~/.storage-optimizer/
├── manifest.json                  ← daftar semua file karantina
├── quarantine/
│   └── 2026-09-11/                ← folder per tanggal
│       └── 20260911_203253-97d138a9_app.log
├── logs/
│   └── audit-2026-09-11.jsonl     ← audit trail harian
└── reports/
    ├── scan-20260911-203253.json
    └── last-scan-report.json
```

Saat file dipindahkan ke karantina, sistem:

1. menghitung **SHA-256** penuh file (untuk verifikasi & audit);
2. memberi **ID unik** `YYYYMMDD_HHMMSS-<8-huruf-hash>`;
3. mencatat entri ke `manifest.json` (tulis atomik);
4. jika karantina dan file asli beda disk → **copy + delete** dengan cadangan
   `.part` (file tidak pernah hilang walau proses gagal di tengah).

Retensi default per kategori: Cache/Temp **14 hari**, Log/Duplikat **30 hari**,
Besar/Installer **90 hari**.

---

## Auto-Purge & Penjadwalan

### Daemon `--every`

```bash
storage-optimizer purge --every=12h
```

Menjalankan proses yang memeriksa file kedaluwarsa secara berkala dan
menghapusnya saat lewat masa retensi. Berhenti dengan `Ctrl+C`.

### Untuk mesin yang sering mati/hidup (Task Scheduler)

Gunakan Task Scheduler Windows pada event *logon* / *weekly*:

```
schtasks /create /tn "Bersihin AutoPurge" /tr "\path\to\storage-optimizer.exe purge --force" /sc weekly /d MON /st 09:00
```

- `--force` dipakai karena sesi non-interaktif tidak bisa mengetik `yes`.
- Aman: tanpa `--now`, `purge` **hanya** menghapus file yang sudah lewat retensi.

---

## Konfigurasi

Konfigurasi default: `config/default.yaml`. File kustom via flag `--config` atau
`storage-optimizer config --generate`.

```yaml
app:
  data_dir: "~/.storage-optimizer"
  retention_days: 30

scan:
  include_external: false        # jangan scan drive eksternal
  worker_count: 0                # 0 = otomatis (maks 8)
  hash_threshold_bytes: 104857600  # 100 MB — hash prefix utk file raksasa
  dedup_min_size: 1048576          # 1 MB — file lebih kecil tak di-hash

exclude:
  windows:
    - C:\Windows
    - C:\Program Files
    - C:\ProgramData
    - C:\$Recycle.Bin
    - C:\Users\*\AppData\Local\Temp
  linux:
    - /proc
    - /sys
    - /dev
    - /boot
    - /run
  darwin:
    - /System
    - /Library
  custom: []                    # tambahkan folder lain di sini

retention:
  cache_temp_days: 14           # retensi Cache & Temp
  log_duplicate_days: 30        # retensi Log & Duplikat
  large_old_days: 90            # retensi file besar / installer

thresholds:
  important_age_days: 30        # file diubah < 30 hari → dianggap dipakai
  large_file_bytes: 104857600   # ambang "file besar"
  old_file_days: 365            # "tidak terpakai" = 1 tahun
  installer_age_days: 90
  log_age_days: 90
```

---

## Struktur Data & Penyimpanan

Tipe data inti:

```
FileMeta    → path, size, mod_time, access_time, extension, is_dir
Candidate   → Meta + Label + Category + Confidence + Reason + DupGroup
ScanReport  → GeneratedAt, Duration, Roots, FilesScanned, TotalBytes,
              DirsScanned, Candidates[], Errors[]
Manifest    → ID, OriginalPath, QuarantinePath, Size, SHA256,
              MovedAt, RetentionUntil, Category, Label
```

Konsistensi & atomicity:

- `manifest.json` ditulis **atomik** (`.tmp` → rename) agar tidak korup jika
  proses mati mendadak.
- Pindah antar-disk: *copy → delete sumber → rename* + pembersihan `.part`.
- ID karantina berbasis timestamp + hash → tidak pernah bertabrakan.

---

## Audit Trail

Semua peristiwa penting ditulis ke `logs/audit-YYYY-MM-DD.jsonl` (satu JSON per
baris, berotasi harian):

```json
{"timestamp":"2026-09-11T20:32:53Z","event":"quarantine_move","level":"info",
 "message":"pindahkan ke quarantine",
 "data":{"id":"20260911_203253-97d138a9","path":"D:\\Data\\logs\\app.log",
         "quarantine":"...\\quarantine\\2026-09-11\\20260911_203253-97d138a9_app.log",
         "size":475325,"retention_until":"2026-10-11"}}
```

| Event | Kapan | Level |
|---|---|---|
| `scan_started` / `scan_complete` | Pemindaian dimulai / selesai | info |
| `quarantine_move` | File dipindahkan ke karantina | info |
| `restore` | File dipulihkan | info |
| `purge` | File dihapus permanen | info |
| `validation_skip` | File dilewati validasi | warn |
| `error` | Terjadi kesalahan | error |

---

## Mekanisme Keamanan

Pertahanan berlapis (defense in depth):

```
┌ 1. Exclude list        → folder sistem & data penting dilewati
├ 2. Klasifikasi         → file penting tidak pernah jadi kandidat
├ 3. Confidence score    → ≥90% = AMAN, <90% = perlu review
├ 4. Konfirmasi tunggal  → user harus setuju eksplisit
├ 5. Validasi 7 langkah  → file berubah/dipakai otomatis dilewati
├ 6. Quarantine          → file belum dihancurkan, bisa pulih
├ 7. Manifest + SHA-256  → setiap file terjejak & terverifikasi
└ 8. Audit trail         → semua aksi tercatat permanen
```

Validasi ulang 7 langkah sebelum file dipindahkan:

1. File masih ada
2. Bukan direktori
3. Waktu modifikasi tidak berubah (±10 detik)
4. Ukuran masih sama
5. Bukan symlink/junction
6. Folder induk dapat diakses
7. File tidak terkunci (sedang dibuka aplikasi → otomatis dilewati)

---

## Exit Codes

| Kode | Arti |
|---|---|
| `0` | Sukses |
| `1` | Kesalahan tak terduga |
| `2` | Pemakaian flag/path salah |
| `130` | Dibatalkan user (Ctrl+C / SIGTERM) |

---

## Struktur Kode

```
.
├── main.go                       → entry point CLI
├── config/default.yaml           → konfigurasi bawaan
├── internal/
│   ├── app/                      → orkestrasi pipeline (scan, execute)
│   ├── cli/                      → perintah Cobra (scan, review, restore,
│   │                               purge, config), deteksi drive, export
│   ├── classifier/               → klasifikasi & confidence score
│   ├── config/                   → muat/simpan config YAML
│   ├── disk/                     → kapasitas disk (GetDiskFreeSpaceEx/Statfs)
│   ├── logging/                  → audit trail JSONL
│   ├── models/                   → tipe data inti (FileMeta, ScanReport, ...)
│   ├── quarantine/               → folder karantina + manifest
│   ├── safety/                   → validasi 7 langkah
│   ├── scanner/                  → walk direktori + deteksi duplikat SHA-256
│   ├── store/                    → simpan laporan atomik
│   └── ui/                       → dashboard TUI (Bubble Tea)
└── testdata/                     → data contoh untuk pengujian
```

---

## Pengembangan & Test

```bash
go build ./...     # kompilasi semua paket
go vet ./...       # static check
go test ./...      # jalankan unit test
```

Paket dengan unit test: `classifier`, `safety`, `quarantine`, `scanner`, `cli`
(export). Snapshot smoke-test:

```bash
go build -o storage-optimizer.exe .
./storage-optimizer.exe scan --mode=custom --path=testdata --dry-run
```

---

## Keterbatasan

- **Progress bar** scan menampilkan jumlah file (drain satu arah). Pause/resume
  penuh belum didukung — gunakan `Ctrl+C` untuk membatalkan dengan aman.
- **Notifikasi berkala** (email/desktop) belum tersedia; kombinasi
  Task Scheduler + `--yes` sudah mencakup sebagian besar kebutuhan.
- Klasifikasi berbasis aturan (heuristic), bukan ML. Confidence score adalah
  perkiraan, bukan jaminan.
- Windows mematikan atime di beberapa skenario; rule "file besar lama" memakai
  `max(modtime, atime)` untuk menghindari negatif palsu.

---

## Disclaimer

Alat ini memindahkan dan menghapus file. Meskipun didesain berlapis-lapis
(karantina + validasi + audit), **tidak ada jaminan tanpa risiko**:

- Selalu buat **backup** data penting Anda.
- Tinjau setiap kandidat pada dashboard sebelum approve.
- File yang dihapus permanen (`purge`) **tidak dapat dipulihkan lagi**.

Gunakan dengan tanggung jawab Anda sendiri.
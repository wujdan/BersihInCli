# Dokumentasi Alur Sistem — Bersihin (Storage Optimizer)

**Versi:** 1.1.0
**Tanggal:** 11 September 2026
**Status:** Lengkap (GUI Desktop + CLI)

---

## Daftar Isi

- [Bagian 1 — Gambaran Umum](#bagian-1--gambaran-umum)
- [Bagian 2 — Arsitektur: GUI & CLI](#bagian-2--arsitektur-gui--cli)
- [Bagian 3 — Alur Sistem Inti (Dipakai Keduanya)](#bagian-3--alur-sistem-inti-dipakai-keduanya)
- [Bagian 4 — Panduan Aplikasi GUI (Bersihin.exe)](#bagian-4--panduan-aplikasi-gui-bersihinexe)
- [Bagian 5 — Panduan CLI (storage-optimizer)](#bagian-5--panduan-cli-storage-optimizer)
- [Bagian 6 — Klasifikasi File & Confidence Score](#bagian-6--klasifikasi-file--confidence-score)
- [Bagian 7 — Mekanisme Keamanan](#bagian-7--mekanisme-keamanan)
- [Bagian 8 — Quarantine & Masa Retensi](#bagian-8--quarantine--masa-retensi)
- [Bagian 9 — Konfigurasi](#bagian-9--konfigurasi)
- [Bagian 10 — Struktur Data & Penyimpanan](#bagian-10--struktur-data--penyimpanan)
- [Bagian 11 — Audit Trail](#bagian-11--audit-trail)
- [Bagian 12 — Kepatuhan Standar](#bagian-12--kepatuhan-standar)
- [Lampiran](#lampiran)

---

# Bagian 1 — Gambaran Umum

**Bersihin** adalah aplikasi pembersih penyimpanan (storage optimizer) untuk Windows yang memindai drive atau folder, mengidentifikasi file yang tidak perlu (cache, temp, log lama, duplikat, installer bekas, file besar lama), lalu membantu pengguna membersihkannya secara **aman**.

### Kata Kunci

| Istilah | Arti |
|---------|------|
| **Kandidat** | File yang memenuhi kriteria layak dibersihkan |
| **Quarantine** | Area aman (`~/.storage-optimizer/quarantine/`) tempat file ditahan sebelum dihapus permanen |
| **Confidence Score** | Nilai 0–1 seberapa yakin sistem bahwa file aman dihapus |
| **Retensi** | Jangka waktu file boleh dipulihkan sebelum dihapus |
| **Restore** | Memindahkan file kembali dari karantina ke lokasi asli |
| **Purge** | Menghapus file karantina secara permanen |
| **Dry-run** | Simulasi scan tanpa mengubah apa pun |

### Sistem Punya Dua Antarmuka, Satu Mesin

| | **Aplikasi GUI** | **Alat CLI** |
|---|---|---|
| Eksekutabel | `Bersihin.exe` | `storage-optimizer.exe` |
| Pengguna target | Orang awam (klik & centang) | Pengguna teknis / otomasi |
| Tampilan | Jendela desktop + tombol + checkbox | Terminal + dashboard bergaya TUI |
| Instalasi | Double-click, tidak perlu terminal | Perintah di CMD/PowerShell |
| Cocok untuk | Membersihkan secara visual, pemilihan manual | `--yes`, penjadwalan via Task Scheduler |

> Keduanya memakai **backend Go yang sama persis** dan **data dir yang sama**
> (`~/.storage-optimizer`). Artinya file yang dikarantina dari GUI bisa
> dipulihkan/dihapus dari CLI dan sebaliknya.

---

# Bagian 2 — Arsitektur: GUI & CLI

```
┌──────────────────────────────────────────────────────────────┐
│                     PENGUNA / USER                            │
└──────────────────────┬───────────────────────────────────────┘
                       │
        ┌──────────────┴───────────────┐
        │                              │
┌───────▼────────┐        ┌───────────▼───────────┐
│   GUI Desktop  │        │        CLI            │
│  Bersihin.exe  │        │  storage-optimizer    │
│                │        │                       │
│  Halaman:      │        │  Perintah:            │
│  • Pindai      │        │  • scan/review        │
│  • Quarantine  │        │  • restore/purge      │
│                │        │  • config             │
└───────┬────────┘        └───────────┬───────────┘
        │                              │
        └──────────────┬───────────────┘
                       │
        ┌──────────────▼────────────────────────────┐
        │              BACKEND INTI (GO)            │
        │                                            │
        │  Scanner → Dedup → Classifier → Safety     │
        │  → Quarantine → Store → Logging            │
        └────────────────────────────────────────────┘
                       │
                       ▼
        ┌─────────────────────────────────┐
        │   Data dir: ~/.storage-optimizer │
        │  manifest · quarantine · logs   │
        │  reports · config               │
        └─────────────────────────────────┘
```

### Antarmuka GUI (Wails)

```
Bersihin.exe
├── backend Go (wails bindings di app.go)
│   └── memanggil internal/* yang sama dengan CLI
└── frontend (HTML/CSS/JS — Vite)
    ├── index.html      ← layout 2 halaman
    ├── src/style.css   ← tema gelap modern
    └── src/main.js     ← logika UI + pemanggilan binding
```

### Antarmuka CLI (Cobra + Bubble Tea)

```
storage-optimizer.exe
├── cmd (Cobra)       ← scan, review, restore, purge, config
└── UI dashboard (Bubble Tea)
    └── tabel interaktif dengan navigasi keyboard
```

---

# Bagian 3 — Alur Sistem Inti (Dipakai Keduanya)

Alur berikut **sama persis** di GUI maupun CLI — hanya tampilannya yang berbeda.

## Tahap 1 — Memilih Sasaran Pemindaian

| GUI | CLI |
|-----|-----|
| Dropdown drive + tombol **📁 Cari** (dialog folder native Windows) + input jalur bebas | `--mode=full` (semua drive) atau `--mode=custom --path=...` |

## Tahap 2 — Menentukan Cakupan (Drive & Exclude)

- Drive removable (USB, SD, network) **otomatis dilewati** — melindungi media eksternal.
- Folder sistem dilewati via exclude list, contoh: `C:\Windows`, `C:\Program Files`, `C:\ProgramData`, `C:\$Recycle.Bin`, `C:\Users\*\AppData\Local\Temp`.
- Daftar exclude bisa ditambah di file konfigurasi.

## Tahap 3 — Pemindaian Paralel

- Penjelajahan direktori + baca metadata (path, ukuran, modtime, ekstensi) dengan beberapa worker (max 8).
- GUI menampilkan **progress bar live**; CLI menampilkan hitung file.

## Tahap 4 — Deteksi Duplikat

- File ≥ 1 MB dikelompokkan per ukuran; setiap kelompok 2+ anggota di-hash **SHA-256** (prefix 100 MB untuk file raksasa).
- Hash identik → kandidat **Duplikat** (confidence 88%).

## Tahap 5 — Klasifikasi (lihat Bagian 6)

Urutan keputusan classifier:

```
File masuk
   │
   ├─ ① Ekstensi/konteks penting? ──→ SELAMATKAN (bukan kandidat)
   │
   ├─ ② Cache / Temp? ────────────→ Kandidat 96% (AMAN)
   │
   ├─ ③ Log berumur ≥90 hari? ─────→ Kandidat 94% (AMAN)
   │
   ├─ ④ Installer berumur ≥90 hari? → Kandidat 68% (REVIEW)
   │
   ├─ ⑤ Duplikat? ────────────────→ Kandidat 88% (REVIEW)
   │
   ├─ ⑥ Diubah ≤30 hari? ──────────→ SELAMATKAN (diduga dipakai)
   │
   └─ ⑦ Besar ≥100MB & lama ≥365 hari? → Kandidat 55% (REVIEW)
```

> Aturan cache/temp (②) diproses **sebelum** batas umur (⑥): berada di folder
> cache/temp sudah cukup sinyal, dan file yang sedang terpakai akan tertangkap
> oleh validasi lock-file di tahap 9.

## Tahap 6 — Dashboard Hasil Scan

| GUI | CLI |
|-----|-----|
| Kartu per kategori + checkbox, klik kategori untuk expand, centang per file/kategori | Tabel TUI dengan keyboard: `↑↓/jk` pindah, `space` centang, `enter` expand, `a` semua, `n` kosong, `y` approve |

## Tahap 7 — Ringkasan & Konfirmasi Tunggal

- Bar total menampilkan **jumlah file + ukuran yang akan dibebaskan**.
- **Satu tombol/konfirmasi** sebelum aksi — tidak ada perubahan tanpa persetujuan.

## Tahap 8 — Keputusan Akhir User

| Aksi | Hasil |
|------|-------|
| Approve | File diproses ke tahap 9–10 |
| Batal / keluar | Tidak ada yang diubah |

## Tahap 9 — Validasi Diam, 7 Langkah

Dijalankan ulang pada setiap file **sesaat sebelum dipindahkan**:

1. File masih ada
2. Bukan direktori
3. Waktu modifikasi tidak berubah (±10 detik) → file aktif = dilewati
4. Ukuran masih sama
5. Bukan symlink/junction
6. Folder induk dapat diakses
7. File tidak terkunci (lock) → file yang sedang dibuka aplikasi otomatis dilewati

File gagal validasi → dilewati + dicatat di audit.

## Tahap 10 — Pemindahan ke Quarantine

- SHA-256 penuh dihitung sebelum pindah.
- Tujuan: `~/.storage-optimizer/quarantine/YYYY-MM-DD/<ID>_<nama-file>`
- ID unik: `YYYYMMDD_HHMMSS-<8-huruf-hash>` (contoh: `20260911_203253-97d138a9`).
- Windows: rename antar-disk gagal → otomatis **copy + delete** dengan cadangan `.part`; jika gagal di tengah, sisa parsial dibersihkan (tidak ada data hilang).
- Catatan masuk ke `manifest.json`.

## Tahap 11 — Laporan & Audit

- Laporan scan → `~/.storage-optimizer/reports/scan-<timestamp>.json` + `last-scan-report.json`.
- Audit trail JSONL mencatat: `scan_started`, `scan_complete`, `quarantine_move`, `restore`, `purge`, `validation_skip`, `error`.

## Tahap 12 — Pulihkan, Retensi & Hapus Permanen

- **Restore** kapan saja selagi file masih di quarantine.
- **Purge** tanpa argumen = hanya file yang lewat masa retensi.
- **Purge `--now` / GUI "Hapus Terpilih"** = hapus segera sesuai pilihan user.

---

# Bagian 4 — Panduan Aplikasi GUI (Bersihin.exe)

> Cara membuka: **double-click `Bersihin.exe`** — tidak perlu terminal apa pun.

Jendela: 960×680 (min 800×560). Tema gelap. Dua halaman di panel kiri:
**Pindai** dan **Quarantine**.

## 4.1 Halaman Pindai

### Langkah 1 — Pilih sasaran

Pilih **salah satu** dari tiga cara:

1. **Dropdown drive** (C:\, D:\, dst) — hanya drive local yang tampil; drive eksternal otomatis disaring agar data USB tidak terhapus.
2. Klik tombol **"📁 Cari"** → dialog folder native Windows → pilih folder mana saja di seluruh sistem.
3. **Ketik langsung** jalur folder pada kolom input (contoh: `D:\Proyek\Foto`).

### Langkah 2 — Pilih mode

| Mode | Arti |
|------|------|
| **Folder ini saja** | Scan hanya folder yang dipilih (custom) |
| **Full scan** | Scan semua drive local |

### Langkah 3 — Mulai & pantau

- Klik **Mulai Pindai** → progress bar `Pemindahan...` berjalan dengan jumlah file.
- Klik **Batal** untuk menghentikan.

### Langkah 4 — Review hasil

```
Scan selesai · 56.485 file · 44.3 GB

  Toggle Kategori            File            Ukuran          Confidence
  ■ [Cache & Temp]            318 file       2.2 MB            96%
  ■ [Log Files]                 1 file     475.3 KB            94%
  □ [Installer Lama]            5 file       2.1 GB            68%
  □ [File Besar Tidak Terpakai] 35 file     28.5 GB            55%
  □ [Duplikat]                  2 file       5.1 GB            88%

  ⏳ 319 file — 2.7 MB akan dibebaskan          [HAPUS PERMANEN]
```

- Kategori dengan confidence **≥ 90%** tercentang otomatis (aman).
- **Klik baris kategori** untuk membuka daftar file → centang/batal per file.
- Centang kategori untuk memilih semuanya.
- Tombol **HAPUS PERMANEN** hanya aktif jika ada yang tercentang.

### Langkah 5 — Approve

Klik **HAPUS PERMANEN** → file yang dicentang melewati validasi 7 langkah
dan dipindahkan ke **Karantina** (bukan dihapus!). Notifikasi hijau muncul jika
berhasil.

## 4.2 Halaman Quarantine

Menampilkan tabel semua file yang dikarantina:

```
□ Nama File | Kategori | Ukuran | Expired | Waktu | Aksi
□ app.log   | Log Files| 475 KB | 2026-10-11 | ... | [Pulihkan] [Hapus]
```

### Tombol aksi

| Tombol | Fungsi | Kapan dipakai |
|--------|--------|----------------|
| **Pulihkan Terpilih** | Kembalikan file yang dicentang ke lokasi asli | Tidak jadi dihapus |
| **Hapus Terpilih** | Hapus **segera & permanen** file yang dicentang (konfirmasi dulu) | File benar-benar tidak terpakai |
| **Hapus Semua Expired** | Hapus semua file yang sudah lewat masa retensi | Sekali klik bersihkan |
| **Pulihkan** / **Hapus** per baris | Tindakan cepat untuk satu file | Manual |

### Caranya:

1. Centang checkbox pada file yang ingin diproses (atau checkbox header untuk **pilih semua**).
2. Bar ringkasan "X file terpilih" muncul.
3. Klik tombol sesuai keinginan.
4. Hapus permanen selalu meminta konfirmasi **sebelum file dihancurkan**.

### Notifikasi (Toast)

Setiap aksi memberi notifikasi kanan-bawah: **hijau** = sukses,
**merah** = gagal (beserta alasan).

---

# Bagian 5 — Panduan CLI (storage-optimizer)

> Cara membuka: jalankan dari **PowerShell / CMD** (bukan double-click).
> Materi UI interaktif pada bagian ini dijelaskan dari alur perintah.

## 5.1 Perintah Inti

```bash
storage-optimizer scan     # pindai penyimpanan
storage-optimizer review   # dashboard hasil scan terakhir
storage-optimizer restore  # pulihkan file dari quarantine
storage-optimizer purge    # hapus permanen isi quarantine
storage-optimizer config   # tampilkan / buat konfigurasi
```

## 5.2 Scan

```bash
# Scan semua drive local
storage-optimizer scan --mode=full

# Scan folder tertentu
storage-optimizer scan --mode=custom --path=D:\Data

# Simulasi tanpa mengubah apa pun
storage-optimizer scan --mode=full --dry-run

# Mode non-interaktif (otomasi) — hanya kandidat confidence ≥ ambang
# (default 90%; ubah dengan --min-confidence) yang dipindahkan
storage-optimizer scan --mode=full --yes
storage-optimizer scan --mode=full --yes --min-confidence=0.8

# Ekspor daftar kandidat ke CSV/TSV/JSON
storage-optimizer scan --mode=quick --export=hasil.csv

# Quick scan
storage-optimizer scan --mode=quick
```

### Alur di terminal (mode interaktif)

Pipeline yang tampil saat `scan` berjalan:

```text
Pemindaian... hasil terlihat seperti di bawah
────────────────────────────────────────────
  [baris permintaan kategori, navigasi keyboard]
    1.  ● Cache & Temp ...      318 file       2.2 MB     Confidence: 96%
        Log Files ...            1 file       475.3 KB    Confidence: 94%
        Installer Lama ...       5 file        2.1 GB     Confidence: 68%
        ...
────────────────────────────────────────────
  Tekan data kategori Y/n?  →
```

Navigasi dashboard (TUI):

| Tombol | Aksi |
|--------|------|
| `↑` / `↓` atau `j` / `k` | Pindah antar baris |
| `space` | Centang / batal centang file |
| `enter` | Expand kategori → lihat daftar file |
| `a` | Pilih semua |
| `n` | Kosongkan semua |
| `y` | Approve & lanjut ke karantina |
| `q` | Keluar tanpa mengubah apa pun |

> Mode `--yes` menghilangkan tahap interaktif: langsung memproses
> kandidat AMAN (≥90%) ke karantina. Cocok dijadwalkan di Task Scheduler.

## 5.3 Review

```bash
storage-optimizer review
# Membuka dashboard hasil scan terakhir tanpa melakukan scan baru
```

## 5.4 Restore (Pulihkan)

```bash
storage-optimizer restore          # lihat daftar file di quarantine
storage-optimizer restore <id>     # pulihkan satu file, contoh file-id
storage-optimizer restore all      # pulihkan semua file
```

Contoh tampilan daftar:

```text
Manifest ~/.storage-optimizer/manifest.json
  3 file di quarantine
  [20260911_203253-97d138a9] app.log   475.3 KB  Log Files   ret s.d. 2026-10-11
  [20260911_203253-79dbd546] file1.tmp  18 B      Cache       ret s.d. 2026-09-25
  [20260911_203253-0a07079b] file2.tmp  18 B      Cache       ret s.d. 2026-09-25
```

> Pulihkan lintas-drive (mis. karantina di C: → asli di D:) otomatis
> memakai *copy + delete* karena Windows menolak rename antar-disk.

## 5.5 Purge (Hapus Permanen)

```bash
storage-optimizer purge            # hapus hanya yang LEWAT masa retensi
storage-optimizer purge <id>       # hapus satu file tertentu
storage-optimizer purge --now      # hapus SEMUA segera (kelola dengan hati-hati)
storage-optimizer purge --force    # lewati prompt konfirmasi (skrip/automasi)
storage-optimizer purge --every=24h  # auto-purge terjadwal: periksa tiap 24 jam
```

> `purge` dan `purge --now` membutuhkan konfirmasi `y` sebelum mengeksekusi.
> `--every` menjalankan Tahap 12 (auto-purge) sebagai proses daemon; berhenti
> dengan Ctrl+C. Untuk penjadwalan saat mesin mati, kombinasikan dengan
> Task Scheduler yang memanggil `storage-optimizer purge --force` berkala.

## 5.6 Config

```bash
storage-optimizer config                # lihat konfigurasi aktif
storage-optimizer config --generate     # buat file konfigurasi
```

---

# Bagian 6 — Klasifikasi File & Confidence Score

## 6.1 Kategori Kandidat

| Kategori | Ekstensi/Indikator | Confidence | Label | Retensi |
|----------|-------------------|------------|-------|---------|
| **Cache & Temp** | `.tmp` `.temp` `.cache` `.swp` `.lock`, folder berisi `cache/temp/tmp/trash/preview`, `thumbs.db`, file `~...` | **96%** | AMAN | 14 hari |
| **Log Files** | `.log` `.logs` `.logt`, berumur ≥ 90 hari | **94%** | AMAN | 30 hari |
| **Installer Lama** | `.msi` `.msix` `.dmg` `.pkg` `.deb` `.rpm`, berumur ≥ 90 hari | **68%** | REVIEW | 90 hari |
| **Duplikat** | Hash SHA-256 sama & ukuran ≥ 1 MB | **88%** | REVIEW | 30 hari |
| **File Besar Tidak Terpakai** | Ukuran ≥ 100 MB & tidak disentuh ≥ 365 hari | **55%** | REVIEW | 90 hari |

### Label

| Label | Arti | Perlakuan GUI/CLI `--yes` |
|-------|------|---------------------------|
| **AMAN** | Confidence ≥ 90% | Tercentang otomatis / otomatis diproses di `--yes` |
| **REVIEW** | Confidence < 90% | Tidak tercentang otomatis; perlu review manual |

## 6.2 File yang TIDAK PERNAH Ditawarkan untuk Dihapus

- **Kode sumber:** `.go` `.rs` `.py` `.js` `.ts` `.java` `.c` `.cpp` `.php` `.sql` `.sh` dll.
- **Dokumen:** `.pdf` `.docx` `.xlsx` `.pptx` `.txt` `.md` `.epub` dll.
- **Media pribadi:** `.jpg` `.png` `.mp4` `.mkv` `.mp3` `.wav` `.flac` dll.
- **Data pribadi:** `.vcf` `.ics` `.eml` `.pst`.
- **Database/proyek:** `.db` `.sqlite` `.dwg` `.psd` `.ai`.
- **Dotfiles:** `.gitignore` `.gitkeep` `.env`.

## 6.3 Ambang Default

| Parameter | Nilai | Arti |
|-----------|-------|------|
| `important_age_days` | 30 | Diubah < 30 hari → dianggap dipakai |
| `log_age_days` | 90 | Log basi setelah 90 hari |
| `installer_age_days` | 90 | Installer lama setelah 90 hari |
| `old_file_days` | 365 | File besar "tidak terpakai" = 1 tahun |
| `large_file_bytes` | 100 MB | Ambang "file besar" |
| `dedup_min_size` | 1 MB | File < 1 MB tak di-hash |
| `hash_threshold_bytes` | 100 MB | Hash prefix hanya utk file raksasa |

---

# Bagian 7 — Mekanisme Keamanan

Pertahanan berlapis (defense in depth), berlaku di GUI & CLI:

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

> Prinsip inti: **file tidak pernah langsung dihancurkan** pada tahap pertama —
> selalu melewati karantina yang dapat dipulihkan.

---

# Bagian 8 — Quarantine & Masa Retensi

## 8.1 Lokasi & Struktur

```
~/.storage-optimizer/
├── manifest.json                  ← daftar semua file karantina
├── quarantine/
│   └── 2026-09-11/                ← folder per tanggal
│       └── 20260911_203253-97d138a9_app.log
├── logs/
│   └── audit-2026-09-11.jsonl     ← audit hari ini (rotasi harian)
└── reports/
    ├── scan-20260911-203253.json
    └── last-scan-report.json
```

## 8.2 Isi Manifest (per file)

```json
{
  "id": "20260911_203253-97d138a9",
  "original_path": "D:\\Data\\logs\\app.log",
  "quarantine_path": "C:\\Users\\...\\quarantine\\2026-09-11\\20260911_203253-97d138a9_app.log",
  "size": 475325,
  "sha256": "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
  "moved_at": "2026-09-11T20:32:53+07:00",
  "retention_until": "2026-10-11T20:32:53+07:00",
  "category": "Log Files",
  "label": "AMAN"
}
```

## 8.3 Cara Mengelola (GUI vs CLI)

| Aksi | GUI (Bersihin.exe) | CLI |
|------|--------------------|-----|
| Lihat isi | Halaman Quarantine (tabel) | `storage-optimizer restore` |
| Pulihkan satu | Centang → Pulihkan Terpilih / tombol per baris | `restore <id>` |
| Pulihkan semua | (centang semua → Pulihkan Terpilih) | `restore all` |
| Hapus terpilih (segera) | Centang → Hapus Terpilih | `purge <id>` |
| Hapus expired | Hapus Semua Expired | `purge` |
| Hapus semua (segera) | — | `purge --now` |

---

# Bagian 9 — Konfigurasi

Dimuat dari `config/default.yaml` (bawaan) atau file kustom via flag `--config`.

```yaml
app:
  data_dir: ""            # default: ~/.storage-optimizer
  retention_days: 30      # retensi default

scan:
  include_external: false # jangan scan drive eksternal
  worker_count: 0         # 0 = otomatis (max 8)
  hash_threshold_bytes: 104857600   # 100 MB
  dedup_min_size: 1048576           # 1 MB

exclude:
  windows:
    - C:\Windows
    - C:\Program Files
    - C:\ProgramData
    - C:\$Recycle.Bin
    - C:\Users\*\AppData\Local\Temp
  custom: []              # tambahkan folder lain di sini

retention:
  cache_temp_days: 14
  log_duplicate_days: 30
  large_old_days: 90

thresholds:
  important_age_days: 30
  large_file_bytes: 104857600
  old_file_days: 365
  installer_age_days: 90
  log_age_days: 90
```

**CLI:** `storage-optimizer config` untuk melihat, `config --generate` untuk membuat file.
**GUI:** menggunakan konfigurasi default yang sama.

---

# Bagian 10 — Struktur Data & Penyimpanan

## 10.1 Tipe Data Inti

```
FileMeta    → path, size, mod_time, access_time, extension, is_dir
Candidate   → Meta + Label + Category + Confidence + Reason + DupGroup
ScanReport  → GeneratedAt, Duration, Roots, FilesScanned, TotalBytes,
              DirsScanned, Candidates[], Errors[]
Manifest    → ID, OriginalPath, QuarantinePath, Size, SHA256,
              MovedAt, RetentionUntil, Category, Label
```

## 10.2 Lokasi Penyimpanan

| Data | Lokasi |
|------|--------|
| Data aplikasi | `C:\Users\<nama>\.storage-optimizer\` |
| File karantina | `...\quarantine\<YYYY-MM-DD>\` |
| Manifest | `...\manifest.json` |
| Audit trail | `...\logs\audit-<YYYY-MM-DD>.jsonl` |
| Laporan scan | `...\reports\scan-<timestamp>.json` |
| Laporan terakhir | `...\reports\last-scan-report.json` |

## 10.3 Konsistensi & Atomicity

- Manifest ditulis **atomik** (tulis `.tmp` → rename) agar tak korup jika mati mendadak.
- Pindah file antar-drive: **copy → delete sumber → rename** + pembersihan `.part` jika gagal.
- ID karantina berbasis timestamp + hash → **tidak pernah bertabrakan**.

---

# Bagian 11 — Audit Trail

Ditulis ke `logs/audit-YYYY-MM-DD.jsonl` (satu JSON per baris, rotasi harian).

```json
{"timestamp":"2026-09-11T20:32:53Z","event":"quarantine_move","level":"info",
 "message":"pindahkan ke quarantine",
 "data":{"id":"20260911_203253-97d138a9","path":"D:\\Data\\logs\\app.log",
         "quarantine":"...\\quarantine\\2026-09-11\\20260911_203253-97d138a9_app.log",
         "size":475325,"retention_until":"2026-10-11"}}
```

| Event | Kapan | Level |
|-------|-------|-------|
| `scan_started` | Pemindaian dimulai | info |
| `scan_complete` | Pemindaian selesai | info |
| `quarantine_move` | File ke karantina | info |
| `restore` | File dipulihkan | info |
| `purge` | File dihapus permanen | info |
| `validation_skip` | File dilewati validasi | warn |
| `error` | Terjadi kesalahan | error |

---

# Bagian 12 — Kepatuhan Standar

| Standar | Penerapan |
|---------|-----------|
| **NIST SP 800-88 Rev. 1** | Sanitasi media berjenjang (karantina → hapus), hindari penghapusan langsung |
| **NIST SP 800-12 (Ch. 18)** | Desain audit trail; peristiwa penting tercatat append-only |
| **ISO/IEC 27001:2022** (A.8.12, A.8.15) | Manajemen siklus hidup aset: penanganan/penghapusan dengan kontrol dan jejak |
| **POSIX.1-2017 / IEEE 1003.1** | Perilaku berkas standar (rename, copy, hardlink) lintas platform |
| **Le Coz et al. (2024, NeurIPS)** | Confidence Scores untuk kalibrasi keputusan |
| **Tang et al. (2021, IEEE/ACM)** | Desain I/O asinkron paralel untuk scan cepat |
| **Ratliff et al. (2023, IEEE S&P)** | Konsistensi crash — atomic write + recovery |
| **El-Shimi et al. (2012, USENIX FAST)** | Desain deduplikasi berbasis hash konten |
| **Bach et al. (2023, IEEE)** | Pola dashboard interaktif: kelompok → detail → konfirmasi |

---

# Lampiran

## Lampiran A — Flowchart Ringkas

```
PILIH FOLDER/DRIVE
        │
        ▼
   PEMINDAIAN  ───► Progress Bar
        │
        ▼
  DETEKSI DUPLIKAT (SHA-256)
        │
        ▼
 SCAN SELESAI → SORTIR KE 5 KATEGORI
        │
        ▼
  DASHBOARD (checkbox per kategori/file)
        │
        ▼
 APPROVE? ──TIDAK──► SELESAI (tidak ada yang berubah)
        │
       YA
        │
        ▼
 VALIDASI 7 LANGKAH
        │
   ┌────┴────────────┐
   │ lulus           │ gagal → SKIP + catat audit
   ▼                 │
 KARANTINA  ─────────┘
 (manifest + SHA-256)
        │
   ┌────┴────────────┐
   │                 │
   ▼                 ▼
 PULIHKAN        HAPUS PERMANEN
 (kapan saja)    (purge / expired / --now)
```

## Lampiran B — Perbandingan Cepat GUI vs CLI

| Aspek | GUI (Bersihin.exe) | CLI (storage-optimizer) |
|-------|--------------------|--------------------------|
| Menjalankan | Double-click | `storage-optimizer <command>` di PowerShell/CMD |
| Cocok untuk | Pengguna awam | Pengguna teknis |
| Pilih sasaran | Dropdown + 📁 Cari + ketik jalur | Flag `--mode` + `--path` |
| Selama scan | Progress bar visual | Persentase/hitungan |
| Pilih file | Checkbox mouse | Keyboard (space) |
| Auto-approve | — | `--yes` (confidence ≥90%) |
| Hapus segera | Hapus Terpilih (checklist) | `purge <id>` / `purge --now` |
| Data & backend | Sama persis dengan CLI | Sama persis dengan GUI |

## Lampiran C — Glossary

| Istilah | Arti |
|---------|------|
| **Kandidat** | File yang memenuhi kriteria layak dibersihkan |
| **Quarantine** | Area aman tempat file ditahan sebelum dihapus permanen |
| **Retensi** | Jangka waktu file boleh dipulihkan sebelum dihapus |
| **Confidence** | Keyakinan sistem (0–1) bahwa file aman dihapus |
| **Manifest** | Catatan terstruktur semua file dalam karantina |
| **Audit trail** | Catatan kronologis semua aksi sistem |
| **Dry-run** | Simulasi scan tanpa melakukan perubahan apa pun |
| **Purge** | Menghapus file karantina secara permanen |
| **Restore** | Memindahkan file kembali dari karantina ke lokasi asli |

---

*Dokumen ini dapat diperbarui kapan pun saat implementasi berkembang.*
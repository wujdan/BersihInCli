# YukBersihIn — Storage Optimizer

> Bersihkan penyimpanan secara aman: scan duplikat, hapus cache/log lama,
> karantina dulu sebelum hapus permanen.

![Go](https://img.shields.io/badge/Go-1.27-blue)
![Platform](https://img.shields.io/badge/Windows%20%7C%20Linux%20%7C%20macOS-lightgrey)
![GitHub License](https://img.shields.io/github/license/wujdan/BersihInCli)

## Fitur Utama

- **Scan penyimpanan multi-drive** - mode full (semua drive), quick (folder umum), custom (folder spesifik)
- **Deteksi file duplikat** - via hash SHA-256
- **Klasifikasi otomatis + confidence score** - cache, temp, log, installer, file besar, sisa aplikasi terhapus
- **Auto-clean cache/temp/log** - dipindah ke karantina, bukan hapus langsung
- **Dashboard TUI interaktif** - pilih per file atau per kategori
- **Karantina wajib** - file bisa dipulihkan kapan saja
- **Dry-run mode** - preview sebelum eksekusi
- **Export laporan** - ke CSV / TSV / JSON
- **Auto-purge daemon** - hapus file lewat retensi secara berkala

## Instalasi

```bash
# Clone & build
git clone https://github.com/wujdan/BersihInCli.git
cd BersihInCli
go build -o storage-optimizer.exe .

# Atau langsung tanpa build
go run . scan --mode=full
```

### Yang harus di-install dulu

| Kebutuhan | Versi | Untuk apa |
|-----------|-------|-----------|
| [Go](https://go.dev/dl/) | **1.27 atau lebih baru** | Meng-compile & menjalankan program `storage-optimizer` |
| [Git](https://git-scm.com/downloads) | Versi terbaru | Meng-unduh (`clone`) kode dari GitHub |
| Terminal (PowerShell / CMD) | — | Menjalankan perintah; wajib mendukung ANSI/UTF-8 untuk tampilan TUI |

> **Catatan platform:**
> - **Windows** (target utama) — deteksi kategori *Sisa Aplikasi Terhapus* hanya aktif di sini (membaca registry aplikasi terpasang), begitu pula pencarian drive lokal.
> - **Linux / macOS** — program tetap bisa di-build & dipakai; hanya fitur orphan yang nonaktif (aman, tanpa deteksi palsu).
> - Go tidak wajib setelah binary di-build — tetapi karena perintah `uninstall` tidak menghapus binary, pastikan `storage-optimizer.exe` berada di folder yang Anda ingat.

## Penggunaan (Usage)

| Command            | Contoh                                           | Keterangan                              |
|--------------------|--------------------------------------------------|-----------------------------------------|
| `scan`             | `storage-optimizer scan --mode=full`             | Scan semua drive lokal                  |
|                    | `storage-optimizer scan --mode=quick`            | Scan folder umum (Downloads, Temp)      |
|                    | `storage-optimizer scan --mode=custom --path=D:\Data` | Scan folder spesifik            |
| `review`           | `storage-optimizer review`                       | Lihat hasil scan terakhir               |
| `restore`          | `storage-optimizer restore`                      | Lihat file di karantina                 |
|                    | `storage-optimizer restore <id>`                 | Pulihkan satu file                      |
|                    | `storage-optimizer restore all`                  | Pulihkan semua                          |
| `purge`            | `storage-optimizer purge`                        | Hapus file yang lewat retensi           |
|                    | `storage-optimizer purge --now`                  | Hapus semua sekarang (hati-hati!)       |
| `config`           | `storage-optimizer config`                       | Tampilkan konfigurasi aktif             |
| `uninstall`        | `storage-optimizer uninstall`                    | Hapus semua data aplikasi lewat CLI     |

### Flag Penting

| Flag                 | Keterangan                                  | Default |
|----------------------|---------------------------------------------|---------|
| `--mode`             | `full` / `quick` / `custom`                 | `full`  |
| `--path`             | Path target (wajib untuk `custom`)           | —       |
| `--dry-run`          | Simulasi, tanpa hapus apa pun                | `false` |
| `-y, --yes`          | Non-interaktif, auto-approve                 | `false` |
| `--min-confidence`   | Ambang confidence untuk auto-pilih           | `0.9`   |
| `--export`           | Ekspor ke CSV / TSV / JSON                   | —       |
| `--every`            | Auto-purge daemon (mis. `--every=24h`)        | —       |
| `--force`            | Lewati konfirmasi (untuk skrip)              | `false` |

## Uninstal

Ada dua cara; keduanya **tidak menghapus binary** — hapus manual setelahnya.

### 1. Lewat CLI (disarankan)

```bash
# Hapus semua data aplikasi: file karantina, manifest, laporan, log audit
storage-optimizer uninstall

# Tanpa konfirmasi (untuk skrip/automasi)
storage-optimizer uninstall --force
```

Yang terhapus:
- `%USERPROFILE%\.storage-optimizer\quarantine\` — **semua file karantina (permanen, tidak bisa dipulihkan)**
- `manifest.json`, `reports\*`, `logs\*`

Lalu hapus binary & kodenya:

```bash
del storage-optimizer.exe        # dari folder build
Remove-Item BersihInCli -Recurse -Force   # hapus folder clone
```

### 2. Manual (tanpa CLI)

```powershell
Remove-Item "$env:USERPROFILE\.storage-optimizer" -Recurse -Force
del storage-optimizer.exe
```

### Contoh Output

```text
SISTEM PEMBERSIHAN PENYIMPANAN
  Cakupan scan: D:\Data

  HASIL PEMINDAIAN
  --------------------------------------------------------------------------
  File dipindai : 12,453
  Direktori     : 1,208
  Total ukuran  : 34.2 GB
  Kandidat hapus: 328 file (1.8 GB)
  Waktu         : 1m 42s
  Kapasitas     : 87.3 GB tersisa dari 195.3 GB
  --------------------------------------------------------------------------
```

## Konfigurasi

File default: `config/default.yaml`. Contoh:

```yaml
retention:
  cache_temp_days: 14
  log_duplicate_days: 30
  large_old_days: 90

scan:
  include_external: false
  hash_threshold_bytes: 104857600  # 100 MB
  dedup_min_size: 1048576          # 1 MB
```

## ⚠ Peringatan Penting

1. **Selalu gunakan `--dry-run`** terlebih dahulu sebelum menghapus apa pun.
2. File di karantina **bisa dipulihkan**; file yang sudah di-purge **tidak bisa**.
3. `purge --now` menghapus semua file di karantina secara permanen.
4. Klasifikasi berbasis heuristic — review tetap disarankan.

## Roadmap

- [ ] Progress bar scan yang lebih detail
- [ ] Notifikasi desktop / email
- [ ] Mode kompresi file besar

## Kontribusi

```bash
go build ./...
go vet ./...
go test ./...
```

Pull request dan issue diterima di GitHub. Dokumentasi lengkap: `Dokumentasi-Alur-Sistem.md`.

## Lisensi

[MIT](LICENSE) © 2026 Danss
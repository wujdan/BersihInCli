package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"storage-optimizer/internal/models"
	"storage-optimizer/internal/quarantine"
)

func newPurgeCmd() *cobra.Command {
	now := false
	force := false
	every := time.Duration(0)
	cmd := &cobra.Command{
		Use:   "purge [file-id...]",
		Short: "Menghapus permanen isi quarantine",
		Long: `Menghapus permanen file yang sudah melewati masa retensi di quarantine.
File dalam masa retensi TIDAK akan dihapus kecuali dengan flag --now.
Setiap penghapusan permanen wajib dikonfirmasi dengan mengetik 'yes'
(lewati dengan flag --force untuk skrip/automasi).
Mode daemon --every memeriksa file kedaluwarsa secara berkala (auto-purge).

Contoh:
  storage-optimizer purge               # hapus permanen file yang sudah melewati retensi
  storage-optimizer purge --now         # hapus permanen semua file segera
  storage-optimizer purge <file-id>     # hapus permanen satu file tertentu
  storage-optimizer purge --every=24h   # auto-purge: periksa tiap 24 jam`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			q := application.Quarantine()

			ids := args
			if now {
				ids = nil // purge everything
			}

			if every > 0 {
				return runPurgeDaemon(ctx, q, every)
			}

			// Documented safety: permanent deletion always asks for explicit
			// confirmation ('yes') unless --force is set.
			if !force && !confirmPurge(ids, now) {
				fmt.Println("  ✘ Dibatalkan. Tidak ada file yang dihapus.")
				return nil
			}

			count, freed, err := q.PurgeExpired(ids, now)
			if err != nil {
				return err
			}
			application.Logger().Info("purge", "purge quarantine", map[string]interface{}{
				"count": count, "freed": freed, "forced": now,
			})
			if count == 0 {
				fmt.Println("  ℹ Tidak ada file yang memenuhi syarat untuk dihapus permanen.")
				fmt.Println("    Gunakan --now untuk menghapus semua isi quarantine segera.")
				return nil
			}
			fmt.Printf("  ✔ %d file dihapus permanen (%s)\n", count, models.HumanBytes(freed))
			return nil
		},
	}
	cmd.Flags().BoolVar(&now, "now", false, "hapus permanen SEMUA isi quarantine segera")
	cmd.Flags().BoolVar(&force, "force", false, "lewati konfirmasi 'yes' (untuk skrip/automasi)")
	cmd.Flags().DurationVar(&every, "every", 0, "auto-purge: jalankan terus-menerus dan periksa tiap jangka waktu (mis. 24h)")
	return cmd
}

// runPurgeDaemon periodically purges expired quarantine entries (tahap 12).
func runPurgeDaemon(ctx context.Context, q *quarantine.Dir, interval time.Duration) error {
	fmt.Printf("  Auto-purge aktif: memeriksa file kedaluwarsa setiap %s (Ctrl+C untuk berhenti)\n", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		count, freed, err := q.PurgeExpired(nil, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✘ Auto-purge gagal: %v\n", err)
		} else if count > 0 {
			fmt.Printf("  ✔ %d file melewati retensi dihapus permanen (%s)\n",
				count, models.HumanBytes(freed))
			application.Logger().Info("purge", "auto-purge", map[string]interface{}{
				"count": count, "freed": freed,
			})
		}
		select {
		case <-ctx.Done():
			fmt.Println("\n  Auto-purge dihentikan.")
			return nil
		case <-ticker.C:
		}
	}
}

// confirmPurge asks the user to explicitly approve permanent deletion.
func confirmPurge(ids []string, now bool) bool {
	fmt.Println()
	fmt.Println("  PERINGATAN: tindakan ini menghapus file secara PERMANEN dari karantina.")
	fmt.Println("  File yang dihapus tidak dapat dipulihkan lagi.")
	if now {
		fmt.Println("  Sasaran  : SEMUA isi quarantine")
	} else if len(ids) > 0 {
		fmt.Printf("  Sasaran  : %d file tertentu\n", len(ids))
	} else {
		fmt.Println("  Sasaran  : file yang sudah lewat masa retensi")
	}
	fmt.Print("  Ketik 'yes' untuk melanjutkan, 'no' untuk membatalkan: ")

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	ans := strings.ToLower(strings.TrimSpace(line))
	return ans == "yes" || ans == "ya" || ans == "y"
}
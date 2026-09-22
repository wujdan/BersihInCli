package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"storage-optimizer/internal/models"
)

// newUninstallCmd removes all application data (quarantine, manifest,
// reports, logs) directly from the CLI. The binary itself is not deleted.
func newUninstallCmd() *cobra.Command {
	force := false
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Menghapus semua data aplikasi (uninstal lewat CLI)",
		Long: `Menghapus seluruh data aplikasi Storage Optimizer dari komputer:
  - file karantina (quarantine/) + manifest.json
  - laporan scan (reports/)
  - log audit (logs/)

File yang masih di karantina akan dihapus PERMANEN dan tidak dapat
dipulihkan lagi. Perintah ini TIDAK menghapus binary/klon — untuk itu hapus
manual (mis. ` + "`del storage-optimizer.exe`" + `).

Contoh:
  storage-optimizer uninstall          # hapus semua data aplikasi (konfirmasi 'yes')
  storage-optimizer uninstall --force  # tanpa konfirmasi (skrip/automasi)`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			application.Logger().Info("uninstall", "uninstall aplikasi", map[string]interface{}{
				"data_dir": application.DataDir(),
			})
			application.Close()

			list := application.Quarantine().List()
			var bytes int64
			for _, m := range list {
				bytes += m.Size
			}

			fmt.Println()
			fmt.Println("  MENGHAPUS SEMUA DATA APLIKASI")
			fmt.Printf("  Data dir  : %s\n", application.DataDir())
			fmt.Printf("  Karantina : %d file (%s) — akan dihapus PERMANEN\n",
				len(list), models.HumanBytes(bytes))
			if !force && !confirmUninstall() {
				fmt.Println("  ✘ Dibatalkan. Tidak ada yang dihapus.")
				return nil
			}

			if err := os.RemoveAll(application.DataDir()); err != nil {
				return fmt.Errorf("hapus data dir: %w", err)
			}
			fmt.Printf("  ✔ Data aplikasi dihapus: %s\n", application.DataDir())
			fmt.Println("  ℹ Binary storage-optimizer masih ada. Hapus manual jika diinginkan:")
			fmt.Println("      del storage-optimizer.exe")
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "lewati konfirmasi 'yes' (untuk skrip/automasi)")
	return cmd
}

// confirmUninstall asks the user to explicitly approve permanent data removal.
func confirmUninstall() bool {
	fmt.Println()
	fmt.Println("  PERINGATAN: semua file karantina akan dihapus PERMANEN.")
	fmt.Println("  Laporan scan dan log audit juga ikut terhapus.")
	fmt.Print("  Ketik 'yes' untuk melanjutkan, 'no' untuk membatalkan: ")

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	ans := strings.ToLower(strings.TrimSpace(line))
	return ans == "yes" || ans == "ya" || ans == "y"
}
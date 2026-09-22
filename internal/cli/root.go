package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"storage-optimizer/internal/app"
	"storage-optimizer/internal/config"
)

var (
	cfgPath     string
	cfg         *config.Config
	application *app.App
)

// Name of the binary.
const Name = "storage-optimizer"

// NewRootCmd builds the root command hierarchy.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     Name,
		Short:   "Optimasi penyimpanan laptop/komputer (CLI)",
		Long: `Storage Optimizer memindai drive, mengidentifikasi file yang berpotensi
tidak penting (cache, temp, log, duplikat, installer bekas, file besar lama),
lalu memberi kontrol penuh kepada user untuk memilih dan menyetujui file
yang akan dipindahkan ke karantina sebelum dihapus permanen.

Seluruh proses tercatat dalam audit trail dan file bisa dipulihkan selama
masa retensi.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			c, err := config.Load(cfgPath)
			if err != nil {
				return err
			}
			cfg = c
			a, err := app.New(cfg)
			if err != nil {
				return err
			}
			application = a
			return nil
		},
	}

	root.PersistentFlags().StringVar(&cfgPath, "config", "", "path ke file konfigurasi YAML (default: config/default.yaml)")

	root.AddCommand(
		newScanCmd(),
		newReviewCmd(),
		newRestoreCmd(),
		newPurgeCmd(),
		newConfigCmd(),
		newUninstallCmd(),
	)

	// graceful exit context
	return root
}

// Execute runs the CLI and returns the exit code.
func Execute() int {
	root := NewRootCmd()
	root.SilenceUsage = true
	root.SilenceErrors = true

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "✘", err)
		return categorizeExit(err)
	}
	return 0
}

// categorizeExit maps errors to documented exit codes:
//
//	0   sukses
//	1   kesalahan tidak terduga
//	2   kesalahan pemakaian (flag/path)
//	130   dibatalkan user (Ctrl+C / SIGTERM)
func categorizeExit(err error) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, context.Canceled) {
		return 130
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unknown flag") ||
		strings.Contains(msg, "flag needs an argument") ||
		strings.Contains(msg, "mode tidak dikenal") ||
		strings.Contains(msg, "mode custom membutuhkan") ||
		strings.Contains(msg, "tidak ada drive") {
		return 2
	}
	return 1
}
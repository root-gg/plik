package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
	"github.com/root-gg/plik/server/metadata"
	"github.com/root-gg/plik/server/server"
)

type migrateFlagParams struct {
	to           string
	dataOnly     bool
	metadataOnly bool
	ignoreErrors bool
	workers      int
	dryRun       bool
}

var migrateParams = migrateFlagParams{}

// migrateCmd migrates metadata and/or data from one backend to another
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate metadata and/or data backend",
	Long: `Migrate metadata (users, tokens, uploads, files, settings) and file data
from the configured source backends to the backends configured in the target config file.

Both the source plikd.cfg (--config flag or default location) and the target plikd.cfg
(--to flag) are required. The target config only needs to specify the backend fields
that differ from the source.

Typical use cases:
  - SQLite → PostgreSQL (metadata only):  plikd migrate --to new.cfg --metadata-only
  - Local files → S3 (data only):        plikd migrate --to new.cfg --data-only
  - Full migration:                       plikd migrate --to new.cfg`,
	Example: `  plikd migrate --to /etc/plikd-new.cfg
  plikd migrate --to /etc/plikd-new.cfg --metadata-only
  plikd migrate --to /etc/plikd-new.cfg --data-only --workers 8
  plikd migrate --to /etc/plikd-new.cfg --dry-run
  plikd migrate --to /etc/plikd-new.cfg --ignore-errors`,
	Run: migrateBackends,
}

func init() {
	migrateCmd.Flags().StringVar(&migrateParams.to, "to", "", "path to target plikd.cfg (required)")
	migrateCmd.Flags().BoolVar(&migrateParams.dataOnly, "data-only", false, "migrate file data only (skip metadata)")
	migrateCmd.Flags().BoolVar(&migrateParams.metadataOnly, "metadata-only", false, "migrate metadata only (skip file data)")
	migrateCmd.Flags().BoolVar(&migrateParams.ignoreErrors, "ignore-errors", false, "continue on individual record/file errors")
	migrateCmd.Flags().IntVar(&migrateParams.workers, "workers", 4, "number of parallel file copy workers")
	migrateCmd.Flags().BoolVar(&migrateParams.dryRun, "dry-run", false, "print what would be migrated without writing anything")
	_ = migrateCmd.MarkFlagRequired("to")
	rootCmd.AddCommand(migrateCmd)
}

func migrateBackends(cmd *cobra.Command, args []string) {
	if migrateParams.dataOnly && migrateParams.metadataOnly {
		fmt.Println("--data-only and --metadata-only are mutually exclusive")
		os.Exit(1)
	}

	start := time.Now()

	if migrateParams.dryRun {
		fmt.Println("[dry-run mode — nothing will be written]")
	}

	// Load target config
	targetConfig, err := common.LoadConfiguration(migrateParams.to)
	if err != nil {
		fmt.Printf("unable to load target config %s: %s\n", migrateParams.to, err)
		os.Exit(1)
	}

	// Initialize source backends (from primary config, already loaded by initConfig)
	initializeMetadataBackend()
	srcMeta := metadataBackend

	// Initialize target metadata backend
	var dstMeta *metadata.Backend
	if !migrateParams.dataOnly {
		dstMeta, err = server.NewMetadataBackend(targetConfig.MetadataBackendConfig, targetConfig.NewLogger())
		if err != nil {
			fmt.Printf("unable to initialize target metadata backend: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("Migrating metadata: %s → %s\n",
			config.MetadataBackendConfig["Driver"],
			targetConfig.MetadataBackendConfig["Driver"])

		if !migrateParams.dryRun {
			metaOpts := &metadata.MigrateOptions{
				IgnoreErrors: migrateParams.ignoreErrors,
			}
			metaStats, err := metadata.Migrate(srcMeta, dstMeta, metaOpts)
			if err != nil {
				fmt.Printf("metadata migration failed: %s\n", err)
				os.Exit(1)
			}
			totalMetaErrors := metaStats.UserErrors + metaStats.TokenErrors +
				metaStats.UploadErrors + metaStats.FileErrors + metaStats.SettingErrors
			fmt.Printf("metadata migration complete: %d users, %d tokens, %d uploads, %d files, %d settings (%d errors)\n",
				metaStats.Users, metaStats.Tokens, metaStats.Uploads, metaStats.Files, metaStats.Settings,
				totalMetaErrors)
		} else {
			// Dry-run: enumerate and print all metadata items that would be migrated
			fmt.Println("[dry-run] would migrate:")
			var dryRunErr error
			dryRunErr = srcMeta.ForEachUsers(func(u *common.User) error {
				fmt.Printf("  user     %s (%s)\n", u.ID, u.Login)
				return nil
			})
			if dryRunErr != nil {
				fmt.Printf("error enumerating users: %s\n", dryRunErr)
				os.Exit(1)
			}
			dryRunErr = srcMeta.ForEachToken(func(t *common.Token) error {
				fmt.Printf("  token    %s (user: %s)\n", t.Token, t.UserID)
				return nil
			})
			if dryRunErr != nil {
				fmt.Printf("error enumerating tokens: %s\n", dryRunErr)
				os.Exit(1)
			}
			dryRunErr = srcMeta.ForEachUploadUnscoped(func(u *common.Upload) error {
				fmt.Printf("  upload   %s\n", u.ID)
				return nil
			})
			if dryRunErr != nil {
				fmt.Printf("error enumerating uploads: %s\n", dryRunErr)
				os.Exit(1)
			}
			dryRunErr = srcMeta.ForEachFile(func(f *common.File) error {
				fmt.Printf("  file     %s (%s, %s)\n", f.ID, f.Name, f.Status)
				return nil
			})
			if dryRunErr != nil {
				fmt.Printf("error enumerating files: %s\n", dryRunErr)
				os.Exit(1)
			}
			dryRunErr = srcMeta.ForEachSetting(func(s *common.Setting) error {
				fmt.Printf("  setting  %s\n", s.Key)
				return nil
			})
			if dryRunErr != nil {
				fmt.Printf("error enumerating settings: %s\n", dryRunErr)
				os.Exit(1)
			}
		}
	}

	// Migrate data
	if !migrateParams.metadataOnly {
		initializeDataBackend()
		srcData := dataBackend

		dstDataBackend, err := server.NewDataBackend(targetConfig.DataBackend, targetConfig.DataBackendConfig)
		if err != nil {
			fmt.Printf("unable to initialize target data backend: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("Migrating file data: %s → %s\n", config.DataBackend, targetConfig.DataBackend)

		dataOpts := &data.MigrateOptions{
			IgnoreErrors: migrateParams.ignoreErrors,
			Workers:      migrateParams.workers,
			DryRun:       migrateParams.dryRun,
		}

		// Use source metadata to enumerate files (even if we just migrated metadata,
		// the source metadata backend is always the canonical file list)
		dataStats, err := data.MigrateFiles(srcData, dstDataBackend, srcMeta, dataOpts)
		if err != nil {
			fmt.Printf("data migration failed: %s\n", err)
			os.Exit(1)
		}

		prefix := ""
		if migrateParams.dryRun {
			prefix = "[dry-run] "
		}
		fmt.Printf("%sdata migration complete: %d files copied, %d skipped, %d errors, %d bytes\n",
			prefix, dataStats.Copied, dataStats.Skipped, dataStats.Errors, dataStats.Bytes)
	}

	fmt.Printf("migration completed in %s\n", time.Since(start).Round(time.Millisecond))
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/borg0ai/dx-issuegraph/internal/config"
	"github.com/borg0ai/dx-issuegraph/internal/metrics"
	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Back up your issuegraph database",
	Long: `Back up your issuegraph database for off-machine recovery.

This is a Dolt-native database backup. It preserves the database state,
including tables, branches, commit history, and working-set data. This is
different from 'issuegraph export', which writes issue records to JSONL for migration
and interoperability.

Commands:
  issuegraph backup init <path>    Set up a backup destination (filesystem or DoltHub)
  issuegraph backup sync           Push to configured backup destination
  issuegraph backup restore [path] Restore from a backup directory
  issuegraph backup remove         Remove backup destination
  issuegraph backup status         Show backup status

DoltHub is recommended for cloud backup:
  issuegraph backup init https://doltremoteapi.dolthub.com/<user>/<repo>
  Set DOLT_REMOTE_USER and DOLT_REMOTE_PASSWORD for authentication.

Auto-backup default:
  When backup.enabled is unset, auto-backup turns ON in embedded mode if a
  git remote exists, and stays OFF in sql-server / shared-server mode. In
  server mode many issuegraph clients share one Dolt server, and each would register
  a server-side backup remote under the same name pointing at its own local
  dir and full-sync the whole database — a self-amplifying storm. To back up
  a shared server, run 'issuegraph backup' explicitly (or set backup.enabled=true and
  coordinate destinations). 'issuegraph config get backup.enabled' shows the effective
  value and its source.

  On a proxied-server workspace that opt-in does not apply: auto-backup has
  no post-command hook on that path, so backup.enabled=true is inert there.
  The five backup verbs above are honored on a managed-local proxied server —
  run 'issuegraph backup sync' explicitly to back one up.`,
	GroupID: "sync",
}

type backupSizeFunc func(context.Context) (bytes int64, available bool, err error)

var backupStatusCmd = newBackupStatusCommand(doltBackupSize)

func newBackupStatusCommand(sizeDatabase backupSizeFunc) *cobra.Command {
	return &cobra.Command{
		Use:           "status",
		Short:         "Show last backup status",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if usesProxiedServer() {
				if err := requireLocalProxiedBackup("backup status"); err != nil {
					return err
				}
			}
			evt := metrics.NewCommandEvent("backup-status")
			defer func() {
				if c := metrics.Global(); c != nil {
					c.CloseEventAndAdd(evt)
				}
			}()

			dir, err := backupDir()
			if err != nil {
				return err
			}

			state, err := loadBackupState(dir)
			if err != nil {
				return err
			}
			databaseSize, sizeAvailable, err := sizeDatabase(cmd.Context())
			if err != nil {
				return HandleErrorRespectJSON("measure database size: %v", err)
			}

			if jsonOutput {
				result := map[string]interface{}{
					"backup": state,
					"dolt":   showDoltBackupStatusJSON(),
				}
				if sizeAvailable {
					result["database_size"] = showDBSizeJSON(databaseSize)
				}
				data, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil
			}

			hasBackup := state.LastDoltCommit != ""
			hasDolt := false
			if cfg, _ := loadDoltBackupConfig(); cfg != nil {
				hasDolt = true
			}

			if !hasBackup && !hasDolt {
				fmt.Println("No backup has been performed yet.")
				fmt.Println()
				fmt.Println("Setup:")
				fmt.Println("  issuegraph backup init <path>    Set up a backup destination")
				fmt.Println("  issuegraph backup sync           Push to backup destination")
				if sizeAvailable {
					showDBSize(databaseSize)
				}
				return nil
			}

			if hasBackup {
				fmt.Println("Backup:")
				fmt.Printf("  Last backup: %s (%s ago)\n",
					state.Timestamp.Format(time.RFC3339),
					time.Since(state.Timestamp).Round(time.Second))
				fmt.Printf("  Dolt commit: %s\n", state.LastDoltCommit)
			}

			// Show config (effective values with source)
			enabled := isBackupAutoEnabled()
			interval := config.GetDuration("backup.interval")
			enabledNote := ""
			if note := backupAutoStatusNote(enabled); note != "" {
				enabledNote = " (" + note + ")"
			}
			fmt.Printf("\nConfig: enabled=%v%s interval=%s\n",
				enabled, enabledNote, interval)

			// Show Dolt backup info
			showDoltBackupStatus()

			// Show database size when the active backend can measure it locally.
			if sizeAvailable {
				showDBSize(databaseSize)
			}

			return nil
		},
	}
}

func init() {
	backupCmd.AddCommand(backupStatusCmd)
	rootCmd.AddCommand(backupCmd)
}

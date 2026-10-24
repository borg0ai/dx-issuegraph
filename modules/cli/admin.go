package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// requireServerMode returns an error if issuegraph is running in embedded mode.
// Used by admin subcommands to guard against embedded-mode execution.
// This must be called inside RunE (after store init), not in PersistentPreRunE,
// because !usesSQLServer() reads cmdCtx.ServerMode which is only populated
// after main.go's PersistentPreRun completes (cobra runs child PreRunE before
// parent PreRun, so the check fires too early if placed on adminCmd itself).
func requireServerMode(cmdName string) error {
	if !usesSQLServer() {
		return fmt.Errorf("'issuegraph admin %s' is not yet supported in embedded mode", cmdName)
	}
	return nil
}

var adminCmd = &cobra.Command{
	Use:     "admin",
	GroupID: "advanced",
	Short:   "Administrative commands for database maintenance",
	Long: `Administrative commands for issuegraph database maintenance.

These commands are for advanced users and should be used carefully:
  cleanup   Delete closed issues (issue lifecycle)
  compact   Compact old closed issues to save space (storage optimization)
  reset     Remove all issuegraph data and configuration (full reset)

For routine maintenance, prefer 'issuegraph doctor --fix' which handles common repairs
automatically. Use these admin commands for targeted database operations.`,
}

func init() {
	rootCmd.AddCommand(adminCmd)
	adminCmd.AddCommand(cleanupCmd)
	adminCmd.AddCommand(compactCmd)
	adminCmd.AddCommand(resetCmd)
}

package fix

import (
	"fmt"
	"os"
	"strings"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
)

// DatabaseConfig auto-detects and fixes metadata.json database config mismatches.
// This fix only applies to SQLite backends where .db files on disk may not match
// the configured database name. Dolt backends store data on a server, so there
// are no local .db files to reconcile.
func DatabaseConfig(path string) error {
	issuegraphDir, err := resolvedWorkspaceIssueGraphDir(path)
	if err != nil {
		return err
	}

	// Load existing config
	cfg, err := configfile.Load(issuegraphDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if cfg == nil {
		// No config exists - nothing to fix
		return fmt.Errorf("no metadata.json found")
	}

	// Dolt backend stores data on the server — no local .db files to reconcile
	if cfg.GetBackend() == configfile.BackendDolt {
		return fmt.Errorf("database config fix not applicable for Dolt backend (data is on the server)")
	}

	fixed := false

	// Check if configured database name matches the actual .db file on disk
	actualDB := findActualDBFile(issuegraphDir)
	if actualDB != "" && cfg.Database != actualDB {
		fmt.Printf("  Updating database: %s → %s\n", cfg.Database, actualDB)
		cfg.Database = actualDB
		fixed = true
	}

	if !fixed {
		return fmt.Errorf("no configuration mismatches detected")
	}

	// Save updated config
	if err := cfg.Save(issuegraphDir); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// findActualDBFile scans .issuegraph/ for the actual SQLite database file in use.
// Only finds .db files; does not handle Dolt directories.
// Prefers issuegraph.db (canonical name), skips backups and vc.db.
func findActualDBFile(issuegraphDir string) string {
	entries, err := os.ReadDir(issuegraphDir)
	if err != nil {
		return ""
	}

	var candidates []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()

		// Must end with .db
		if !strings.HasSuffix(name, ".db") {
			continue
		}

		// Skip backups and vc.db
		if strings.Contains(name, "backup") || name == "vc.db" {
			continue
		}

		candidates = append(candidates, name)
	}

	if len(candidates) == 0 {
		return ""
	}

	// Prefer issuegraph.db (canonical name)
	for _, name := range candidates {
		if name == "issuegraph.db" {
			return name
		}
	}

	// Fall back to first candidate
	return candidates[0]
}

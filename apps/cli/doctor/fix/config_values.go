package fix

import (
	"fmt"
	"strings"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
)

// ConfigValues fixes invalid configuration values in metadata.json.
// Currently handles: database field pointing to SQLite name when backend is Dolt.
func ConfigValues(path string) error {
	issuegraphDir, err := resolvedWorkspaceIssueGraphDir(path)
	if err != nil {
		return err
	}

	cfg, err := configfile.Load(issuegraphDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if cfg == nil {
		return fmt.Errorf("no metadata.json found")
	}

	fixed := false

	// Fix database field: when backend is Dolt, database should be "dolt" not "issuegraph.db"
	if cfg.GetBackend() == configfile.BackendDolt {
		if strings.HasSuffix(cfg.Database, ".db") || strings.HasSuffix(cfg.Database, ".sqlite") || strings.HasSuffix(cfg.Database, ".sqlite3") {
			fmt.Printf("  Updating database: %q → %q (Dolt backend uses directory)\n", cfg.Database, "dolt")
			cfg.Database = "dolt"
			fixed = true
		}
	}

	if !fixed {
		fmt.Println("  → No configuration issues to fix")
		return nil
	}

	if err := cfg.Save(issuegraphDir); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

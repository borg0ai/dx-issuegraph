package fix

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
)

// DatabaseIntegrity attempts to recover from database corruption by:
//  1. Backing up the corrupt database
//  2. Re-initializing via issuegraph init (which will clone from remote if configured)
func DatabaseIntegrity(path string) error {
	issuegraphDir, err := resolvedWorkspaceIssueGraphDir(path)
	if err != nil {
		return err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	return doltIntegrityRecovery(absPath, issuegraphDir)
}

// doltIntegrityRecovery backs up the corrupted Dolt database and reinitializes.
func doltIntegrityRecovery(path, issuegraphDir string) error {
	if err := serverModeIntegrityRecoveryGuard(issuegraphDir); err != nil {
		return err
	}

	doltPath := getDatabasePath(issuegraphDir)

	// Check if dolt directory exists
	if _, err := os.Stat(doltPath); os.IsNotExist(err) {
		return fmt.Errorf("no Dolt database to recover at %s", doltPath)
	}

	// Back up corrupted dolt directory
	ts := time.Now().UTC().Format("20060102T150405Z")
	backupPath := doltPath + "." + ts + ".corrupt.backup"
	fmt.Printf("  Backing up corrupted Dolt database to %s\n", filepath.Base(backupPath))
	if err := os.Rename(doltPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup corrupted Dolt database: %w", err)
	}

	// Get issuegraph binary path
	issuegraphBinary, err := getBdBinary()
	if err != nil {
		// Restore corrupted database on failure
		_ = os.Rename(backupPath, doltPath)
		return err
	}

	// Reinitialize: issuegraph init --force -q
	// issuegraph init will clone from remote if sync.remote is configured.
	fmt.Printf("  Reinitializing Dolt database (will clone from remote if configured)\n")
	initCmd := newBdCmd(issuegraphBinary, "init", "--force", "-q", "--skip-hooks")
	initCmd.Dir = path
	initCmd.Stdout = os.Stdout
	initCmd.Stderr = os.Stderr

	if err := initCmd.Run(); err != nil {
		// Restore backup on failure
		fmt.Printf("  Warning: recovery failed, restoring corrupted Dolt database from %s\n", filepath.Base(backupPath))
		_ = os.RemoveAll(doltPath)
		_ = os.Rename(backupPath, doltPath)
		return fmt.Errorf("failed to reinitialize Dolt database: %w", err)
	}

	fmt.Printf("  Recovered Dolt database\n")
	fmt.Printf("  Corrupted database preserved at: %s\n", filepath.Base(backupPath))
	return nil
}

func serverModeIntegrityRecoveryGuard(issuegraphDir string) error {
	cfg, err := configfile.Load(issuegraphDir)
	if err != nil || cfg == nil || !cfg.IsDoltServerMode() {
		return nil
	}

	dbName := cfg.GetDoltDatabase()
	if dbName == "" {
		dbName = configfile.DefaultDoltDatabase
	}

	return fmt.Errorf(
		"automatic integrity recovery is disabled for server-mode repos because it can replace the wrong Dolt root; preserve %s and verify the configured database %q manually before reinitializing",
		getDatabasePath(issuegraphDir),
		dbName,
	)
}

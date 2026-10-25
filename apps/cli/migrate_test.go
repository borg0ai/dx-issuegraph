//go:build cgo

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/storage/dolt"
)

// TestMigrateCommand removed: detectDatabases, getDBVersion, formatDBList, dbInfo
// were removed in Dolt-native pruning. Migration is now handled by issuegraph init --from-jsonl.

// TestFormatDBList removed: formatDBList and dbInfo types were removed.

func TestMigrateSchemaFlagBoundaries(t *testing.T) {
	// Registration only: strict subprocess tests use the unique root persistent
	// --format=json alias because duplicate local --json registration can shadow
	// whether PersistentPreRunE sees the root flag as explicitly set.
	for _, name := range []string{"inspect", "dry-run"} {
		if flag := migrateSchemaCmd.Flags().Lookup(name); flag != nil {
			t.Errorf("migrate schema unexpectedly registers local --%s", name)
		}
	}
	for _, name := range []string{"json", "force"} {
		if flag := migrateSchemaCmd.Flags().Lookup(name); flag == nil {
			t.Errorf("migrate schema must register local --%s", name)
		}
	}
	if flag := rootCmd.PersistentFlags().Lookup("json"); flag == nil {
		t.Error("root command must register persistent --json")
	}
	if flag := rootCmd.PersistentFlags().Lookup("format"); flag == nil {
		t.Error("root command must register persistent --format")
	}
}

func TestMigrateRespectsConfigJSON(t *testing.T) {
	t.Skip("SQLite-specific: Dolt backend always uses 'dolt' directory, not custom database filenames")
	// Test that migrate respects custom database name from metadata.json
	tmpDir := t.TempDir()
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0750); err != nil {
		t.Fatalf("Failed to create .issuegraph directory: %v", err)
	}

	// Create metadata.json with custom database name
	configPath := filepath.Join(issuegraphDir, "metadata.json")
	configData := `{"database": "issuey.db", "version": "0.21.1"}`
	if err := os.WriteFile(configPath, []byte(configData), 0600); err != nil {
		t.Fatalf("Failed to create metadata.json: %v", err)
	}

	// Create old database with custom name
	oldDBPath := filepath.Join(issuegraphDir, "issuey.db")
	store, err := dolt.New(context.Background(), &dolt.Config{Path: oldDBPath})
	if err != nil {
		t.Skipf("skipping: Dolt server not available: %v", err)
	}
	ctx := context.Background()
	if err := store.SetLocalMetadata(ctx, "issuegraph_version", "0.21.1"); err != nil {
		t.Fatalf("Failed to set version: %v", err)
	}
	_ = store.Close()

	// Load config
	cfg, err := loadOrCreateConfig(issuegraphDir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify config respects custom database name
	if cfg.Database != "issuey.db" {
		t.Errorf("Expected database name 'issuey.db', got %s", cfg.Database)
	}

	expectedPath := filepath.Join(issuegraphDir, "issuey.db")
	actualPath := cfg.DatabasePath(issuegraphDir)
	if actualPath != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, actualPath)
	}

	// Verify database exists at custom path
	if _, err := os.Stat(actualPath); os.IsNotExist(err) {
		t.Errorf("Database does not exist at custom path: %s", actualPath)
	}
}

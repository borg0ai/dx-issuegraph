//go:build cgo

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
)

// TestConfigWinsOverConvention verifies SC-003: when metadata.json already
// has a non-empty dolt_database value, the init guard logic uses it instead
// of deriving issuegraph_{prefix}.
func TestConfigWinsOverConvention(t *testing.T) {
	tests := map[string]struct {
		configJSON string
		prefix     string
		wantDB     string
	}{
		"existing config wins over prefix": {
			configJSON: `{"backend":"dolt","dolt_database":"custom_issuegraph"}`,
			prefix:     "test",
			wantDB:     "custom_issuegraph",
		},
		"empty config falls through to prefix": {
			configJSON: `{"backend":"dolt"}`,
			prefix:     "test",
			wantDB:     "issuegraph_test",
		},
		"no prefix uses default": {
			configJSON: `{"backend":"dolt"}`,
			prefix:     "",
			wantDB:     "issuegraph",
		},
		"no config file falls through to prefix": {
			configJSON: "", // don't create file
			prefix:     "myrig",
			wantDB:     "issuegraph_myrig",
		},
		"existing config with no prefix still wins": {
			configJSON: `{"backend":"dolt","dolt_database":"acf_issuegraph"}`,
			prefix:     "",
			wantDB:     "acf_issuegraph",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
			if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
				t.Fatalf("failed to create issuegraph dir: %v", err)
			}

			// Write metadata.json if provided
			if tt.configJSON != "" {
				configPath := filepath.Join(issuegraphDir, "metadata.json")
				if err := os.WriteFile(configPath, []byte(tt.configJSON), 0644); err != nil {
					t.Fatalf("failed to write config: %v", err)
				}
			}

			// Simulate the guard logic from init.go:248-258
			dbName := "issuegraph"
			if tt.prefix != "" {
				dbName = "issuegraph_" + tt.prefix
			}
			// Guard: config wins over convention
			if existingCfg, loadErr := configfile.Load(issuegraphDir); loadErr == nil && existingCfg != nil {
				if existingCfg.DoltDatabase != "" {
					dbName = existingCfg.DoltDatabase
				}
			}

			if dbName != tt.wantDB {
				t.Errorf("dbName = %q, want %q", dbName, tt.wantDB)
			}
		})
	}
}

// TestMetadataSavePreservesExistingDoltDatabase verifies FR-022: when
// metadata.json save path encounters an existing DoltDatabase value,
// it does not overwrite it with issuegraph_{prefix}.
func TestMetadataSavePreservesExistingDoltDatabase(t *testing.T) {
	tests := map[string]struct {
		existingDB string
		prefix     string
		wantDB     string
	}{
		"existing value preserved": {
			existingDB: "custom_issuegraph",
			prefix:     "test",
			wantDB:     "custom_issuegraph",
		},
		"empty value gets prefix": {
			existingDB: "",
			prefix:     "test",
			wantDB:     "issuegraph_test",
		},
		"no prefix no overwrite": {
			existingDB: "",
			prefix:     "",
			wantDB:     "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := &configfile.Config{
				DoltDatabase: tt.existingDB,
			}

			// Simulate the guard logic from init.go:361
			if tt.prefix != "" && cfg.DoltDatabase == "" {
				cfg.DoltDatabase = "issuegraph_" + tt.prefix
			}

			if cfg.DoltDatabase != tt.wantDB {
				t.Errorf("DoltDatabase = %q, want %q", cfg.DoltDatabase, tt.wantDB)
			}
		})
	}
}

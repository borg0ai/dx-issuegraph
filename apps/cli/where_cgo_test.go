//go:build cgo

package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/storage"
	"github.com/borg0ai/dx-issuegraph/internal/storage/embeddeddolt"
	"github.com/borg0ai/dx-issuegraph/internal/utils"
)

func TestWhereCommand_ReadsPrefixFromEmbeddedStore(t *testing.T) {
	saveAndRestoreGlobals(t)
	ensureCleanGlobalState(t)
	initConfigForTest(t)

	originalCmdCtx := cmdCtx
	originalJSONOutput := jsonOutput
	originalRootCtx := rootCtx
	defer func() {
		cmdCtx = originalCmdCtx
		jsonOutput = originalJSONOutput
		rootCtx = originalRootCtx
	}()

	resetCommandContext()

	repoDir := t.TempDir()
	issuegraphDir := filepath.Join(repoDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0o755); err != nil {
		t.Fatalf("mkdir issuegraph dir: %v", err)
	}

	cfg := &configfile.Config{
		Database:     "dolt",
		Backend:      configfile.BackendDolt,
		DoltMode:     configfile.DoltModeEmbedded,
		DoltDatabase: "embedcfg",
	}
	if err := cfg.Save(issuegraphDir); err != nil {
		t.Fatalf("save metadata: %v", err)
	}

	store, err := embeddeddolt.Open(context.Background(), issuegraphDir, "embedcfg", "main")
	if err != nil {
		t.Fatalf("embeddeddolt.Open: %v", err)
	}
	if err := store.SetConfig(context.Background(), "issue_prefix", "storeprefix"); err != nil {
		_ = store.Close()
		t.Fatalf("SetConfig(issue_prefix): %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close(): %v", err)
	}

	dbDir := filepath.Join(issuegraphDir, "dolt")
	t.Setenv("ISSUEGRAPH_DIR", "")
	t.Setenv("ISSUEGRAPH_DB", dbDir)
	t.Setenv("ISSUEGRAPH_DB", "")
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_MODE", "")
	t.Setenv("ISSUEGRAPH_DOLT_SHARED_SERVER", "")

	dbFlag := rootCmd.PersistentFlags().Lookup("db")
	originalFlagValue := dbFlag.Value.String()
	originalFlagChanged := dbFlag.Changed
	if err := dbFlag.Value.Set(""); err != nil {
		t.Fatalf("reset db flag: %v", err)
	}
	dbFlag.Changed = false
	t.Cleanup(func() {
		_ = dbFlag.Value.Set(originalFlagValue)
		dbFlag.Changed = originalFlagChanged
	})

	jsonOutput = true
	rootCtx = context.Background()

	if err := withStorage(rootCtx, nil, dbDir, func(currentStore storage.DoltStorage) error {
		prefix, err := currentStore.GetConfig(rootCtx, "issue_prefix")
		if err != nil {
			return err
		}
		if prefix != "storeprefix" {
			t.Fatalf("precheck issue_prefix = %q, want %q", prefix, "storeprefix")
		}
		return nil
	}); err != nil {
		t.Fatalf("withStorage precheck: %v", err)
	}

	output := captureStdout(t, func() error {
		return whereCmd.RunE(whereCmd, nil)
	})

	var result WhereResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("json.Unmarshal(%q): %v", output, err)
	}

	if !utils.PathsEqual(result.Path, issuegraphDir) {
		t.Fatalf("Path = %q, want %q", result.Path, issuegraphDir)
	}
	if result.DatabasePath == "" {
		t.Fatal("DatabasePath = empty, want resolved dolt path")
	}
	base := filepath.Base(result.DatabasePath)
	if base != "dolt" && base != "embeddeddolt" {
		t.Fatalf("DatabasePath = %q, want dolt-style basename", result.DatabasePath)
	}
	if result.Prefix != "storeprefix" {
		t.Fatalf("Prefix = %q, want %q", result.Prefix, "storeprefix")
	}
}

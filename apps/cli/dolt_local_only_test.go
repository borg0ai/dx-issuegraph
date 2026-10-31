package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/config"
	"github.com/borg0ai/dx-issuegraph/internal/issuegraph"
	"github.com/spf13/cobra"
)

// Unit tests for isDoltLocalOnly() and related guards.
// Cannot be parallel: tests modify global config state.

func TestIsDoltLocalOnly_AlwaysTrue(t *testing.T) {
	config.ResetForTesting()
	t.Cleanup(func() { config.ResetForTesting() })
	if err := config.Initialize(); err != nil {
		t.Fatalf("config.Initialize: %v", err)
	}

	if !isDoltLocalOnly() {
		t.Error("isDoltLocalOnly() = false, want true without config")
	}
}

func TestIsDoltLocalOnly_TrueWhenConfigSet(t *testing.T) {
	config.ResetForTesting()
	t.Cleanup(func() { config.ResetForTesting() })
	if err := config.Initialize(); err != nil {
		t.Fatalf("config.Initialize: %v", err)
	}
	config.Set("dolt.local-only", true)

	if !isDoltLocalOnly() {
		t.Error("isDoltLocalOnly() = false, want true when legacy config is enabled")
	}
}

func TestIsDoltLocalOnly_CannotBeDisabledByConfig(t *testing.T) {
	config.ResetForTesting()
	t.Cleanup(func() { config.ResetForTesting() })
	if err := config.Initialize(); err != nil {
		t.Fatalf("config.Initialize: %v", err)
	}
	config.Set("dolt.local-only", true)
	config.Set("dolt.local-only", false)
	if !isDoltLocalOnly() {
		t.Error("isDoltLocalOnly() = false after disabling legacy config, want permanent local-only")
	}
}

func TestMaybeAutoPushDisabled(t *testing.T) {
	oldStore := store
	store = nil
	t.Cleanup(func() { store = oldStore })
	config.ResetForTesting()
	t.Cleanup(func() { config.ResetForTesting() })
	if err := config.Initialize(); err != nil {
		t.Fatalf("config.Initialize: %v", err)
	}
	config.Set("dolt.local-only", true)
	t.Setenv("ISSUEGRAPH_DOLT_AUTO_PUSH", "true")
	maybeAutoPush(context.Background())
}

func TestDoltRemoteSyncCommandsFailClosedBeforeStoreAccess(t *testing.T) {
	oldStore := store
	oldJSON := jsonOutput
	store = nil
	jsonOutput = false
	t.Cleanup(func() {
		store = oldStore
		jsonOutput = oldJSON
		issuegraph.ResetCaches()
	})
	t.Chdir(t.TempDir())
	issuegraph.ResetCaches()
	if err := os.MkdirAll(".issuegraph", 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(".issuegraph", "config.yaml"), []byte("project: test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	backupConfig, err := json.Marshal(doltBackupConfig{BackupURL: "https://example.test/backup"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(".issuegraph", "dolt-backup.json"), backupConfig, 0o600); err != nil {
		t.Fatal(err)
	}
	issuegraph.ResetCaches()

	cases := []struct {
		name string
		cmd  *cobra.Command
		args []string
	}{
		{name: "dolt push", cmd: doltPushCmd},
		{name: "dolt pull", cmd: doltPullCmd},
		{name: "dolt remote add", cmd: doltRemoteAddCmd, args: []string{"origin", "https://example.test/issues.git"}},
		{name: "dolt remote list", cmd: doltRemoteListCmd},
		{name: "dolt remote remove", cmd: doltRemoteRemoveCmd, args: []string{"origin"}},
		{name: "dolt remote reset-data", cmd: doltRemoteResetDataCmd, args: []string{"origin"}},
		{name: "sync", cmd: syncCmd},
		{name: "federation sync", cmd: federationSyncCmd},
		{name: "federation status", cmd: federationStatusCmd},
		{name: "federation add-peer", cmd: federationAddPeerCmd, args: []string{"peer", "file:///tmp/peer"}},
		{name: "remote backup init", cmd: backupInitCmd, args: []string{"https://example.test/backup"}},
		{name: "remote backup sync", cmd: backupSyncCmd},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			stderr := captureStderr(t, func() { err = rootCmd.PersistentPreRunE(tc.cmd, tc.args) })
			if err == nil {
				t.Fatal("command succeeded; want local-only refusal")
			}
			if !strings.Contains(stderr, "IssueGraph keeps issue data local") {
				t.Fatalf("stderr = %q, want stable local-only refusal", stderr)
			}
			if store != nil {
				t.Fatal("remote command opened a store")
			}
		})
	}
}

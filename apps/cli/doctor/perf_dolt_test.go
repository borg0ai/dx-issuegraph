//go:build cgo

package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
)

func TestRunDoltPerformanceDiagnostics_RequiresServer(t *testing.T) {
	// Server-only mode: diagnostics require a running dolt sql-server.
	// Without a server, RunDoltPerformanceDiagnostics should return an error.
	tmpDir := t.TempDir()
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write config pointing to a nonexistent database
	cfg := configfile.DefaultConfig()
	cfg.Backend = configfile.BackendDolt
	cfg.DoltMode = configfile.DoltModeServer
	cfg.DoltServerHost = "127.0.0.1"
	cfg.DoltServerPort = doctorTestServerPort()
	cfg.DoltDatabase = "doctest_perf_nonexistent"
	if err := cfg.Save(issuegraphDir); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	_, err := RunDoltPerformanceDiagnostics(tmpDir, false)
	if err == nil {
		t.Fatal("expected error when no dolt server is running")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "not running") && !strings.Contains(errStr, "not reachable") && !strings.Contains(errStr, "database not found") {
		t.Errorf("expected server/database error, got: %v", err)
	}
}

func TestRunDoltPerformanceDiagnosticsDeclineNonDoltBackend(t *testing.T) {
	tmpDir := t.TempDir()
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0o750); err != nil {
		t.Fatal(err)
	}
	cfg := &configfile.Config{Backend: configfile.BackendSQLite, SQLitePath: "issuegraph.db"}
	if err := cfg.Save(issuegraphDir); err != nil {
		t.Fatalf("save SQLite config: %v", err)
	}

	_, err := RunDoltPerformanceDiagnostics(tmpDir, false)
	if err == nil {
		t.Fatal("expected Dolt-only diagnostics to decline a non-Dolt workspace")
	}
	message := err.Error()
	if !strings.Contains(message, "Dolt") || !strings.Contains(message, `"sqlite"`) {
		t.Fatalf("diagnostic error should explain the backend mismatch: %v", err)
	}
}

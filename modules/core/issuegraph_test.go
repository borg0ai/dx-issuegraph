package issuegraph_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/borg0ai/dx-issuegraph"
	"github.com/borg0ai/dx-issuegraph/internal/testutil"
)

// testServerPort is the port of the shared test Dolt server (0 = not running).
var testServerPort int

func TestMain(m *testing.M) {
	os.Exit(testMainInner(m))
}

// testMainInner holds TestMain's body so its defers run before the process
// exits — os.Exit skips deferred calls, so TestMain itself must never defer
// anything (be-5kkk6).
func testMainInner(m *testing.M) int {
	os.Setenv("ISSUEGRAPH_TEST_MODE", "1")
	// AD-01 (be-c5p): allow root tests to connect to the test container.
	os.Setenv("ISSUEGRAPH_TEST_SERVER", "1")
	if err := testutil.EnsureDoltContainerForTestMain(); err != nil {
		fmt.Fprintf(os.Stderr, "WARN: %v, skipping Dolt tests\n", err)
	} else {
		defer testutil.TerminateDoltContainer()
		testServerPort = testutil.DoltContainerPortInt()
	}

	code := m.Run()

	os.Unsetenv("ISSUEGRAPH_DOLT_PORT")
	os.Unsetenv("ISSUEGRAPH_TEST_MODE")
	return code
}

func skipIfNoDolt(t *testing.T) {
	t.Helper()
	testutil.RequireDoltBinary(t)
}

func skipIfNoDoltServer(t *testing.T) {
	t.Helper()
	if testServerPort == 0 {
		t.Skip("Test Dolt server not available, skipping test")
	}
	addr := fmt.Sprintf("127.0.0.1:%d", testServerPort)
	conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
	if err != nil {
		t.Skipf("Dolt server not running on %s, skipping test", addr)
	}
	_ = conn.Close()
}

func TestOpen(t *testing.T) {
	skipIfNoDoltServer(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test-dolt")

	ctx := context.Background()
	store, err := issuegraph.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	if store == nil {
		t.Error("expected non-nil storage")
	}
}

func TestFindDatabasePath(t *testing.T) {
	// This will return empty string in test environment without a database
	path := issuegraph.FindDatabasePath()
	// Just verify it doesn't panic
	_ = path
}

func TestFindIssueGraphDir(t *testing.T) {
	// This will return empty string or a valid path
	dir := issuegraph.FindIssueGraphDir()
	// Just verify it doesn't panic
	_ = dir
}

func TestOpenFromConfig_Embedded(t *testing.T) {
	// This test requires a running Dolt server (embedded mode is not yet implemented;
	// New() always connects via MySQL protocol to dolt sql-server).
	skipIfNoDoltServer(t)

	// Create a .issuegraph dir with metadata.json configured for embedded mode
	tmpDir := t.TempDir()
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatalf("failed to create .issuegraph dir: %v", err)
	}

	metadata := `{"backend":"dolt","database":"dolt","dolt_database":"testdb","dolt_mode":"embedded"}`
	if err := os.WriteFile(filepath.Join(issuegraphDir, "metadata.json"), []byte(metadata), 0644); err != nil {
		t.Fatalf("failed to write metadata.json: %v", err)
	}

	ctx := context.Background()
	store, err := issuegraph.OpenFromConfig(ctx, issuegraphDir)
	if err != nil {
		t.Fatalf("OpenFromConfig (embedded) failed: %v", err)
	}
	defer store.Close()

	if store == nil {
		t.Error("expected non-nil storage")
	}
}

func TestOpenFromConfig_DefaultsToEmbedded(t *testing.T) {
	// This test requires a running Dolt server (embedded mode is not yet implemented;
	// New() always connects via MySQL protocol to dolt sql-server).
	skipIfNoDoltServer(t)

	// metadata.json without dolt_mode should default to embedded
	tmpDir := t.TempDir()
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatalf("failed to create .issuegraph dir: %v", err)
	}

	metadata := `{"backend":"dolt","database":"dolt"}`
	if err := os.WriteFile(filepath.Join(issuegraphDir, "metadata.json"), []byte(metadata), 0644); err != nil {
		t.Fatalf("failed to write metadata.json: %v", err)
	}

	ctx := context.Background()
	store, err := issuegraph.OpenFromConfig(ctx, issuegraphDir)
	if err != nil {
		t.Fatalf("OpenFromConfig (default) failed: %v", err)
	}
	defer store.Close()

	if store == nil {
		t.Error("expected non-nil storage")
	}
}

func TestOpenFromConfig_ServerModeFailsWithoutServer(t *testing.T) {
	// Server mode should fail-fast when no server is listening.
	// Clear port env vars so the config port from metadata.json is used —
	// ISSUEGRAPH_DOLT_SERVER_PORT takes priority over ISSUEGRAPH_DOLT_PORT in
	// GetDoltServerPort, so both must be cleared. Keep ISSUEGRAPH_TEST_MODE=1
	// so auto-start is suppressed and the connection uses the free port
	// from metadata.json (which has nothing listening).
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_PORT", "")
	t.Setenv("ISSUEGRAPH_DOLT_PORT", "")

	tmpDir := t.TempDir()
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatalf("failed to create .issuegraph dir: %v", err)
	}

	// Dynamically find an unused port by binding to :0 then closing
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	freePort := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	metadata := fmt.Sprintf(`{"backend":"dolt","database":"dolt","dolt_mode":"server","dolt_server_host":"127.0.0.1","dolt_server_port":%d}`, freePort)
	if err := os.WriteFile(filepath.Join(issuegraphDir, "metadata.json"), []byte(metadata), 0644); err != nil {
		t.Fatalf("failed to write metadata.json: %v", err)
	}

	ctx := context.Background()
	_, openErr := issuegraph.OpenFromConfig(ctx, issuegraphDir)
	if openErr == nil {
		t.Fatal("OpenFromConfig (server mode) should fail when no server is running")
	}
	// Should contain "unreachable" from the fail-fast TCP check
	if !strings.Contains(openErr.Error(), "unreachable") {
		t.Errorf("expected 'unreachable' in error, got: %v", openErr)
	}
}

func TestOpenFromConfig_NoMetadata(t *testing.T) {
	skipIfNoDoltServer(t)
	// Missing metadata.json should use defaults (server mode)
	tmpDir := t.TempDir()
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatalf("failed to create .issuegraph dir: %v", err)
	}

	ctx := context.Background()
	store, err := issuegraph.OpenFromConfig(ctx, issuegraphDir)
	if err != nil {
		t.Fatalf("OpenFromConfig (no metadata) failed: %v", err)
	}
	defer store.Close()

	if store == nil {
		t.Error("expected non-nil storage")
	}
}

func TestOpenBestAvailable_ServerMode(t *testing.T) {
	skipIfNoDoltServer(t)

	tmpDir := t.TempDir()
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatalf("failed to create .issuegraph dir: %v", err)
	}

	metadata := fmt.Sprintf(`{"backend":"dolt","database":"dolt","dolt_mode":"server","dolt_server_host":"127.0.0.1","dolt_server_port":%d}`, testServerPort)
	if err := os.WriteFile(filepath.Join(issuegraphDir, "metadata.json"), []byte(metadata), 0644); err != nil {
		t.Fatalf("failed to write metadata.json: %v", err)
	}

	ctx := context.Background()
	store, err := issuegraph.OpenBestAvailable(ctx, issuegraphDir)
	if err != nil {
		t.Fatalf("OpenBestAvailable (server mode) failed: %v", err)
	}
	defer store.Close()

	if store == nil {
		t.Error("expected non-nil storage")
	}
}

func TestOpenBestAvailable_ServerMode_FailsWithoutServer(t *testing.T) {
	// OpenBestAvailable in server mode should propagate the fail-fast error.
	// Clear port env vars (both ISSUEGRAPH_DOLT_SERVER_PORT and ISSUEGRAPH_DOLT_PORT)
	// so the config port from metadata.json is used. Keep ISSUEGRAPH_TEST_MODE=1
	// so auto-start is suppressed.
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_PORT", "")
	t.Setenv("ISSUEGRAPH_DOLT_PORT", "")

	tmpDir := t.TempDir()
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatalf("failed to create .issuegraph dir: %v", err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	freePort := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	metadata := fmt.Sprintf(`{"backend":"dolt","database":"dolt","dolt_mode":"server","dolt_server_host":"127.0.0.1","dolt_server_port":%d}`, freePort)
	if err := os.WriteFile(filepath.Join(issuegraphDir, "metadata.json"), []byte(metadata), 0644); err != nil {
		t.Fatalf("failed to write metadata.json: %v", err)
	}

	ctx := context.Background()
	_, openErr := issuegraph.OpenBestAvailable(ctx, issuegraphDir)
	if openErr == nil {
		t.Fatal("OpenBestAvailable (server mode) should fail when no server is running")
	}
	if !strings.Contains(openErr.Error(), "unreachable") {
		t.Errorf("expected 'unreachable' in error, got: %v", openErr)
	}
}

func TestFindAllDatabases(t *testing.T) {
	// This scans the file system, just verify it doesn't panic
	dbs := issuegraph.FindAllDatabases()
	// Should return a slice (possibly empty)
	if dbs == nil {
		t.Error("expected non-nil slice")
	}
}

// Test that exported constants have correct values
func TestConstants(t *testing.T) {
	// Status constants
	if issuegraph.StatusOpen != "open" {
		t.Errorf("StatusOpen = %q, want %q", issuegraph.StatusOpen, "open")
	}
	if issuegraph.StatusInProgress != "in_progress" {
		t.Errorf("StatusInProgress = %q, want %q", issuegraph.StatusInProgress, "in_progress")
	}
	if issuegraph.StatusBlocked != "blocked" {
		t.Errorf("StatusBlocked = %q, want %q", issuegraph.StatusBlocked, "blocked")
	}
	if issuegraph.StatusClosed != "closed" {
		t.Errorf("StatusClosed = %q, want %q", issuegraph.StatusClosed, "closed")
	}

	// IssueType constants
	if issuegraph.TypeBug != "bug" {
		t.Errorf("TypeBug = %q, want %q", issuegraph.TypeBug, "bug")
	}
	if issuegraph.TypeFeature != "feature" {
		t.Errorf("TypeFeature = %q, want %q", issuegraph.TypeFeature, "feature")
	}
	if issuegraph.TypeTask != "task" {
		t.Errorf("TypeTask = %q, want %q", issuegraph.TypeTask, "task")
	}
	if issuegraph.TypeEpic != "epic" {
		t.Errorf("TypeEpic = %q, want %q", issuegraph.TypeEpic, "epic")
	}

	// DependencyType constants
	if issuegraph.DepBlocks != "blocks" {
		t.Errorf("DepBlocks = %q, want %q", issuegraph.DepBlocks, "blocks")
	}
	if issuegraph.DepRelated != "related" {
		t.Errorf("DepRelated = %q, want %q", issuegraph.DepRelated, "related")
	}
}

func TestPublicAPITypeAssertions(t *testing.T) {
	skipIfNoDoltServer(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test-dolt")

	ctx := context.Background()
	store, err := issuegraph.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	t.Run("RemoteStore", func(t *testing.T) {
		rs, ok := store.(issuegraph.RemoteStore)
		if !ok {
			t.Fatal("store does not satisfy issuegraph.RemoteStore")
		}
		// Verify a method is callable (no remotes configured, so empty list)
		remotes, err := rs.ListRemotes(ctx)
		if err != nil {
			t.Fatalf("ListRemotes failed: %v", err)
		}
		_ = remotes
	})

	t.Run("SyncStore", func(t *testing.T) {
		_, ok := store.(issuegraph.SyncStore)
		if !ok {
			t.Fatal("store does not satisfy issuegraph.SyncStore")
		}
	})

	t.Run("VersionControlReader", func(t *testing.T) {
		vcr, ok := store.(issuegraph.VersionControlReader)
		if !ok {
			t.Fatal("store does not satisfy issuegraph.VersionControlReader")
		}

		branch, err := vcr.CurrentBranch(ctx)
		if err != nil {
			t.Fatalf("CurrentBranch failed: %v", err)
		}
		if branch == "" {
			t.Error("expected non-empty branch name")
		}

		commit, err := vcr.GetCurrentCommit(ctx)
		if err != nil {
			t.Fatalf("GetCurrentCommit failed: %v", err)
		}
		if commit == "" {
			t.Error("expected non-empty commit hash")
		}

		exists, err := vcr.CommitExists(ctx, commit)
		if err != nil {
			t.Fatalf("CommitExists failed: %v", err)
		}
		if !exists {
			t.Errorf("CommitExists(%s) = false, want true", commit)
		}

		status, err := vcr.Status(ctx)
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}
		_ = status

		logs, err := vcr.Log(ctx, 5)
		if err != nil {
			t.Fatalf("Log failed: %v", err)
		}
		_ = logs
	})
}

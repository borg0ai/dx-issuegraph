package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/utils"
)

func TestSharedStore_NilSafe(t *testing.T) {
	// A nil SharedStore should not panic
	var ss *SharedStore
	if ss.Store() != nil {
		t.Error("expected nil store from nil SharedStore")
	}
	if ss.IssueGraphDir() != "" {
		t.Error("expected empty issuegraphDir from nil SharedStore")
	}
	// Close should not panic on nil
	ss.Close()
}

func TestSharedStore_NoDatabase(t *testing.T) {
	// SharedStore with no database should return nil Store
	tmpDir := t.TempDir()
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}

	ss := NewSharedStore(tmpDir)
	defer ss.Close()

	if ss.Store() != nil {
		t.Error("expected nil store when no database exists")
	}
	if ss.IssueGraphDir() == "" {
		t.Error("expected non-empty issuegraphDir")
	}
}

func TestSharedStore_BareParentWorktreeFallback(t *testing.T) {
	clearResolveIssueGraphDirCache()
	t.Cleanup(clearResolveIssueGraphDirCache)

	bareDir, worktreeDir := setupDoctorBareParentWorktree(t)
	bareIssueGraphDir := filepath.Join(bareDir, ".issuegraph")
	if err := os.MkdirAll(bareIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}

	ss := NewSharedStore(worktreeDir)
	defer ss.Close()

	if ss.IssueGraphDir() != utils.CanonicalizePath(bareIssueGraphDir) {
		t.Fatalf("IssueGraphDir() = %q, want %q", ss.IssueGraphDir(), utils.CanonicalizePath(bareIssueGraphDir))
	}
	if ss.Store() != nil {
		t.Fatal("expected nil store when bare parent has no database")
	}
}

func TestSharedStore_DoubleClose(t *testing.T) {
	// Double close should not panic
	tmpDir := t.TempDir()
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}

	ss := NewSharedStore(tmpDir)
	ss.Close()
	ss.Close() // Should not panic
}

func TestSharedStore_WithStoreChecks_NoDatabase(t *testing.T) {
	// WithStore variants should return sensible results when store is nil
	tmpDir := t.TempDir()
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}

	ss := NewSharedStore(tmpDir)
	defer ss.Close()

	// All WithStore checks should handle nil store gracefully
	tests := []struct {
		name  string
		check DoctorCheck
	}{
		{"DatabaseVersion", CheckDatabaseVersionWithStore(ss, "0.61.0")},
		{"SchemaCompatibility", CheckSchemaCompatibilityWithStore(ss)},
		{"DatabaseIntegrity", CheckDatabaseIntegrityWithStore(ss)},
		{"IDFormat", CheckIDFormatWithStore(ss)},
		{"DependencyCycles", CheckDependencyCyclesWithStore(ss)},
		{"RepoFingerprint", CheckRepoFingerprintWithStore(ss, tmpDir)},
		{"DatabaseSize", CheckDatabaseSizeWithStore(ss)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic and should return a valid check
			if tt.check.Name == "" {
				t.Error("expected non-empty check name")
			}
			if tt.check.Status == "" {
				t.Error("expected non-empty status")
			}
		})
	}
}

func TestCheckDatabaseVersionWithStore_BareParentWorktreeServerModeNoLocalDolt(t *testing.T) {
	clearResolveIssueGraphDirCache()
	t.Cleanup(clearResolveIssueGraphDirCache)

	bareDir, worktreeDir := setupDoctorBareParentWorktree(t)
	bareIssueGraphDir := filepath.Join(bareDir, ".issuegraph")
	if err := os.MkdirAll(bareIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(bareIssueGraphDir, "metadata.json"),
		[]byte("{\"backend\":\"dolt\",\"dolt_mode\":\"server\",\"dolt_database\":\"issuegraph_feature\"}"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	ss := NewSharedStore(worktreeDir)
	defer ss.Close()

	check := CheckDatabaseVersionWithStore(ss, "0.61.0")
	if check.Message == "No dolt database found" {
		t.Fatalf("expected server-mode bare worktree to avoid false missing-db error, got %q", check.Message)
	}
	if check.Message != "Unable to open database" {
		t.Fatalf("expected Unable to open database, got %q", check.Message)
	}
	if check.Detail != "Storage: Dolt" {
		t.Fatalf("expected Storage: Dolt detail, got %q", check.Detail)
	}
	if _, err := os.Stat(filepath.Join(worktreeDir, ".issuegraph")); !os.IsNotExist(err) {
		t.Fatalf("expected no local .issuegraph in worktree, got err=%v", err)
	}
}

func TestGetSuppressedChecksWithStore_NilStore(t *testing.T) {
	tmpDir := t.TempDir()
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}

	ss := NewSharedStore(tmpDir)
	defer ss.Close()

	suppressed := GetSuppressedChecksWithStore(ss)
	if len(suppressed) != 0 {
		t.Errorf("expected empty suppressed map, got %v", suppressed)
	}
}

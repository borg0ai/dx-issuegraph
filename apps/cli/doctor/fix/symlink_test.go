package fix

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// skipOnWindows skips tests that rely on Unix file permissions or symlinks.
func skipOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("skipping Unix permission/symlink test on Windows")
	}
}

// TestPermissions_SkipsSymlinkedIssueGraphDir verifies that permission fixes are skipped
// when .issuegraph directory is a symlink (common on NixOS with home-manager).
//
// Behavior being tested:
// - When .issuegraph is a symlink, Permissions() should return nil without changing anything
// - This prevents attempts to chmod symlink targets (which may be read-only like /nix/store)
func TestPermissions_SkipsSymlinkedIssueGraphDir(t *testing.T) {
	skipOnWindows(t)
	tmpDir := t.TempDir()

	// Create target .issuegraph directory with wrong permissions
	targetDir := filepath.Join(tmpDir, "target-issuegraph")
	if err := os.MkdirAll(targetDir, 0777); err != nil { // intentionally wrong permissions
		t.Fatal(err)
	}

	// Create workspace with symlinked .issuegraph
	workspaceDir := filepath.Join(tmpDir, "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatal(err)
	}

	symlinkPath := filepath.Join(workspaceDir, ".issuegraph")
	if err := os.Symlink(targetDir, symlinkPath); err != nil {
		t.Fatal(err)
	}

	// Get target's permissions before fix
	targetInfoBefore, err := os.Stat(targetDir)
	if err != nil {
		t.Fatal(err)
	}
	permsBefore := targetInfoBefore.Mode().Perm()

	// Run Permissions fix
	err = Permissions(workspaceDir)
	if err != nil {
		t.Fatalf("Permissions() returned error for symlinked .issuegraph: %v", err)
	}

	// Verify target's permissions were NOT changed
	targetInfoAfter, err := os.Stat(targetDir)
	if err != nil {
		t.Fatal(err)
	}
	permsAfter := targetInfoAfter.Mode().Perm()

	if permsAfter != permsBefore {
		t.Errorf("Target directory permissions were changed through symlink!")
		t.Errorf("Before: %o, After: %o", permsBefore, permsAfter)
		t.Error("This could cause issues on NixOS where target may be in /nix/store (read-only)")
	}
}

// TestPermissions_SkipsSymlinkedDatabase verifies that chmod is skipped for
// symlinked database files, but .issuegraph directory permissions are still fixed.
func TestPermissions_SkipsSymlinkedDatabase(t *testing.T) {
	skipOnWindows(t)
	tmpDir := t.TempDir()

	// Create real .issuegraph directory with wrong permissions
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0777); err != nil { // intentionally wrong
		t.Fatal(err)
	}

	// Create target database file with wrong permissions
	targetDir := filepath.Join(tmpDir, "target")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	targetDB := filepath.Join(targetDir, "issuegraph.db")
	if err := os.WriteFile(targetDB, []byte("test"), 0644); err != nil { // intentionally world-readable
		t.Fatal(err)
	}

	// Create symlink to database
	symlinkPath := filepath.Join(issuegraphDir, "issuegraph.db")
	if err := os.Symlink(targetDB, symlinkPath); err != nil {
		t.Fatal(err)
	}

	// Get target's permissions before fix
	targetInfoBefore, err := os.Stat(targetDB)
	if err != nil {
		t.Fatal(err)
	}
	permsBefore := targetInfoBefore.Mode().Perm()

	// Run Permissions fix
	err = Permissions(tmpDir)
	if err != nil {
		t.Fatalf("Permissions() returned error for symlinked database: %v", err)
	}

	// Verify .issuegraph directory permissions WERE fixed (not a symlink)
	issuegraphInfo, err := os.Stat(issuegraphDir)
	if err != nil {
		t.Fatal(err)
	}
	if issuegraphInfo.Mode().Perm() != 0700 {
		t.Errorf("Expected .issuegraph to have 0700 permissions, got %o", issuegraphInfo.Mode().Perm())
	}

	// Verify target database permissions were NOT changed (it's a symlink)
	targetInfoAfter, err := os.Stat(targetDB)
	if err != nil {
		t.Fatal(err)
	}
	permsAfter := targetInfoAfter.Mode().Perm()

	if permsAfter != permsBefore {
		t.Errorf("Target database permissions were changed through symlink!")
		t.Errorf("Before: %o, After: %o", permsBefore, permsAfter)
		t.Error("chmod should not be called on symlinked files")
	}
}

// TestPermissions_FixesRegularFiles verifies that permissions ARE fixed for
// regular (non-symlinked) files.
func TestPermissions_FixesRegularFiles(t *testing.T) {
	skipOnWindows(t)
	tmpDir := t.TempDir()

	// Create .issuegraph directory with wrong permissions
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0777); err != nil { // intentionally wrong
		t.Fatal(err)
	}

	// Create database with wrong permissions
	dbPath := filepath.Join(issuegraphDir, "issuegraph.db")
	if err := os.WriteFile(dbPath, []byte("test"), 0644); err != nil { // intentionally world-readable
		t.Fatal(err)
	}

	// Run Permissions fix
	err := Permissions(tmpDir)
	if err != nil {
		t.Fatalf("Permissions() failed: %v", err)
	}

	// Verify .issuegraph directory now has 0700
	issuegraphInfo, err := os.Stat(issuegraphDir)
	if err != nil {
		t.Fatal(err)
	}
	if issuegraphInfo.Mode().Perm() != 0700 {
		t.Errorf("Expected .issuegraph to have 0700 permissions, got %o", issuegraphInfo.Mode().Perm())
	}

	// Verify database now has at least 0600 (read/write for owner)
	dbInfo, err := os.Stat(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if dbInfo.Mode().Perm()&0600 != 0600 {
		t.Errorf("Expected database to have at least 0600 permissions, got %o", dbInfo.Mode().Perm())
	}
}

func TestPermissions_FixesSharedWorktreeIssueGraphDir(t *testing.T) {
	skipOnWindows(t)
	mainRepoDir, worktreeDir := setupSharedWorktreeWorkspace(t)
	sharedIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
	if err := os.MkdirAll(sharedIssueGraphDir, 0o777); err != nil {
		t.Fatal(err)
	}

	if err := Permissions(worktreeDir); err != nil {
		t.Fatalf("Permissions() failed for shared worktree: %v", err)
	}

	sharedInfo, err := os.Stat(sharedIssueGraphDir)
	if err != nil {
		t.Fatal(err)
	}
	if sharedInfo.Mode().Perm() != 0700 {
		t.Errorf("Expected shared .issuegraph to have 0700 permissions, got %o", sharedInfo.Mode().Perm())
	}
	if _, err := os.Stat(filepath.Join(worktreeDir, ".issuegraph")); !os.IsNotExist(err) {
		t.Fatalf("expected no worktree-local .issuegraph directory, got err=%v", err)
	}
}

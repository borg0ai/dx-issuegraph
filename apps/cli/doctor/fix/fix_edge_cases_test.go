package fix

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestIsWithinWorkspace_PathTraversal tests path traversal attempts
func TestIsWithinWorkspace_PathTraversal(t *testing.T) {
	root := t.TempDir()

	tests := []struct {
		name      string
		candidate string
		want      bool
	}{
		{
			name:      "simple dotdot traversal",
			candidate: filepath.Join(root, "..", "etc"),
			want:      false,
		},
		{
			name:      "dotdot in middle of path",
			candidate: filepath.Join(root, "subdir", "..", "..", "etc"),
			want:      false,
		},
		{
			name:      "multiple dotdot",
			candidate: filepath.Join(root, "..", "..", ".."),
			want:      false,
		},
		{
			name:      "dotdot stays within workspace",
			candidate: filepath.Join(root, "a", "b", "..", "c"),
			want:      true,
		},
		{
			name:      "relative path with dotdot",
			candidate: filepath.Join(root, "subdir", "..", "file"),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isWithinWorkspace(root, tt.candidate)
			if got != tt.want {
				t.Errorf("isWithinWorkspace(%q, %q) = %v, want %v", root, tt.candidate, got, tt.want)
			}
		})
	}
}

// TestValidateIssueGraphWorkspace_EdgeCases tests edge cases for workspace validation
func TestValidateIssueGraphWorkspace_EdgeCases(t *testing.T) {
	t.Run("nested .issuegraph directories", func(t *testing.T) {
		// Create a workspace with nested .issuegraph directories
		dir := setupTestWorkspace(t)
		nestedDir := filepath.Join(dir, "subdir")
		nestedIssueGraphDir := filepath.Join(nestedDir, ".issuegraph")
		if err := os.MkdirAll(nestedIssueGraphDir, 0755); err != nil {
			t.Fatalf("failed to create nested .issuegraph: %v", err)
		}

		// Root workspace should be valid
		if err := validateIssueGraphWorkspace(dir); err != nil {
			t.Errorf("expected root workspace to be valid, got: %v", err)
		}

		// Nested workspace should also be valid
		if err := validateIssueGraphWorkspace(nestedDir); err != nil {
			t.Errorf("expected nested workspace to be valid, got: %v", err)
		}
	})

	t.Run(".issuegraph as a file not directory", func(t *testing.T) {
		dir := t.TempDir()
		issuegraphFile := filepath.Join(dir, ".issuegraph")
		// Create .issuegraph as a file instead of directory
		if err := os.WriteFile(issuegraphFile, []byte("not a directory"), 0600); err != nil {
			t.Fatalf("failed to create .issuegraph file: %v", err)
		}

		err := validateIssueGraphWorkspace(dir)
		if err == nil {
			t.Fatal("expected error when .issuegraph is a file")
		}
		if got := err.Error(); got != "not a issuegraph workspace: .issuegraph directory not found for "+dir {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run(".issuegraph as symlink to directory", func(t *testing.T) {
		dir := t.TempDir()
		// Create actual .issuegraph directory elsewhere
		actualIssueGraphDir := filepath.Join(t.TempDir(), "actual_issuegraph")
		if err := os.MkdirAll(actualIssueGraphDir, 0755); err != nil {
			t.Fatalf("failed to create actual issuegraph dir: %v", err)
		}

		// Create symlink .issuegraph -> actual_issuegraph
		symlinkPath := filepath.Join(dir, ".issuegraph")
		if err := os.Symlink(actualIssueGraphDir, symlinkPath); err != nil {
			t.Skipf("symlink creation failed (may not be supported): %v", err)
		}

		// Should be valid - symlink to directory is acceptable
		if err := validateIssueGraphWorkspace(dir); err != nil {
			t.Errorf("expected symlinked .issuegraph directory to be valid, got: %v", err)
		}
	})

	t.Run(".issuegraph as symlink to file", func(t *testing.T) {
		dir := t.TempDir()
		// Create a file
		actualFile := filepath.Join(t.TempDir(), "actual_file")
		if err := os.WriteFile(actualFile, []byte("test"), 0600); err != nil {
			t.Fatalf("failed to create actual file: %v", err)
		}

		// Create symlink .issuegraph -> file
		symlinkPath := filepath.Join(dir, ".issuegraph")
		if err := os.Symlink(actualFile, symlinkPath); err != nil {
			t.Skipf("symlink creation failed (may not be supported): %v", err)
		}

		err := validateIssueGraphWorkspace(dir)
		if err == nil {
			t.Fatal("expected error when .issuegraph symlink points to a file")
		}
		if got := err.Error(); got != "not a issuegraph workspace: .issuegraph directory not found for "+dir {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run(".issuegraph as broken symlink", func(t *testing.T) {
		dir := t.TempDir()
		// Create symlink to non-existent target
		symlinkPath := filepath.Join(dir, ".issuegraph")
		if err := os.Symlink("/nonexistent/target", symlinkPath); err != nil {
			t.Skipf("symlink creation failed (may not be supported): %v", err)
		}

		err := validateIssueGraphWorkspace(dir)
		if err == nil {
			t.Error("expected error when .issuegraph is a broken symlink")
		}
	})

	t.Run("relative path resolution", func(t *testing.T) {
		dir := setupTestWorkspace(t)
		// Test with relative path
		originalWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get working directory: %v", err)
		}
		defer func() {
			if err := os.Chdir(originalWd); err != nil {
				t.Logf("failed to restore working directory: %v", err)
			}
		}()

		if err := os.Chdir(filepath.Dir(dir)); err != nil {
			t.Fatalf("failed to change directory: %v", err)
		}

		relPath := filepath.Base(dir)
		if err := validateIssueGraphWorkspace(relPath); err != nil {
			t.Errorf("expected relative path to be valid, got: %v", err)
		}
	})

	t.Run("shared worktree fallback", func(t *testing.T) {
		mainRepoDir, worktreeDir := setupSharedWorktreeWorkspace(t)
		mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
		if err := os.MkdirAll(mainIssueGraphDir, 0o755); err != nil {
			t.Fatalf("failed to create main .issuegraph dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(mainIssueGraphDir, "config.yaml"), []byte("json: true\n"), 0o644); err != nil {
			t.Fatalf("failed to write config.yaml: %v", err)
		}

		if err := validateIssueGraphWorkspace(worktreeDir); err != nil {
			t.Errorf("expected shared-worktree workspace to be valid, got: %v", err)
		}
	})
}

// TestGitHooks_EdgeCases tests GitHooks with edge cases
func TestGitHooks_EdgeCases(t *testing.T) {
	// Skip if running as test binary (can't execute issuegraph subcommands)
	skipIfTestBinary(t)

	t.Run("hooks directory does not exist", func(t *testing.T) {
		dir := setupTestGitRepo(t)

		// Verify .git/hooks doesn't exist or remove it
		hooksDir := filepath.Join(dir, ".git", "hooks")
		_ = os.RemoveAll(hooksDir)

		// GitHooks should create the directory via issuegraph hooks install
		err := GitHooks(dir)
		if err != nil {
			t.Errorf("GitHooks should succeed when hooks directory doesn't exist, got: %v", err)
		}

		// Verify hooks directory was created
		if _, err := os.Stat(hooksDir); os.IsNotExist(err) {
			t.Error("expected hooks directory to be created")
		}
	})

	t.Run("git worktree with .git file", func(t *testing.T) {
		// Create main repo
		mainDir := setupTestGitRepo(t)

		// Create a commit so we can create a worktree
		testFile := filepath.Join(mainDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		runGit(t, mainDir, "add", "test.txt")
		runGit(t, mainDir, "commit", "-m", "initial")

		// Create a worktree
		worktreeDir := t.TempDir()
		runGit(t, mainDir, "worktree", "add", worktreeDir, "-b", "feature")

		// Create .issuegraph in worktree
		issuegraphDir := filepath.Join(worktreeDir, ".issuegraph")
		if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
			t.Fatalf("failed to create .issuegraph in worktree: %v", err)
		}

		// GitHooks should work with worktrees where .git is a file
		err := GitHooks(worktreeDir)
		if err != nil {
			t.Errorf("GitHooks should work with git worktrees, got: %v", err)
		}
	})
}

// TestUntrackedJSONL_EdgeCases — removed: UntrackedJSONL function removed (bd-9ni.2)
func TestUntrackedJSONL_EdgeCases(t *testing.T) {
	t.Skip("UntrackedJSONL removed as part of JSONL removal (bd-9ni.2)")
}

// TestPermissions_EdgeCases tests Permissions with edge cases
func TestPermissions_EdgeCases(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping Unix permission/symlink test on Windows")
	}
	t.Run("symbolic link to .issuegraph directory", func(t *testing.T) {
		dir := t.TempDir()

		// Create actual .issuegraph directory elsewhere
		actualIssueGraphDir := filepath.Join(t.TempDir(), "actual-issuegraph")
		if err := os.MkdirAll(actualIssueGraphDir, 0755); err != nil {
			t.Fatalf("failed to create actual .issuegraph: %v", err)
		}

		// Create symlink to it
		symlinkPath := filepath.Join(dir, ".issuegraph")
		if err := os.Symlink(actualIssueGraphDir, symlinkPath); err != nil {
			t.Fatalf("failed to create symlink: %v", err)
		}

		// Permissions should skip symlinked directories
		err := Permissions(dir)
		if err != nil {
			t.Errorf("expected no error for symlinked .issuegraph, got: %v", err)
		}

		// Verify target directory permissions were not changed
		info, err := os.Stat(actualIssueGraphDir)
		if err != nil {
			t.Fatalf("failed to stat actual .issuegraph: %v", err)
		}

		// Should still have 0755, not 0700
		if info.Mode().Perm() == 0700 {
			t.Error("symlinked directory permissions should not be changed to 0700")
		}
	})

	t.Run("symbolic link to database file", func(t *testing.T) {
		dir := setupTestWorkspace(t)

		// Create actual database file elsewhere
		actualDbPath := filepath.Join(t.TempDir(), "actual-issuegraph.db")
		if err := os.WriteFile(actualDbPath, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create actual db: %v", err)
		}

		// Create symlink to it
		dbSymlinkPath := filepath.Join(dir, ".issuegraph", "issuegraph.db")
		if err := os.Symlink(actualDbPath, dbSymlinkPath); err != nil {
			t.Fatalf("failed to create symlink: %v", err)
		}

		// Permissions should skip symlinked files
		err := Permissions(dir)
		if err != nil {
			t.Errorf("expected no error for symlinked db, got: %v", err)
		}

		// Verify target file permissions were not changed
		info, err := os.Stat(actualDbPath)
		if err != nil {
			t.Fatalf("failed to stat actual db: %v", err)
		}

		// Should still have 0644, not 0600
		if info.Mode().Perm() == 0600 {
			t.Error("symlinked database permissions should not be changed to 0600")
		}
	})

	t.Run("fixes incorrect .issuegraph directory permissions", func(t *testing.T) {
		dir := setupTestWorkspace(t)

		issuegraphDir := filepath.Join(dir, ".issuegraph")

		// Set incorrect permissions (too permissive)
		if err := os.Chmod(issuegraphDir, 0755); err != nil {
			t.Fatalf("failed to set permissions: %v", err)
		}

		err := Permissions(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify permissions were fixed to 0700
		info, err := os.Stat(issuegraphDir)
		if err != nil {
			t.Fatalf("failed to stat .issuegraph: %v", err)
		}

		if info.Mode().Perm() != 0700 {
			t.Errorf("expected permissions 0700, got %o", info.Mode().Perm())
		}
	})

	t.Run("fixes incorrect database file permissions", func(t *testing.T) {
		dir := setupTestWorkspace(t)

		dbPath := filepath.Join(dir, ".issuegraph", "issuegraph.db")
		if err := os.WriteFile(dbPath, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create db: %v", err)
		}

		err := Permissions(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify permissions were fixed to 0600
		info, err := os.Stat(dbPath)
		if err != nil {
			t.Fatalf("failed to stat db: %v", err)
		}

		if info.Mode().Perm() != 0600 {
			t.Errorf("expected permissions 0600, got %o", info.Mode().Perm())
		}
	})

	t.Run("handles missing database file gracefully", func(t *testing.T) {
		dir := setupTestWorkspace(t)

		// No database file exists
		err := Permissions(dir)
		if err != nil {
			t.Errorf("expected no error when database doesn't exist, got: %v", err)
		}
	})
}

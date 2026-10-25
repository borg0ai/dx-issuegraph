package doctor

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCheckInstallation(t *testing.T) {
	t.Run("missing issuegraph directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		check := CheckInstallation(tmpDir)

		if check.Status != StatusError {
			t.Errorf("expected StatusError, got %s", check.Status)
		}
		if check.Name != "Installation" {
			t.Errorf("expected name 'Installation', got %s", check.Name)
		}
	})
}

func TestCheckPermissions(t *testing.T) {
	t.Run("no issuegraph directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		check := CheckPermissions(tmpDir)

		// Should return error when .issuegraph dir doesn't exist (can't write to it)
		if check.Status != StatusError {
			t.Errorf("expected StatusError for missing dir, got %s", check.Status)
		}
	})

	t.Run("writable directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
		if err := os.Mkdir(issuegraphDir, 0755); err != nil {
			t.Fatal(err)
		}

		check := CheckPermissions(tmpDir)

		if check.Status != StatusOK {
			t.Errorf("expected StatusOK for writable dir, got %s", check.Status)
		}
	})

	t.Run("bare parent worktree fallback", func(t *testing.T) {
		bareDir, worktreeDir := setupDoctorBareParentWorktree(t)
		bareIssueGraphDir := filepath.Join(bareDir, ".issuegraph")
		if err := os.MkdirAll(bareIssueGraphDir, 0755); err != nil {
			t.Fatal(err)
		}

		check := CheckPermissions(worktreeDir)
		if check.Status != StatusOK {
			t.Fatalf("expected StatusOK for bare-parent fallback, got %s: %s", check.Status, check.Message)
		}
	})

	t.Run("with store bare parent worktree fallback", func(t *testing.T) {
		clearResolveIssueGraphDirCache()
		t.Cleanup(clearResolveIssueGraphDirCache)

		bareDir, worktreeDir := setupDoctorBareParentWorktree(t)
		bareIssueGraphDir := filepath.Join(bareDir, ".issuegraph")
		if err := os.MkdirAll(bareIssueGraphDir, 0o755); err != nil {
			t.Fatal(err)
		}

		ss := &SharedStore{issuegraphDir: bareIssueGraphDir}
		check := CheckPermissionsWithStore(worktreeDir, ss)

		if check.Status != StatusOK {
			t.Fatalf("expected StatusOK for bare-parent fallback with store, got %s: %s", check.Status, check.Message)
		}
		if _, err := os.Stat(filepath.Join(worktreeDir, ".issuegraph")); !os.IsNotExist(err) {
			t.Fatalf("expected no local .issuegraph in worktree, got err=%v", err)
		}
	})
}

func TestCheckPermissions_TempFileUsesSecurePerms(t *testing.T) {
	// Verify that the temporary write-test file created by CheckPermissions
	// uses 0600 (not 0644) so it doesn't expose data to other users.
	if runtime.GOOS == "windows" {
		t.Skip("Skipping file permissions test on Windows")
	}

	tmpDir := t.TempDir()
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.Mkdir(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Run the check — it creates and deletes a temp file
	check := CheckPermissions(tmpDir)
	if check.Status != StatusOK {
		t.Fatalf("expected StatusOK, got %s: %s", check.Status, check.Message)
	}

	// The temp file should be cleaned up
	testFile := filepath.Join(issuegraphDir, ".doctor-test-write")
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("CheckPermissions should clean up its temp file after use")
	}
}

func TestCheckPermissions_CleansUpOnSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.Mkdir(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}

	check := CheckPermissions(tmpDir)
	if check.Status != StatusOK {
		t.Fatalf("expected StatusOK, got %s", check.Status)
	}

	// Verify the temp file does not remain on disk
	testFile := filepath.Join(issuegraphDir, ".doctor-test-write")
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Errorf("temp file %s should be removed after CheckPermissions", testFile)
	}
}

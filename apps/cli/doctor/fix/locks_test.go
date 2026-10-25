package fix

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStaleLockFiles(t *testing.T) {
	t.Run("no issuegraph dir", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := StaleLockFiles(tmpDir); err != nil {
			t.Errorf("expected no error for missing .issuegraph dir, got %v", err)
		}
	})

	t.Run("no lock files", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(tmpDir, ".issuegraph"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := StaleLockFiles(tmpDir); err != nil {
			t.Errorf("expected no error for empty .issuegraph dir, got %v", err)
		}
	})

	t.Run("stale bootstrap lock still removed", func(t *testing.T) {
		// Verify we didn't break existing cleanup
		tmpDir := t.TempDir()
		issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
		if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
			t.Fatal(err)
		}

		lockPath := filepath.Join(issuegraphDir, "dolt.bootstrap.lock")
		if err := os.WriteFile(lockPath, []byte("lock"), 0600); err != nil {
			t.Fatal(err)
		}
		oldTime := time.Now().Add(-10 * time.Minute)
		if err := os.Chtimes(lockPath, oldTime, oldTime); err != nil {
			t.Fatal(err)
		}

		if err := StaleLockFiles(tmpDir); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
			t.Error("stale bootstrap lock should be removed")
		}
	})

	t.Run("shared worktree stale lock is removed from shared issuegraph dir", func(t *testing.T) {
		mainRepoDir, worktreeDir := setupSharedWorktreeWorkspace(t)
		sharedIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
		if err := os.MkdirAll(sharedIssueGraphDir, 0o755); err != nil {
			t.Fatal(err)
		}

		lockPath := filepath.Join(sharedIssueGraphDir, "dolt.bootstrap.lock")
		if err := os.WriteFile(lockPath, []byte("lock"), 0o600); err != nil {
			t.Fatal(err)
		}
		oldTime := time.Now().Add(-10 * time.Minute)
		if err := os.Chtimes(lockPath, oldTime, oldTime); err != nil {
			t.Fatal(err)
		}

		if err := StaleLockFiles(worktreeDir); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
			t.Error("shared stale bootstrap lock should be removed")
		}
	})
}

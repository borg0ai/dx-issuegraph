package fix

import (
	"os"
	"path/filepath"
	"testing"
)

func resolvePathForTest(t *testing.T, path string) string {
	t.Helper()

	resolved, err := filepath.EvalSymlinks(filepath.Clean(path))
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) failed: %v", path, err)
	}

	return resolved
}

func absPathForTest(t *testing.T, path string) string {
	t.Helper()

	absPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("Abs(%q) failed: %v", path, err)
	}

	return absPath
}

func TestSafeWorkspacePath(t *testing.T) {
	root := t.TempDir()
	absEscape, _ := filepath.Abs(filepath.Join(root, "..", "escape"))

	tests := []struct {
		name    string
		relPath string
		wantErr bool
	}{
		{
			name:    "normal relative path",
			relPath: ".issuegraph/issues.jsonl",
			wantErr: false,
		},
		{
			name:    "nested relative path",
			relPath: filepath.Join(".issuegraph", "nested", "file.txt"),
			wantErr: false,
		},
		{
			name:    "absolute path rejected",
			relPath: absEscape,
			wantErr: true,
		},
		{
			name:    "path traversal rejected",
			relPath: filepath.Join("..", "escape"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := safeWorkspacePath(root, tt.relPath)
			if (err != nil) != tt.wantErr {
				t.Fatalf("safeWorkspacePath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				if !isWithinWorkspace(root, got) {
					t.Fatalf("resolved path %q not within workspace %q", got, root)
				}
				if !filepath.IsAbs(got) {
					t.Fatalf("resolved path is not absolute: %q", got)
				}
			}
		})
	}
}

func TestResolvedWorkspaceIssueGraphDir(t *testing.T) {
	t.Run("local workspace returns local issuegraph dir", func(t *testing.T) {
		dir := setupTestWorkspace(t)

		got, err := resolvedWorkspaceIssueGraphDir(dir)
		if err != nil {
			t.Fatalf("resolvedWorkspaceIssueGraphDir() error = %v", err)
		}

		want := resolvePathForTest(t, filepath.Join(dir, ".issuegraph"))
		if got != want {
			t.Fatalf("resolvedWorkspaceIssueGraphDir() = %q, want %q", got, want)
		}
	})

	t.Run("shared worktree falls back to main repo issuegraph dir", func(t *testing.T) {
		mainRepoDir, worktreeDir := setupSharedWorktreeWorkspace(t)
		mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
		if err := os.MkdirAll(mainIssueGraphDir, 0o755); err != nil {
			t.Fatalf("failed to create main .issuegraph dir: %v", err)
		}

		got, err := resolvedWorkspaceIssueGraphDir(worktreeDir)
		if err != nil {
			t.Fatalf("resolvedWorkspaceIssueGraphDir() error = %v", err)
		}

		want := resolvePathForTest(t, mainIssueGraphDir)
		if got != want {
			t.Fatalf("resolvedWorkspaceIssueGraphDir() = %q, want %q", got, mainIssueGraphDir)
		}
	})

	t.Run("worktree local issuegraph dir takes precedence over shared fallback", func(t *testing.T) {
		mainRepoDir, worktreeDir := setupSharedWorktreeWorkspace(t)
		mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
		worktreeIssueGraphDir := filepath.Join(worktreeDir, ".issuegraph")
		if err := os.MkdirAll(mainIssueGraphDir, 0o755); err != nil {
			t.Fatalf("failed to create main .issuegraph dir: %v", err)
		}
		if err := os.MkdirAll(worktreeIssueGraphDir, 0o755); err != nil {
			t.Fatalf("failed to create worktree .issuegraph dir: %v", err)
		}

		got, err := resolvedWorkspaceIssueGraphDir(worktreeDir)
		if err != nil {
			t.Fatalf("resolvedWorkspaceIssueGraphDir() error = %v", err)
		}

		want := resolvePathForTest(t, worktreeIssueGraphDir)
		if got != want {
			t.Fatalf("resolvedWorkspaceIssueGraphDir() = %q, want %q", got, worktreeIssueGraphDir)
		}
	})
}

func TestLocalWorkspaceIssueGraphDir(t *testing.T) {
	t.Run("returns local issuegraph path for local workspace", func(t *testing.T) {
		dir := setupTestWorkspace(t)

		got, err := localWorkspaceIssueGraphDir(dir)
		if err != nil {
			t.Fatalf("localWorkspaceIssueGraphDir() error = %v", err)
		}

		want := filepath.Join(absPathForTest(t, dir), ".issuegraph")
		if got != want {
			t.Fatalf("localWorkspaceIssueGraphDir() = %q, want %q", got, want)
		}
	})

	t.Run("shared worktree still returns worktree local path", func(t *testing.T) {
		_, worktreeDir := setupSharedWorktreeWorkspace(t)

		got, err := localWorkspaceIssueGraphDir(worktreeDir)
		if err != nil {
			t.Fatalf("localWorkspaceIssueGraphDir() error = %v", err)
		}

		want := filepath.Join(absPathForTest(t, worktreeDir), ".issuegraph")
		if got != want {
			t.Fatalf("localWorkspaceIssueGraphDir() = %q, want %q", got, want)
		}
	})
}

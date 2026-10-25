//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRepairWorktreeIssueGraphPermissions is the regression for GH#3593: after worktree
// creation, permissive .issuegraph/ (e.g. 0755 from checkout + umask) must be repaired to 0700.
func TestRepairWorktreeIssueGraphPermissions(t *testing.T) {
	tmp := t.TempDir()
	worktreePath := filepath.Join(tmp, "demo-wt")
	issuegraphDir := filepath.Join(worktreePath, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0o755); err != nil {
		t.Fatalf("mkdir .issuegraph: %v", err)
	}
	if err := os.Chmod(issuegraphDir, 0o755); err != nil {
		t.Fatalf("chmod .issuegraph: %v", err)
	}

	saveJSON := jsonOutput
	jsonOutput = true // suppress stderr from repair helper
	t.Cleanup(func() { jsonOutput = saveJSON })

	repairWorktreeIssueGraphPermissions(worktreePath)

	info, err := os.Stat(issuegraphDir)
	if err != nil {
		t.Fatalf("stat .issuegraph: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Fatalf(".issuegraph permissions = %04o, want 0700", got)
	}
}

func TestRepairWorktreeIssueGraphPermissionsRejectsSymlink(t *testing.T) {
	tmp := t.TempDir()
	worktreePath := filepath.Join(tmp, "demo-wt")
	if err := os.Mkdir(worktreePath, 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}
	target := filepath.Join(tmp, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := os.Chmod(target, 0o755); err != nil {
		t.Fatalf("chmod target: %v", err)
	}
	if err := os.Symlink(target, filepath.Join(worktreePath, ".issuegraph")); err != nil {
		t.Fatalf("symlink .issuegraph: %v", err)
	}

	suppressRepairOutput(t)
	repairWorktreeIssueGraphPermissions(worktreePath)

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat target: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o755 {
		t.Fatalf("symlink target permissions = %04o, want unchanged 0755", got)
	}
}

func TestRepairWorktreeIssueGraphPermissionsRejectsNonDirectory(t *testing.T) {
	tmp := t.TempDir()
	worktreePath := filepath.Join(tmp, "demo-wt")
	if err := os.Mkdir(worktreePath, 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}
	issuegraphPath := filepath.Join(worktreePath, ".issuegraph")
	if err := os.WriteFile(issuegraphPath, []byte("not a directory"), 0o755); err != nil {
		t.Fatalf("write .issuegraph: %v", err)
	}
	if err := os.Chmod(issuegraphPath, 0o755); err != nil {
		t.Fatalf("chmod .issuegraph: %v", err)
	}

	suppressRepairOutput(t)
	repairWorktreeIssueGraphPermissions(worktreePath)

	info, err := os.Stat(issuegraphPath)
	if err != nil {
		t.Fatalf("stat .issuegraph: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o755 {
		t.Fatalf("non-directory .issuegraph permissions = %04o, want unchanged 0755", got)
	}
}

func suppressRepairOutput(t *testing.T) {
	t.Helper()
	saveJSON := jsonOutput
	jsonOutput = true
	t.Cleanup(func() { jsonOutput = saveJSON })
}

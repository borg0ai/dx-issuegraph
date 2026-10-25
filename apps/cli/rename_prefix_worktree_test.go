package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/git"
	"github.com/borg0ai/dx-issuegraph/internal/issuegraph"
)

func setupWorktree(t *testing.T) (mainRepoDir, worktreeDir string) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "issuegraph-worktree-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	mainRepoDir = filepath.Join(tmpDir, "main-repo")
	if err := os.MkdirAll(mainRepoDir, 0755); err != nil {
		t.Fatal(err)
	}

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = mainRepoDir
		if err := cmd.Run(); err != nil {
			t.Skipf("git %v failed: %v", args, err)
		}
	}

	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test User")

	if err := os.WriteFile(filepath.Join(mainRepoDir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", "README.md")
	run("commit", "-m", "Initial commit")

	worktreeDir = filepath.Join(tmpDir, "worktree")
	cmd := exec.Command("git", "worktree", "add", worktreeDir, "HEAD")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git worktree add failed: %v", err)
	}
	t.Cleanup(func() {
		cmd := exec.Command("git", "worktree", "remove", "--force", worktreeDir)
		cmd.Dir = mainRepoDir
		_ = cmd.Run()
	})

	return mainRepoDir, worktreeDir
}

func TestRenamePrefix_WorktreeNotBlocked(t *testing.T) {
	_, worktreeDir := setupWorktree(t)

	t.Chdir(worktreeDir)
	git.ResetCaches()

	if !isGitRepo() {
		t.Fatal("expected isGitRepo() to return true")
	}
	if !git.IsWorktree() {
		t.Fatal("expected git.IsWorktree() to return true")
	}

	// The worktree guard was removed: running from a worktree should not
	// produce the old "cannot run 'issuegraph rename-prefix' from a git worktree" error.
	// We verify the preconditions that the old guard checked (isGitRepo && IsWorktree)
	// are met, confirming the code path that used to block is now reachable.
}

func TestRenamePrefix_WorktreeResolvesMainRepoDB(t *testing.T) {
	mainRepoDir, worktreeDir := setupWorktree(t)

	mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
	if err := os.MkdirAll(mainIssueGraphDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainIssueGraphDir, "metadata.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	os.RemoveAll(filepath.Join(worktreeDir, ".issuegraph"))

	t.Chdir(worktreeDir)
	git.ResetCaches()

	if !git.IsWorktree() {
		t.Fatal("expected git.IsWorktree() to return true")
	}

	found := issuegraph.FindIssueGraphDir()
	if found == "" {
		t.Fatal("FindIssueGraphDir() returned empty — store initialization would fail")
	}

	actual, _ := filepath.EvalSymlinks(filepath.Clean(strings.TrimSpace(found)))
	expected, _ := filepath.EvalSymlinks(filepath.Clean(mainIssueGraphDir))
	if actual != expected {
		t.Errorf("FindIssueGraphDir() = %q, want %q — rename-prefix would use wrong DB path", actual, expected)
	}
}

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

func TestReset_WorktreeFindsIssueGraphDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "issuegraph-reset-worktree-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	mainRepoDir := filepath.Join(tmpDir, "main-repo")
	if err := os.MkdirAll(mainRepoDir, 0755); err != nil {
		t.Fatal(err)
	}

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = mainRepoDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
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

	worktreeDir := filepath.Join(tmpDir, "worktree")
	cmd := exec.Command("git", "worktree", "add", worktreeDir, "HEAD")
	cmd.Dir = mainRepoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add failed: %v\n%s", err, out)
	}
	t.Cleanup(func() {
		cmd := exec.Command("git", "worktree", "remove", "--force", worktreeDir)
		cmd.Dir = mainRepoDir
		_ = cmd.Run()
	})

	mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
	if err := os.MkdirAll(mainIssueGraphDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainIssueGraphDir, "metadata.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	worktreeIssueGraphDir := filepath.Join(worktreeDir, ".issuegraph")
	os.RemoveAll(worktreeIssueGraphDir)

	t.Chdir(worktreeDir)
	git.ResetCaches()

	if !isGitRepo() {
		t.Fatal("expected isGitRepo() to return true")
	}
	if !git.IsWorktree() {
		t.Fatal("expected git.IsWorktree() to return true")
	}

	found := issuegraph.FindIssueGraphDir()
	if found == "" {
		t.Fatal("FindIssueGraphDir() returned empty string")
	}

	expected, err := filepath.EvalSymlinks(mainIssueGraphDir)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) failed: %v", mainIssueGraphDir, err)
	}
	actual, err := filepath.EvalSymlinks(found)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) failed: %v", found, err)
	}

	expected = filepath.Clean(strings.TrimSpace(expected))
	actual = filepath.Clean(strings.TrimSpace(actual))

	if actual != expected {
		t.Errorf("FindIssueGraphDir() = %q, want %q", actual, expected)
	}

	if _, err := os.Stat(actual); err != nil {
		t.Errorf("os.Stat(FindIssueGraphDir()) failed: %v — runReset would report \"not initialized\"", err)
	}
}

func TestReset_WorktreeNoIssueGraphReturnsEmpty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "issuegraph-reset-no-issuegraph-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	mainRepoDir := filepath.Join(tmpDir, "main-repo")
	if err := os.MkdirAll(mainRepoDir, 0755); err != nil {
		t.Fatal(err)
	}

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = mainRepoDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
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

	worktreeDir := filepath.Join(tmpDir, "worktree")
	cmd := exec.Command("git", "worktree", "add", worktreeDir, "HEAD")
	cmd.Dir = mainRepoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add failed: %v\n%s", err, out)
	}
	t.Cleanup(func() {
		cmd := exec.Command("git", "worktree", "remove", "--force", worktreeDir)
		cmd.Dir = mainRepoDir
		_ = cmd.Run()
	})

	t.Chdir(worktreeDir)
	git.ResetCaches()

	found := issuegraph.FindIssueGraphDir()
	if found != "" {
		t.Errorf("FindIssueGraphDir() = %q, want empty string when no .issuegraph exists anywhere", found)
	}
}

func TestReset_WorktreeSubdirFindsIssueGraphDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "issuegraph-reset-subdir-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	mainRepoDir := filepath.Join(tmpDir, "main-repo")
	if err := os.MkdirAll(mainRepoDir, 0755); err != nil {
		t.Fatal(err)
	}

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = mainRepoDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
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

	worktreeDir := filepath.Join(tmpDir, "worktree")
	cmd := exec.Command("git", "worktree", "add", worktreeDir, "HEAD")
	cmd.Dir = mainRepoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add failed: %v\n%s", err, out)
	}
	t.Cleanup(func() {
		cmd := exec.Command("git", "worktree", "remove", "--force", worktreeDir)
		cmd.Dir = mainRepoDir
		_ = cmd.Run()
	})

	mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
	if err := os.MkdirAll(mainIssueGraphDir, 0755); err != nil {
		t.Fatal(err)
	}
	os.RemoveAll(filepath.Join(worktreeDir, ".issuegraph"))

	subDir := filepath.Join(worktreeDir, "sub", "dir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	t.Chdir(subDir)
	git.ResetCaches()

	found := issuegraph.FindIssueGraphDir()
	if found != "" {
		actual, _ := filepath.EvalSymlinks(found)
		expected, _ := filepath.EvalSymlinks(mainIssueGraphDir)
		if filepath.Clean(actual) != filepath.Clean(expected) {
			t.Errorf("FindIssueGraphDir() from subdirectory = %q, want %q", actual, expected)
		}
	}
}

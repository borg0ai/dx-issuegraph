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

func TestCountExistingIssues_WorktreeFallback(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "issuegraph-worktree-init-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	mainRepoDir := filepath.Join(tmpDir, "main-repo")
	if err := os.MkdirAll(mainRepoDir, 0755); err != nil {
		t.Fatal(err)
	}

	run := func(dir string, args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	run(mainRepoDir, "init")
	run(mainRepoDir, "config", "user.email", "test@example.com")
	run(mainRepoDir, "config", "user.name", "Test User")

	if err := os.WriteFile(filepath.Join(mainRepoDir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run(mainRepoDir, "add", "README.md")
	run(mainRepoDir, "commit", "-m", "Initial commit")

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

	worktreeIssueGraphDir := filepath.Join(worktreeDir, ".issuegraph")
	os.RemoveAll(worktreeIssueGraphDir)

	t.Chdir(worktreeDir)
	git.ResetCaches()

	if !isGitRepo() {
		t.Fatal("expected isGitRepo() to return true in worktree")
	}
	if !git.IsWorktree() {
		t.Fatal("expected git.IsWorktree() to return true in worktree")
	}

	fallback := issuegraph.GetWorktreeFallbackIssueGraphDir()
	fallback = strings.TrimSpace(fallback)
	fallback = filepath.Clean(fallback)
	if fallback == "" {
		t.Fatal("expected GetWorktreeFallbackIssueGraphDir() to return non-empty path in worktree")
	}

	expectedIssueGraph := filepath.Clean(mainIssueGraphDir)

	fallbackResolved, err := filepath.EvalSymlinks(fallback)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) failed: %v", fallback, err)
	}
	expectedResolved, err := filepath.EvalSymlinks(expectedIssueGraph)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) failed: %v", expectedIssueGraph, err)
	}

	if fallbackResolved != expectedResolved {
		t.Errorf("GetWorktreeFallbackIssueGraphDir() = %q (resolved: %q), want %q (resolved: %q)",
			fallback, fallbackResolved, expectedIssueGraph, expectedResolved)
	}
}

func TestCountExistingIssues_WorktreeLocalIssueGraphPreferred(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "issuegraph-worktree-init-local-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	mainRepoDir := filepath.Join(tmpDir, "main-repo")
	if err := os.MkdirAll(mainRepoDir, 0755); err != nil {
		t.Fatal(err)
	}

	run := func(dir string, args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
	run(mainRepoDir, "init")
	run(mainRepoDir, "config", "user.email", "test@example.com")
	run(mainRepoDir, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(mainRepoDir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run(mainRepoDir, "add", "README.md")
	run(mainRepoDir, "commit", "-m", "Initial commit")

	worktreeDir := filepath.Join(tmpDir, "worktree")
	cmd := exec.Command("git", "worktree", "add", worktreeDir, "HEAD")
	cmd.Dir = mainRepoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add failed: %v\n%s", err, out)
	}
	t.Cleanup(func() {
		cleanupCmd := exec.Command("git", "worktree", "remove", "--force", worktreeDir)
		cleanupCmd.Dir = mainRepoDir
		_ = cleanupCmd.Run()
	})

	mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
	worktreeIssueGraphDir := filepath.Join(worktreeDir, ".issuegraph")
	if err := os.MkdirAll(mainIssueGraphDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(worktreeIssueGraphDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worktreeIssueGraphDir, "metadata.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(worktreeDir)
	git.ResetCaches()

	if !git.IsWorktree() {
		t.Fatal("expected git.IsWorktree() to return true")
	}

	found := issuegraph.FindIssueGraphDir()
	if found == "" {
		t.Fatal("FindIssueGraphDir() returned empty")
	}

	foundResolved, _ := filepath.EvalSymlinks(filepath.Clean(found))
	worktreeResolved, _ := filepath.EvalSymlinks(filepath.Clean(worktreeIssueGraphDir))

	if foundResolved != worktreeResolved {
		t.Errorf("FindIssueGraphDir() = %q, want %q (local worktree .issuegraph should take precedence over main repo fallback)",
			foundResolved, worktreeResolved)
	}
}

func TestCountExistingIssues_WorktreeNoIssueGraphAnywhere(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "issuegraph-worktree-init-empty-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	mainRepoDir := filepath.Join(tmpDir, "main-repo")
	if err := os.MkdirAll(mainRepoDir, 0755); err != nil {
		t.Fatal(err)
	}

	run := func(dir string, args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
	run(mainRepoDir, "init")
	run(mainRepoDir, "config", "user.email", "test@example.com")
	run(mainRepoDir, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(mainRepoDir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run(mainRepoDir, "add", "README.md")
	run(mainRepoDir, "commit", "-m", "Initial commit")

	worktreeDir := filepath.Join(tmpDir, "worktree")
	cmd := exec.Command("git", "worktree", "add", worktreeDir, "HEAD")
	cmd.Dir = mainRepoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add failed: %v\n%s", err, out)
	}
	t.Cleanup(func() {
		cleanupCmd := exec.Command("git", "worktree", "remove", "--force", worktreeDir)
		cleanupCmd.Dir = mainRepoDir
		_ = cleanupCmd.Run()
	})

	os.RemoveAll(filepath.Join(worktreeDir, ".issuegraph"))

	t.Chdir(worktreeDir)
	git.ResetCaches()

	found := issuegraph.FindIssueGraphDir()
	if found != "" {
		t.Errorf("FindIssueGraphDir() = %q, want empty string when no .issuegraph exists anywhere", found)
	}
}

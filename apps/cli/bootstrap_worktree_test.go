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

func TestBootstrap_WorktreeFallbackDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "issuegraph-bootstrap-worktree-test-*")
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
	if err := os.MkdirAll(filepath.Join(mainIssueGraphDir, "dolt"), 0755); err != nil {
		t.Fatal(err)
	}

	worktreeIssueGraphDir := filepath.Join(worktreeDir, ".issuegraph")
	_ = os.RemoveAll(worktreeIssueGraphDir)

	t.Chdir(worktreeDir)
	git.ResetCaches()

	if !git.IsWorktree() {
		t.Fatal("expected git.IsWorktree() to return true in worktree")
	}

	got := issuegraph.GetWorktreeFallbackIssueGraphDir()
	if got == "" {
		t.Fatal("GetWorktreeFallbackIssueGraphDir() returned empty string")
	}

	gotClean := filepath.Clean(strings.TrimSpace(got))
	wantClean := filepath.Clean(strings.TrimSpace(mainIssueGraphDir))

	gotResolved, err := filepath.EvalSymlinks(gotClean)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) failed: %v", gotClean, err)
	}
	wantResolved, err := filepath.EvalSymlinks(wantClean)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) failed: %v", wantClean, err)
	}

	if gotResolved != wantResolved {
		t.Errorf("GetWorktreeFallbackIssueGraphDir() = %q (resolved %q), want %q (resolved %q)",
			gotClean, gotResolved, wantClean, wantResolved)
	}

	cwd, _ := os.Getwd()
	cwdIssueGraph := filepath.Join(cwd, ".issuegraph")
	cwdResolved, _ := filepath.EvalSymlinks(filepath.Clean(cwdIssueGraph))
	if gotResolved == cwdResolved {
		t.Errorf("GetWorktreeFallbackIssueGraphDir() returned CWD-based path %q, expected main repo path", gotResolved)
	}
}

func TestBootstrap_WorktreeLocalIssueGraphPreferLocal(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "issuegraph-bootstrap-local-test-*")
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
	if err := os.MkdirAll(filepath.Join(mainIssueGraphDir, "dolt"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(worktreeIssueGraphDir, "dolt"), 0755); err != nil {
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
	worktreeIssueGraphResolved, _ := filepath.EvalSymlinks(filepath.Clean(worktreeIssueGraphDir))

	if foundResolved != worktreeIssueGraphResolved {
		t.Errorf("FindIssueGraphDir() = %q, want %q (local worktree .issuegraph should take precedence over main repo fallback)",
			foundResolved, worktreeIssueGraphResolved)
	}
}

func TestBootstrap_WorktreeNoIssueGraphAnywhereStillPointsToMainRepo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "issuegraph-bootstrap-empty-test-*")
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

	fallback := issuegraph.GetWorktreeFallbackIssueGraphDir()
	if fallback == "" {
		t.Fatal("GetWorktreeFallbackIssueGraphDir() returned empty")
	}

	mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
	fallbackResolved, _ := filepath.EvalSymlinks(filepath.Clean(fallback))
	mainResolved, _ := filepath.EvalSymlinks(filepath.Clean(mainIssueGraphDir))
	if fallbackResolved != mainResolved {
		t.Errorf("GetWorktreeFallbackIssueGraphDir() = %q, want %q (should resolve to main repo even without .issuegraph)", fallbackResolved, mainResolved)
	}
}

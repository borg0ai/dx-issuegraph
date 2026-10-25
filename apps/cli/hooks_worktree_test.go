package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/git"
)

func TestConfigureIssueGraphHooksPath_WorktreeUsesMainRepo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "issuegraph-hooks-worktree-test-*")
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

	worktreeDir := filepath.Join(tmpDir, "worktree")
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

	if err := os.MkdirAll(filepath.Join(mainRepoDir, ".issuegraph", "hooks"), 0755); err != nil {
		t.Fatal(err)
	}

	t.Chdir(worktreeDir)
	git.ResetCaches()

	if !git.IsWorktree() {
		t.Fatal("expected git.IsWorktree() to return true")
	}

	if err := configureIssueGraphHooksPath(); err != nil {
		t.Fatalf("configureIssueGraphHooksPath failed: %v", err)
	}

	cmd = exec.Command("git", "config", "--get", "core.hooksPath")
	cmd.Dir = mainRepoDir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git config --get core.hooksPath failed: %v", err)
	}
	hooksPath := filepath.Clean(strings.TrimSpace(string(out)))
	expected := filepath.Join(mainRepoDir, ".issuegraph", "hooks")
	hooksPath, _ = filepath.EvalSymlinks(hooksPath)
	expected, _ = filepath.EvalSymlinks(expected)
	if hooksPath != expected {
		t.Errorf("core.hooksPath = %q, want %q", hooksPath, expected)
	}
}

func TestConfigureSharedHooksPath_WorktreeUsesMainRepo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "issuegraph-shared-hooks-worktree-test-*")
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

	worktreeDir := filepath.Join(tmpDir, "worktree")
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

	if err := os.MkdirAll(filepath.Join(mainRepoDir, ".issuegraph-hooks"), 0755); err != nil {
		t.Fatal(err)
	}

	t.Chdir(worktreeDir)
	git.ResetCaches()

	if !git.IsWorktree() {
		t.Fatal("expected git.IsWorktree() to return true")
	}

	if err := configureSharedHooksPath(); err != nil {
		t.Fatalf("configureSharedHooksPath failed: %v", err)
	}

	cmd = exec.Command("git", "config", "--get", "core.hooksPath")
	cmd.Dir = mainRepoDir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git config --get core.hooksPath failed: %v", err)
	}
	hooksPath := filepath.Clean(strings.TrimSpace(string(out)))
	expected := filepath.Join(mainRepoDir, ".issuegraph-hooks")
	hooksPath, _ = filepath.EvalSymlinks(hooksPath)
	expected, _ = filepath.EvalSymlinks(expected)
	if hooksPath != expected {
		t.Errorf("core.hooksPath = %q, want %q", hooksPath, expected)
	}
}

func TestResetHooksPathIfIssueGraphManaged_Worktree(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "issuegraph-reset-hooks-worktree-test-*")
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

	worktreeDir := filepath.Join(tmpDir, "worktree")
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

	if err := os.MkdirAll(filepath.Join(mainRepoDir, ".issuegraph", "hooks"), 0755); err != nil {
		t.Fatal(err)
	}

	hooksPathToSet := filepath.Join(mainRepoDir, ".issuegraph", "hooks")
	evaluated, _ := filepath.EvalSymlinks(hooksPathToSet)
	if evaluated != "" {
		hooksPathToSet = evaluated
	}
	cmd = exec.Command("git", "config", "core.hooksPath", hooksPathToSet)
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config core.hooksPath failed: %v", err)
	}

	t.Chdir(worktreeDir)
	git.ResetCaches()

	if !git.IsWorktree() {
		t.Fatal("expected git.IsWorktree() to return true")
	}

	if err := resetHooksPathIfIssueGraphManaged(); err != nil {
		t.Fatalf("resetHooksPathIfIssueGraphManaged failed: %v", err)
	}

	cmd = exec.Command("git", "config", "--get", "core.hooksPath")
	cmd.Dir = mainRepoDir
	out, _ := cmd.Output()
	if strings.TrimSpace(string(out)) != "" {
		t.Errorf("core.hooksPath = %q after reset, want empty", strings.TrimSpace(string(out)))
	}
}

func TestConfigureIssueGraphHooksPath_NormalRepoUnchanged(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "issuegraph-hooks-normal-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			t.Skipf("git %v failed: %v", args, err)
		}
	}
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", "README.md")
	run("commit", "-m", "Initial commit")

	if err := os.MkdirAll(filepath.Join(repoDir, ".issuegraph", "hooks"), 0755); err != nil {
		t.Fatal(err)
	}

	t.Chdir(repoDir)
	git.ResetCaches()

	if git.IsWorktree() {
		t.Fatal("expected git.IsWorktree() to return false in normal repo")
	}

	if err := configureIssueGraphHooksPath(); err != nil {
		t.Fatalf("configureIssueGraphHooksPath failed: %v", err)
	}

	cmd := exec.Command("git", "config", "--get", "core.hooksPath")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git config --get core.hooksPath failed: %v", err)
	}
	hooksPath := filepath.Clean(strings.TrimSpace(string(out)))
	expected := filepath.Join(repoDir, ".issuegraph", "hooks")
	hooksPath, _ = filepath.EvalSymlinks(hooksPath)
	expected, _ = filepath.EvalSymlinks(expected)
	if hooksPath != expected {
		t.Errorf("core.hooksPath = %q, want %q", hooksPath, expected)
	}
}

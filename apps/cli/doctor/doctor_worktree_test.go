package doctor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupWorktreeRepo(t *testing.T) (mainRepoDir, worktreeDir string) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "issuegraph-doctor-worktree-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	mainRepoDir = filepath.Join(tmpDir, "main-repo")
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

	worktreeDir = filepath.Join(tmpDir, "worktree")
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

	return mainRepoDir, worktreeDir
}

func resolve(t testing.TB, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(filepath.Clean(path))
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", path, err)
	}
	return resolved
}

func TestResolveIssueGraphDirForRepo_WorktreeFallback(t *testing.T) {
	mainRepoDir, worktreeDir := setupWorktreeRepo(t)

	mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
	if err := os.MkdirAll(mainIssueGraphDir, 0755); err != nil {
		t.Fatal(err)
	}
	os.Remove(filepath.Join(worktreeDir, ".issuegraph"))

	clearResolveIssueGraphDirCache()

	got := resolve(t, ResolveIssueGraphDirForRepo(worktreeDir))
	want := resolve(t, mainIssueGraphDir)

	if got != want {
		t.Errorf("ResolveIssueGraphDirForRepo(worktree) = %q, want %q", got, want)
	}
}

func TestResolveIssueGraphDirForRepo_WorktreeLocalIssueGraphPreferred(t *testing.T) {
	mainRepoDir, worktreeDir := setupWorktreeRepo(t)

	mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
	worktreeIssueGraphDir := filepath.Join(worktreeDir, ".issuegraph")
	if err := os.MkdirAll(mainIssueGraphDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(worktreeIssueGraphDir, 0755); err != nil {
		t.Fatal(err)
	}

	clearResolveIssueGraphDirCache()

	got := resolve(t, ResolveIssueGraphDirForRepo(worktreeDir))
	want := resolve(t, worktreeIssueGraphDir)

	if got != want {
		t.Errorf("ResolveIssueGraphDirForRepo(worktree) = %q, want %q (local .issuegraph should take precedence)", got, want)
	}
}

func TestResolveIssueGraphDirForRepo_WorktreeNoIssueGraphAnywhere(t *testing.T) {
	_, worktreeDir := setupWorktreeRepo(t)

	os.Remove(filepath.Join(worktreeDir, ".issuegraph"))

	clearResolveIssueGraphDirCache()

	got := ResolveIssueGraphDirForRepo(worktreeDir)

	gotClean := filepath.Clean(strings.TrimSpace(got))
	wantIssueGraph := filepath.Join(worktreeDir, ".issuegraph")
	wantClean := filepath.Clean(wantIssueGraph)

	gotResolved, _ := filepath.EvalSymlinks(gotClean)
	wantResolved, _ := filepath.EvalSymlinks(wantClean)

	if gotResolved != wantResolved {
		t.Errorf("ResolveIssueGraphDirForRepo(worktree with no .issuegraph) = %q, want %q (falls back to local path)", gotResolved, wantResolved)
	}
}

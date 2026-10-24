package issuegraph

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/git"
)

// TestFindIssueGraphDir_WorktreeWithInheritedArtifacts covers the case where a
// git worktree inherits the parent repo's tracked .issuegraph/ artifacts —
// metadata.json, config.yaml, issues.jsonl, etc. — but NOT an actual Dolt
// database directory (which is gitignored under .issuegraph/.gitignore). Without
// a real database, the worktree must resolve to the shared .issuegraph/ via the
// git common-dir fallback; otherwise issuegraph would spawn a sidecar Dolt server
// against an empty data directory and fail every query.
//
// Regression test for "issuegraph worktree create --help" promise: "The worktree
// automatically shares the same issuegraph database as the main repository via
// git common directory discovery — no manual redirect configuration needed."
func TestFindIssueGraphDir_WorktreeWithInheritedArtifacts(t *testing.T) {
	originalEnv := os.Getenv("ISSUEGRAPH_DIR")
	defer func() {
		if originalEnv != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalEnv)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	}()
	os.Unsetenv("ISSUEGRAPH_DIR")

	tmpDir, err := os.MkdirTemp("", "issuegraph-inherited-artifacts-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mainRepoDir := filepath.Join(tmpDir, "main-repo")
	if err := os.MkdirAll(mainRepoDir, 0o755); err != nil {
		t.Fatal(err)
	}

	runGit := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
		}
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}
	runGit(mainRepoDir, "config", "user.email", "test@example.com")
	runGit(mainRepoDir, "config", "user.name", "Test User")

	// Simulate a real parent-repo issuegraph install: metadata.json, config.yaml,
	// issues.jsonl, AND a database. The database is what makes this a real
	// issuegraph project.
	mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
	if err := os.MkdirAll(mainIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(filepath.Join(mainIssueGraphDir, "metadata.json"), `{"database":"dolt","dolt_database":"issuegraph"}`)
	mustWrite(filepath.Join(mainIssueGraphDir, "config.yaml"), "issue-prefix: \"proj\"\n")
	mustWrite(filepath.Join(mainIssueGraphDir, "issues.jsonl"), "")
	mustWrite(filepath.Join(mainIssueGraphDir, "issuegraph.db"), "")

	// Commit the artifact files (NOT the db — gitignore it as real projects do).
	mustWrite(filepath.Join(mainIssueGraphDir, ".gitignore"), "*.db\ndolt/\nembeddeddolt/\n")
	mustWrite(filepath.Join(mainRepoDir, "README.md"), "# Test\n")
	runGit(mainRepoDir, "add", "README.md", ".issuegraph/metadata.json", ".issuegraph/config.yaml", ".issuegraph/issues.jsonl", ".issuegraph/.gitignore")
	runGit(mainRepoDir, "commit", "-m", "initial")

	// Create a worktree by git — `git worktree add` checks out the tracked
	// files, so the worktree's .issuegraph/ ends up with metadata.json + config.yaml
	// + issues.jsonl but no database (since *.db is gitignored).
	worktreeDir := filepath.Join(tmpDir, "worktree")
	runGit(mainRepoDir, "worktree", "add", worktreeDir, "HEAD")
	defer func() {
		cmd := exec.Command("git", "worktree", "remove", "--force", worktreeDir)
		cmd.Dir = mainRepoDir
		_ = cmd.Run()
	}()

	// Sanity-check the shape of the worktree's .issuegraph/.
	worktreeIssueGraphDir := filepath.Join(worktreeDir, ".issuegraph")
	if _, err := os.Stat(filepath.Join(worktreeIssueGraphDir, "metadata.json")); err != nil {
		t.Fatalf("precondition: worktree should have inherited metadata.json, got: %v", err)
	}
	if _, err := os.Stat(filepath.Join(worktreeIssueGraphDir, "issuegraph.db")); err == nil {
		t.Fatalf("precondition: worktree's *.db should be gitignored, but issuegraph.db exists")
	}

	t.Chdir(worktreeDir)
	git.ResetCaches()

	result := FindIssueGraphDir()

	// Expect the shared (main repo's) .issuegraph/ — not the worktree's
	// metadata-only one. Resolve symlinks for macOS /var → /private/var.
	resultResolved, _ := filepath.EvalSymlinks(result)
	mainResolved, _ := filepath.EvalSymlinks(mainIssueGraphDir)
	worktreeResolved, _ := filepath.EvalSymlinks(worktreeIssueGraphDir)

	if resultResolved == worktreeResolved {
		t.Fatalf("FindIssueGraphDir() returned the worktree's inherited-artifacts .issuegraph/ (%q); "+
			"expected the shared .issuegraph/ (%q). This is the sidecar-Dolt-server bug: "+
			"issuegraph should not try to run a database out of a directory that only contains "+
			"tracked metadata files.", result, mainIssueGraphDir)
	}
	if resultResolved != mainResolved {
		t.Errorf("FindIssueGraphDir() = %q, want shared .issuegraph %q", result, mainIssueGraphDir)
	}
}

// TestFindIssueGraphDir_WorktreeSeparateDBPreservesLocal verifies the companion
// case: if a worktree really does own its own database (true separate-DB
// mode), FindIssueGraphDir must continue to return the worktree's .issuegraph/ — the
// inherited-artifacts fix must not regress separate-DB mode.
func TestFindIssueGraphDir_WorktreeSeparateDBPreservesLocal(t *testing.T) {
	originalEnv := os.Getenv("ISSUEGRAPH_DIR")
	defer func() {
		if originalEnv != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalEnv)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	}()
	os.Unsetenv("ISSUEGRAPH_DIR")

	tmpDir, err := os.MkdirTemp("", "issuegraph-separate-db-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mainRepoDir := filepath.Join(tmpDir, "main-repo")
	if err := os.MkdirAll(mainRepoDir, 0o755); err != nil {
		t.Fatal(err)
	}

	runGit := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
		}
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}
	runGit(mainRepoDir, "config", "user.email", "test@example.com")
	runGit(mainRepoDir, "config", "user.name", "Test User")

	// Main repo has its own issuegraph install WITH a database.
	mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
	if err := os.MkdirAll(mainIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainIssueGraphDir, "issuegraph.db"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainRepoDir, "README.md"), []byte("# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(mainRepoDir, "add", "README.md")
	runGit(mainRepoDir, "commit", "-m", "initial")

	worktreeDir := filepath.Join(tmpDir, "worktree")
	runGit(mainRepoDir, "worktree", "add", worktreeDir, "HEAD")
	defer func() {
		cmd := exec.Command("git", "worktree", "remove", "--force", worktreeDir)
		cmd.Dir = mainRepoDir
		_ = cmd.Run()
	}()

	// Worktree has its OWN real database — separate-DB mode.
	worktreeIssueGraphDir := filepath.Join(worktreeDir, ".issuegraph")
	if err := os.MkdirAll(worktreeIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worktreeIssueGraphDir, "issuegraph.db"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(worktreeDir)
	git.ResetCaches()

	result := FindIssueGraphDir()
	resultResolved, _ := filepath.EvalSymlinks(result)
	worktreeResolved, _ := filepath.EvalSymlinks(worktreeIssueGraphDir)

	if resultResolved != worktreeResolved {
		t.Errorf("FindIssueGraphDir() = %q, want worktree .issuegraph %q (separate-DB mode)",
			result, worktreeIssueGraphDir)
	}
}

// TestFindIssueGraphDir_WorktreeSeparateDBPreservesLocalWithDoltDir is the same
// separate-DB regression guard but exercises the `dolt/` directory branch of
// hasIssueGraphDatabase (server mode) rather than the *.db branch.
func TestFindIssueGraphDir_WorktreeSeparateDBPreservesLocalWithDoltDir(t *testing.T) {
	runWorktreeSeparateDBPreservedTest(t, "dolt")
}

// TestFindIssueGraphDir_WorktreeSeparateDBPreservesLocalWithEmbeddedDolt exercises
// the `embeddeddolt/` directory branch of hasIssueGraphDatabase (embedded-engine
// mode) for the separate-DB regression guard.
func TestFindIssueGraphDir_WorktreeSeparateDBPreservesLocalWithEmbeddedDolt(t *testing.T) {
	runWorktreeSeparateDBPreservedTest(t, "embeddeddolt")
}

// runWorktreeSeparateDBPreservedTest is shared by the three separate-DB
// regression tests. databaseMarker names what to create inside the worktree's
// .issuegraph/ so hasIssueGraphDatabase returns true: "dolt" / "embeddeddolt" (directory)
// or "issuegraph.db" (file).
func runWorktreeSeparateDBPreservedTest(t *testing.T, databaseMarker string) {
	t.Helper()

	originalEnv := os.Getenv("ISSUEGRAPH_DIR")
	defer func() {
		if originalEnv != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalEnv)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	}()
	os.Unsetenv("ISSUEGRAPH_DIR")

	tmpDir, err := os.MkdirTemp("", "issuegraph-separate-db-variant-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mainRepoDir := filepath.Join(tmpDir, "main-repo")
	if err := os.MkdirAll(mainRepoDir, 0o755); err != nil {
		t.Fatal(err)
	}

	runGit := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
		}
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}
	runGit(mainRepoDir, "config", "user.email", "test@example.com")
	runGit(mainRepoDir, "config", "user.name", "Test User")

	// Main repo has a real DB so the shared fallback would win if the
	// worktree's own database weren't recognized.
	mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
	if err := os.MkdirAll(mainIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainIssueGraphDir, "issuegraph.db"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainRepoDir, "README.md"), []byte("# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(mainRepoDir, "add", "README.md")
	runGit(mainRepoDir, "commit", "-m", "initial")

	worktreeDir := filepath.Join(tmpDir, "worktree")
	runGit(mainRepoDir, "worktree", "add", worktreeDir, "HEAD")
	defer func() {
		cmd := exec.Command("git", "worktree", "remove", "--force", worktreeDir)
		cmd.Dir = mainRepoDir
		_ = cmd.Run()
	}()

	worktreeIssueGraphDir := filepath.Join(worktreeDir, ".issuegraph")
	if err := os.MkdirAll(worktreeIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	switch databaseMarker {
	case "dolt", "embeddeddolt":
		if err := os.MkdirAll(filepath.Join(worktreeIssueGraphDir, databaseMarker), 0o755); err != nil {
			t.Fatal(err)
		}
	default:
		if err := os.WriteFile(filepath.Join(worktreeIssueGraphDir, databaseMarker), []byte{}, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	t.Chdir(worktreeDir)
	git.ResetCaches()

	result := FindIssueGraphDir()
	resultResolved, _ := filepath.EvalSymlinks(result)
	worktreeResolved, _ := filepath.EvalSymlinks(worktreeIssueGraphDir)

	if resultResolved != worktreeResolved {
		t.Errorf("FindIssueGraphDir() = %q, want worktree .issuegraph %q (separate-DB mode via %q)",
			result, worktreeIssueGraphDir, databaseMarker)
	}
}

// TestFindIssueGraphDir_WorktreeNoDatabaseAnywhereFallsBackToLocal exercises the
// lenient-fallback escape hatch in step 3b: when the worktree has inherited
// metadata but no database AND the shared fallback also has no database,
// FindIssueGraphDir should return the worktree-local .issuegraph/ so a fresh
// `issuegraph init` in the worktree can still bootstrap. If the strict check ran
// unconditionally, a brand-new worktree with no main-repo database would
// return empty and block init.
func TestFindIssueGraphDir_WorktreeNoDatabaseAnywhereFallsBackToLocal(t *testing.T) {
	originalEnv := os.Getenv("ISSUEGRAPH_DIR")
	defer func() {
		if originalEnv != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalEnv)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	}()
	os.Unsetenv("ISSUEGRAPH_DIR")

	tmpDir, err := os.MkdirTemp("", "issuegraph-no-db-anywhere-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mainRepoDir := filepath.Join(tmpDir, "main-repo")
	if err := os.MkdirAll(mainRepoDir, 0o755); err != nil {
		t.Fatal(err)
	}

	runGit := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
		}
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}
	runGit(mainRepoDir, "config", "user.email", "test@example.com")
	runGit(mainRepoDir, "config", "user.name", "Test User")

	// Main repo has a .issuegraph/ with metadata only — NO database. This is the
	// "pre-bd-init" state on both sides.
	mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
	if err := os.MkdirAll(mainIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainIssueGraphDir, "metadata.json"), []byte(`{"database":"dolt"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainIssueGraphDir, ".gitignore"), []byte("*.db\ndolt/\nembeddeddolt/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainRepoDir, "README.md"), []byte("# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(mainRepoDir, "add", "README.md", ".issuegraph/metadata.json", ".issuegraph/.gitignore")
	runGit(mainRepoDir, "commit", "-m", "seed .issuegraph without db")

	worktreeDir := filepath.Join(tmpDir, "worktree")
	runGit(mainRepoDir, "worktree", "add", worktreeDir, "HEAD")
	defer func() {
		cmd := exec.Command("git", "worktree", "remove", "--force", worktreeDir)
		cmd.Dir = mainRepoDir
		_ = cmd.Run()
	}()

	worktreeIssueGraphDir := filepath.Join(worktreeDir, ".issuegraph")
	if _, err := os.Stat(filepath.Join(worktreeIssueGraphDir, "metadata.json")); err != nil {
		t.Fatalf("precondition: worktree should have inherited metadata.json, got: %v", err)
	}

	t.Chdir(worktreeDir)
	git.ResetCaches()

	result := FindIssueGraphDir()
	if result == "" {
		t.Fatal("FindIssueGraphDir returned empty; lenient escape hatch should return the worktree .issuegraph/ when no DB exists anywhere")
	}
	resultResolved, _ := filepath.EvalSymlinks(result)
	worktreeResolved, _ := filepath.EvalSymlinks(worktreeIssueGraphDir)
	if resultResolved != worktreeResolved {
		t.Errorf("FindIssueGraphDir() = %q, want worktree .issuegraph %q (lenient fallback — no DB anywhere)",
			result, worktreeIssueGraphDir)
	}
}

// TestFindDatabasePath_WorktreeServerModeSharesMainRepo verifies that
// FindDatabasePath in a worktree with inherited server-mode metadata.json
// resolves to the main repo's .issuegraph/dolt — NOT the worktree's metadata-only
// .issuegraph/dolt. Without this, each worktree spawns its own dolt sql-server
// against an empty data directory, fails to find the project database, and
// leaves orphaned server processes.
//
// Regression test for the "worktree server mode duplicate server" bug.
func TestFindDatabasePath_WorktreeServerModeSharesMainRepo(t *testing.T) {
	originalEnvDir := os.Getenv("ISSUEGRAPH_DIR")
	originalEnvDB := os.Getenv("ISSUEGRAPH_DB")
	defer func() {
		if originalEnvDir != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalEnvDir)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
		if originalEnvDB != "" {
			os.Setenv("ISSUEGRAPH_DB", originalEnvDB)
		} else {
			os.Unsetenv("ISSUEGRAPH_DB")
		}
	}()
	os.Unsetenv("ISSUEGRAPH_DIR")
	os.Unsetenv("ISSUEGRAPH_DB")

	tmpDir, err := os.MkdirTemp("", "issuegraph-worktree-server-mode-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mainRepoDir := filepath.Join(tmpDir, "main-repo")
	if err := os.MkdirAll(mainRepoDir, 0o755); err != nil {
		t.Fatal(err)
	}

	runGit := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
		}
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}
	runGit(mainRepoDir, "config", "user.email", "test@example.com")
	runGit(mainRepoDir, "config", "user.name", "Test User")

	// Simulate a server-mode issuegraph install: metadata.json has dolt_mode=server,
	// and the main repo has a dolt/ data directory (created by issuegraph init --server).
	mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
	if err := os.MkdirAll(mainIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mainDoltDir := filepath.Join(mainIssueGraphDir, "dolt")
	if err := os.MkdirAll(mainDoltDir, 0o755); err != nil {
		t.Fatal(err)
	}

	mustWrite := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Server-mode metadata — the key trigger for the bug.
	mustWrite(filepath.Join(mainIssueGraphDir, "metadata.json"),
		`{"database":"dolt","backend":"dolt","dolt_mode":"server","dolt_database":"test_project","project_id":"aaaa-bbbb"}`)
	mustWrite(filepath.Join(mainIssueGraphDir, "config.yaml"), "issue-prefix: \"tp\"\n")
	// .gitignore excludes the dolt data dir (as real projects do)
	mustWrite(filepath.Join(mainIssueGraphDir, ".gitignore"), "dolt/\nembeddeddolt/\n*.db\ndolt-server.*\n")

	mustWrite(filepath.Join(mainRepoDir, "README.md"), "# Test\n")
	runGit(mainRepoDir, "add", "README.md", ".issuegraph/metadata.json", ".issuegraph/config.yaml", ".issuegraph/.gitignore")
	runGit(mainRepoDir, "commit", "-m", "initial with server-mode issuegraph")

	// Create a worktree — git checks out the tracked .issuegraph/ files
	// (metadata.json, config.yaml, .gitignore) but NOT the gitignored dolt/ dir.
	worktreeDir := filepath.Join(tmpDir, "worktree")
	runGit(mainRepoDir, "worktree", "add", worktreeDir, "HEAD")
	defer func() {
		cmd := exec.Command("git", "worktree", "remove", "--force", worktreeDir)
		cmd.Dir = mainRepoDir
		_ = cmd.Run()
	}()

	// Sanity checks: worktree has metadata but no dolt/ dir.
	worktreeIssueGraphDir := filepath.Join(worktreeDir, ".issuegraph")
	if _, err := os.Stat(filepath.Join(worktreeIssueGraphDir, "metadata.json")); err != nil {
		t.Fatalf("precondition: worktree should have inherited metadata.json, got: %v", err)
	}
	if _, err := os.Stat(filepath.Join(worktreeIssueGraphDir, "dolt")); err == nil {
		t.Fatalf("precondition: worktree's dolt/ should be gitignored, but it exists")
	}

	t.Chdir(worktreeDir)
	git.ResetCaches()

	// FindDatabasePath must resolve to the main repo's dolt dir, not the worktree's.
	result := FindDatabasePath()
	resultResolved, _ := filepath.EvalSymlinks(result)
	mainDoltResolved, _ := filepath.EvalSymlinks(mainDoltDir)
	worktreeDoltDir := filepath.Join(worktreeIssueGraphDir, "dolt")

	if resultResolved == worktreeDoltDir {
		t.Fatalf("FindDatabasePath() returned worktree's .issuegraph/dolt (%q); "+
			"expected main repo's .issuegraph/dolt (%q). "+
			"This is the duplicate-server bug: the worktree's inherited "+
			"server-mode metadata causes a separate dolt server to spawn "+
			"against an empty data directory.",
			result, mainDoltDir)
	}
	if resultResolved != mainDoltResolved {
		t.Errorf("FindDatabasePath() = %q, want main repo dolt dir %q", result, mainDoltDir)
	}

	// Also verify FindIssueGraphDir resolves to the main repo's .issuegraph.
	issuegraphDirResult := FindIssueGraphDir()
	issuegraphDirResolved, _ := filepath.EvalSymlinks(issuegraphDirResult)
	mainIssueGraphDirResolved, _ := filepath.EvalSymlinks(mainIssueGraphDir)
	if issuegraphDirResolved != mainIssueGraphDirResolved {
		t.Errorf("FindIssueGraphDir() = %q, want main repo .issuegraph %q", issuegraphDirResult, mainIssueGraphDir)
	}
}

// TestFindIssueGraphDir_JJSecondaryWithInheritedArtifacts covers the case where a
// jujutsu secondary workspace (created via `jj workspace add` without
// --colocate from a colocated jj+git primary workspace) is placed inside the
// primary's git working tree. The secondary's .issuegraph/ contains git-tracked
// metadata files but no database, while the primary has the real database.
// FindIssueGraphDir must return the primary's .issuegraph/, not the secondary's.
func TestFindIssueGraphDir_JJSecondaryWithInheritedArtifacts(t *testing.T) {
	originalEnv := os.Getenv("ISSUEGRAPH_DIR")
	defer func() {
		if originalEnv != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalEnv)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	}()
	os.Unsetenv("ISSUEGRAPH_DIR")

	tmpDir, err := os.MkdirTemp("", "issuegraph-jj-secondary-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	// Primary workspace: colocated jj+git, has .jj/repo as a directory and .git.
	primaryDir := filepath.Join(tmpDir, "primary")
	if err := os.MkdirAll(filepath.Join(primaryDir, ".jj", "repo"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Secondary workspace: inside the primary's tree (like `jj workspace add ./workspaces/secondary`).
	// Has .jj/repo as a FILE pointing to primary's .jj/repo.
	secondaryDir := filepath.Join(primaryDir, "workspaces", "secondary")
	if err := os.MkdirAll(filepath.Join(secondaryDir, ".jj"), 0o755); err != nil {
		t.Fatal(err)
	}
	primaryRepoPath := filepath.Join(primaryDir, ".jj", "repo")
	if err := os.WriteFile(filepath.Join(secondaryDir, ".jj", "repo"), []byte(primaryRepoPath+"\n"), 0o640); err != nil {
		t.Fatal(err)
	}

	// Set up git in the primary (so git rev-parse works from the secondary).
	runGit := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, gitErr := cmd.CombinedOutput()
		if gitErr != nil {
			t.Fatalf("git %v in %s: %v\n%s", args, dir, gitErr, out)
		}
	}
	if err := func() error {
		cmd := exec.Command("git", "init")
		cmd.Dir = primaryDir
		return cmd.Run()
	}(); err != nil {
		t.Skipf("git not available: %v", err)
	}
	runGit(primaryDir, "config", "user.email", "test@example.com")
	runGit(primaryDir, "config", "user.name", "Test User")

	mustWrite := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Primary .issuegraph/: real database + config.
	primaryIssueGraphDir := filepath.Join(primaryDir, ".issuegraph")
	if err := os.MkdirAll(primaryIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(filepath.Join(primaryIssueGraphDir, "metadata.json"), `{"database":"dolt"}`)
	mustWrite(filepath.Join(primaryIssueGraphDir, "config.yaml"), "issue-prefix: \"proj\"\n")
	mustWrite(filepath.Join(primaryIssueGraphDir, "issuegraph.db"), "")
	mustWrite(filepath.Join(primaryIssueGraphDir, ".gitignore"), "*.db\ndolt/\nembeddeddolt/\n")

	// Commit the metadata so the secondary inherits it via git checkout.
	mustWrite(filepath.Join(primaryDir, "README.md"), "# Test\n")
	runGit(primaryDir, "add", "README.md", ".issuegraph/metadata.json", ".issuegraph/config.yaml", ".issuegraph/.gitignore")
	runGit(primaryDir, "commit", "-m", "initial")

	// Simulate the secondary workspace inheriting tracked .issuegraph/ artifacts (no DB).
	secondaryIssueGraphDir := filepath.Join(secondaryDir, ".issuegraph")
	if err := os.MkdirAll(secondaryIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(filepath.Join(secondaryIssueGraphDir, "metadata.json"), `{"database":"dolt"}`)
	mustWrite(filepath.Join(secondaryIssueGraphDir, "config.yaml"), "issue-prefix: \"proj\"\n")
	// No database in secondary — this is the inherited-artifacts scenario.

	t.Chdir(secondaryDir)
	git.ResetCaches()

	result := FindIssueGraphDir()

	resultResolved, _ := filepath.EvalSymlinks(result)
	primaryIssueGraphDirResolved, _ := filepath.EvalSymlinks(primaryIssueGraphDir)
	secondaryIssueGraphDirResolved, _ := filepath.EvalSymlinks(secondaryIssueGraphDir)

	if resultResolved == secondaryIssueGraphDirResolved {
		t.Fatalf("FindIssueGraphDir() returned secondary workspace's inherited .issuegraph/ (%q); "+
			"expected primary workspace's .issuegraph/ (%q). "+
			"Secondary has config files but no database; issuegraph should use the primary's DB.",
			result, primaryIssueGraphDir)
	}
	if resultResolved != primaryIssueGraphDirResolved {
		t.Errorf("FindIssueGraphDir() = %q, want primary workspace .issuegraph %q", result, primaryIssueGraphDir)
	}

	fromResult := FindIssueGraphDirFrom(secondaryDir)
	fromResultResolved, _ := filepath.EvalSymlinks(fromResult)
	if fromResultResolved == secondaryIssueGraphDirResolved {
		t.Fatalf("FindIssueGraphDirFrom() returned secondary workspace's inherited .issuegraph/ (%q); "+
			"expected primary workspace's .issuegraph/ (%q).",
			fromResult, primaryIssueGraphDir)
	}
	if fromResultResolved != primaryIssueGraphDirResolved {
		t.Errorf("FindIssueGraphDirFrom() = %q, want primary workspace .issuegraph %q", fromResult, primaryIssueGraphDir)
	}
}

// TestFindIssueGraphDir_JJSecondarySeparateDBPreservesLocal verifies that when a jj
// secondary workspace genuinely has its own database (separate-DB mode),
// FindIssueGraphDir returns the secondary's .issuegraph/ — not the primary's.
func TestFindIssueGraphDir_JJSecondarySeparateDBPreservesLocal(t *testing.T) {
	originalEnv := os.Getenv("ISSUEGRAPH_DIR")
	defer func() {
		if originalEnv != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalEnv)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	}()
	os.Unsetenv("ISSUEGRAPH_DIR")

	tmpDir, err := os.MkdirTemp("", "issuegraph-jj-secondary-sep-db-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	primaryDir := filepath.Join(tmpDir, "primary")
	if err := os.MkdirAll(filepath.Join(primaryDir, ".jj", "repo"), 0o755); err != nil {
		t.Fatal(err)
	}

	secondaryDir := filepath.Join(primaryDir, "workspaces", "secondary")
	if err := os.MkdirAll(filepath.Join(secondaryDir, ".jj"), 0o755); err != nil {
		t.Fatal(err)
	}
	primaryRepoPath := filepath.Join(primaryDir, ".jj", "repo")
	if err := os.WriteFile(filepath.Join(secondaryDir, ".jj", "repo"), []byte(primaryRepoPath+"\n"), 0o640); err != nil {
		t.Fatal(err)
	}

	if err := func() error {
		cmd := exec.Command("git", "init")
		cmd.Dir = primaryDir
		return cmd.Run()
	}(); err != nil {
		t.Skipf("git not available: %v", err)
	}
	runGit := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, gitErr := cmd.CombinedOutput()
		if gitErr != nil {
			t.Fatalf("git %v in %s: %v\n%s", args, dir, gitErr, out)
		}
	}
	runGit(primaryDir, "config", "user.email", "test@example.com")
	runGit(primaryDir, "config", "user.name", "Test User")

	// Primary has a real DB.
	primaryIssueGraphDir := filepath.Join(primaryDir, ".issuegraph")
	if err := os.MkdirAll(primaryIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(primaryIssueGraphDir, "issuegraph.db"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(primaryDir, "README.md"), []byte("# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(primaryDir, "add", "README.md")
	runGit(primaryDir, "commit", "-m", "initial")

	// Secondary ALSO has a real DB — separate-DB mode.
	secondaryIssueGraphDir := filepath.Join(secondaryDir, ".issuegraph")
	if err := os.MkdirAll(secondaryIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secondaryIssueGraphDir, "issuegraph.db"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(secondaryDir)
	git.ResetCaches()

	result := FindIssueGraphDir()
	resultResolved, _ := filepath.EvalSymlinks(result)
	secondaryIssueGraphDirResolved, _ := filepath.EvalSymlinks(secondaryIssueGraphDir)

	if resultResolved != secondaryIssueGraphDirResolved {
		t.Errorf("FindIssueGraphDir() = %q, want secondary workspace .issuegraph %q (separate-DB mode)",
			result, secondaryIssueGraphDir)
	}

	fromResult := FindIssueGraphDirFrom(secondaryDir)
	fromResultResolved, _ := filepath.EvalSymlinks(fromResult)
	if fromResultResolved != secondaryIssueGraphDirResolved {
		t.Errorf("FindIssueGraphDirFrom() = %q, want secondary workspace .issuegraph %q (separate-DB mode)",
			fromResult, secondaryIssueGraphDir)
	}
}

// TestFindDatabasePath_WorktreeServerModeFromSubdir is the same scenario as
// TestFindDatabasePath_WorktreeServerModeSharesMainRepo but with the CWD set
// to a subdirectory of the worktree rather than the worktree root. This
// exercises the walk-up and worktree-specific code paths (as opposed to the
// CWD check which only fires at the worktree root).
func TestFindDatabasePath_WorktreeServerModeFromSubdir(t *testing.T) {
	originalEnvDir := os.Getenv("ISSUEGRAPH_DIR")
	originalEnvDB := os.Getenv("ISSUEGRAPH_DB")
	defer func() {
		if originalEnvDir != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalEnvDir)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
		if originalEnvDB != "" {
			os.Setenv("ISSUEGRAPH_DB", originalEnvDB)
		} else {
			os.Unsetenv("ISSUEGRAPH_DB")
		}
	}()
	os.Unsetenv("ISSUEGRAPH_DIR")
	os.Unsetenv("ISSUEGRAPH_DB")

	tmpDir, err := os.MkdirTemp("", "issuegraph-worktree-server-subdir-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mainRepoDir := filepath.Join(tmpDir, "main-repo")
	if err := os.MkdirAll(mainRepoDir, 0o755); err != nil {
		t.Fatal(err)
	}

	runGit := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
		}
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}
	runGit(mainRepoDir, "config", "user.email", "test@example.com")
	runGit(mainRepoDir, "config", "user.name", "Test User")

	mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
	if err := os.MkdirAll(mainIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mainDoltDir := filepath.Join(mainIssueGraphDir, "dolt")
	if err := os.MkdirAll(mainDoltDir, 0o755); err != nil {
		t.Fatal(err)
	}

	mustWrite := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(filepath.Join(mainIssueGraphDir, "metadata.json"),
		`{"database":"dolt","backend":"dolt","dolt_mode":"server","dolt_database":"test_project"}`)
	mustWrite(filepath.Join(mainIssueGraphDir, "config.yaml"), "issue-prefix: \"tp\"\n")
	mustWrite(filepath.Join(mainIssueGraphDir, ".gitignore"), "dolt/\nembeddeddolt/\n*.db\ndolt-server.*\n")

	mustWrite(filepath.Join(mainRepoDir, "README.md"), "# Test\n")
	runGit(mainRepoDir, "add", "README.md", ".issuegraph/metadata.json", ".issuegraph/config.yaml", ".issuegraph/.gitignore")
	runGit(mainRepoDir, "commit", "-m", "initial")

	worktreeDir := filepath.Join(tmpDir, "worktree")
	runGit(mainRepoDir, "worktree", "add", worktreeDir, "HEAD")
	defer func() {
		cmd := exec.Command("git", "worktree", "remove", "--force", worktreeDir)
		cmd.Dir = mainRepoDir
		_ = cmd.Run()
	}()

	// CWD is a subdirectory of the worktree — tests the worktree-specific
	// code path rather than the CWD check.
	subDir := filepath.Join(worktreeDir, "src", "pkg")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(subDir)
	git.ResetCaches()

	result := FindDatabasePath()
	resultResolved, _ := filepath.EvalSymlinks(result)
	mainDoltResolved, _ := filepath.EvalSymlinks(mainDoltDir)

	if resultResolved != mainDoltResolved {
		t.Errorf("FindDatabasePath() from subdir = %q, want main repo dolt dir %q", result, mainDoltDir)
	}
}

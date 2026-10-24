package issuegraph

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/git"
	"github.com/borg0ai/dx-issuegraph/internal/utils"
)

func TestFindDatabasePathEnvVar(t *testing.T) {
	// Save original env vars
	originalDB := os.Getenv("ISSUEGRAPH_DB")
	originalDir := os.Getenv("ISSUEGRAPH_DIR")
	defer func() {
		if originalDB != "" {
			_ = os.Setenv("ISSUEGRAPH_DB", originalDB)
		} else {
			_ = os.Unsetenv("ISSUEGRAPH_DB")
		}
		if originalDir != "" {
			_ = os.Setenv("ISSUEGRAPH_DIR", originalDir)
		} else {
			_ = os.Unsetenv("ISSUEGRAPH_DIR")
		}
	}()

	// Clear ISSUEGRAPH_DIR to prevent it from interfering
	_ = os.Unsetenv("ISSUEGRAPH_DIR")

	// Set env var to a test path (platform-agnostic)
	testPath := filepath.Join("test", "path", "test.db")
	_ = os.Setenv("ISSUEGRAPH_DB", testPath)

	result := FindDatabasePath()
	// FindDatabasePath canonicalizes to absolute path
	expectedPath, _ := filepath.Abs(testPath)
	if result != expectedPath {
		t.Errorf("Expected '%s', got '%s'", expectedPath, result)
	}
}

func TestFindDatabasePathInTree(t *testing.T) {
	// Save original env vars
	originalDB := os.Getenv("ISSUEGRAPH_DB")
	originalDir := os.Getenv("ISSUEGRAPH_DIR")
	defer func() {
		if originalDB != "" {
			os.Setenv("ISSUEGRAPH_DB", originalDB)
		} else {
			os.Unsetenv("ISSUEGRAPH_DB")
		}
		if originalDir != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalDir)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	}()

	// Clear env vars
	os.Unsetenv("ISSUEGRAPH_DB")

	// Create temporary directory structure
	tmpDir, err := os.MkdirTemp("", "issuegraph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .issuegraph directory with a dolt database directory
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	err = os.MkdirAll(issuegraphDir, 0o750)
	if err != nil {
		t.Fatalf("Failed to create .issuegraph dir: %v", err)
	}

	doltDir := filepath.Join(issuegraphDir, "dolt")
	if err := os.MkdirAll(doltDir, 0o750); err != nil {
		t.Fatalf("Failed to create dolt dir: %v", err)
	}

	// Set ISSUEGRAPH_DIR to our test .issuegraph directory to override git repo detection
	os.Setenv("ISSUEGRAPH_DIR", issuegraphDir)

	// Create a subdirectory and change to it
	subDir := filepath.Join(tmpDir, "sub", "nested")
	err = os.MkdirAll(subDir, 0o750)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	t.Chdir(subDir)

	// Should find the database in the parent directory tree
	result := FindDatabasePath()

	// Resolve symlinks for both paths (macOS uses /private/var symlinked to /var)
	expectedPath, err := filepath.EvalSymlinks(doltDir)
	if err != nil {
		expectedPath = doltDir
	}
	resultPath, err := filepath.EvalSymlinks(result)
	if err != nil {
		resultPath = result
	}

	if resultPath != expectedPath {
		t.Errorf("Expected '%s', got '%s'", expectedPath, resultPath)
	}
}

func TestFindDatabasePathNotFound(t *testing.T) {
	// Save original env var
	originalEnv := os.Getenv("ISSUEGRAPH_DB")
	defer func() {
		if originalEnv != "" {
			os.Setenv("ISSUEGRAPH_DB", originalEnv)
		} else {
			os.Unsetenv("ISSUEGRAPH_DB")
		}
	}()

	// Clear env var
	os.Unsetenv("ISSUEGRAPH_DB")

	// Create temporary directory without .issuegraph
	tmpDir, err := os.MkdirTemp("", "issuegraph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Chdir(tmpDir)

	// Should return empty string (no database found)
	result := FindDatabasePath()
	// Result might be the home directory default if it exists, or empty string
	// Just verify it doesn't error
	_ = result
}

// TestFindDatabasePath_ISSUEGRAPH_DB_Directory tests that FindDatabasePath behaves
// correctly when ISSUEGRAPH_DB points to a directory (like .issuegraph/) rather than a
// .db file. This is a regression test for a bug where main.go:476 does
// issuegraphDir := filepath.Dir(dbPath), which resolves one level too high when
// dbPath is a directory.
//
// With ISSUEGRAPH_DIR, FindDatabasePath returns .issuegraph/dolt (a path inside .issuegraph/),
// so filepath.Dir() correctly yields .issuegraph/. But with ISSUEGRAPH_DB pointing to
// .issuegraph/, it returns .issuegraph/ itself, so filepath.Dir() yields the parent —
// causing stray dolt/ directories and broken server connections.
func TestFindDatabasePath_ISSUEGRAPH_DB_Directory(t *testing.T) {
	// Save original env vars
	originalDB := os.Getenv("ISSUEGRAPH_DB")
	originalDir := os.Getenv("ISSUEGRAPH_DIR")
	defer func() {
		if originalDB != "" {
			os.Setenv("ISSUEGRAPH_DB", originalDB)
		} else {
			os.Unsetenv("ISSUEGRAPH_DB")
		}
		if originalDir != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalDir)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	}()

	// Create a .issuegraph/ directory with dolt/ inside, mimicking a real project
	tmpDir, err := os.MkdirTemp("", "issuegraph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	doltDir := filepath.Join(issuegraphDir, "dolt")
	if err := os.MkdirAll(doltDir, 0o750); err != nil {
		t.Fatalf("Failed to create dolt dir: %v", err)
	}

	// Resolve symlinks for comparison (macOS /var → /private/var)
	issuegraphDirResolved, err := filepath.EvalSymlinks(issuegraphDir)
	if err != nil {
		issuegraphDirResolved = issuegraphDir
	}

	// Test with ISSUEGRAPH_DIR — this works correctly
	os.Unsetenv("ISSUEGRAPH_DB")
	os.Setenv("ISSUEGRAPH_DIR", issuegraphDir)
	resultDir := FindDatabasePath()
	resultDirResolved, _ := filepath.EvalSymlinks(resultDir)
	derivedFromDir := filepath.Dir(resultDirResolved)
	if derivedFromDir != issuegraphDirResolved {
		t.Errorf("ISSUEGRAPH_DIR: filepath.Dir(FindDatabasePath()) = %q, want %q",
			derivedFromDir, issuegraphDirResolved)
	}

	// Test with ISSUEGRAPH_DB pointing to the same directory — this is the bug.
	// FindDatabasePath returns .issuegraph/ itself (not .issuegraph/dolt), so
	// filepath.Dir() yields the parent directory instead of .issuegraph/.
	os.Unsetenv("ISSUEGRAPH_DIR")
	os.Setenv("ISSUEGRAPH_DB", issuegraphDir)
	resultDB := FindDatabasePath()
	resultDBResolved, _ := filepath.EvalSymlinks(resultDB)
	derivedFromDB := filepath.Dir(resultDBResolved)
	if derivedFromDB != issuegraphDirResolved {
		t.Errorf("ISSUEGRAPH_DB (directory): filepath.Dir(FindDatabasePath()) = %q, want %q\n"+
			"FindDatabasePath returned %q — should return a path inside .issuegraph/, not .issuegraph/ itself",
			derivedFromDB, issuegraphDirResolved, resultDBResolved)
	}
}

// TestFindDatabasePath_ISSUEGRAPH_DB_DirectoryTrailingSlash verifies that a
// trailing slash on ISSUEGRAPH_DB doesn't change the outcome — path canonicalization
// strips it, so the same filepath.Dir() bug applies.
func TestFindDatabasePath_ISSUEGRAPH_DB_DirectoryTrailingSlash(t *testing.T) {
	originalDB := os.Getenv("ISSUEGRAPH_DB")
	originalDir := os.Getenv("ISSUEGRAPH_DIR")
	defer func() {
		if originalDB != "" {
			os.Setenv("ISSUEGRAPH_DB", originalDB)
		} else {
			os.Unsetenv("ISSUEGRAPH_DB")
		}
		if originalDir != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalDir)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	}()

	tmpDir, err := os.MkdirTemp("", "issuegraph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	doltDir := filepath.Join(issuegraphDir, "dolt")
	if err := os.MkdirAll(doltDir, 0o750); err != nil {
		t.Fatalf("Failed to create dolt dir: %v", err)
	}

	issuegraphDirResolved, err := filepath.EvalSymlinks(issuegraphDir)
	if err != nil {
		issuegraphDirResolved = issuegraphDir
	}

	// Set ISSUEGRAPH_DB with trailing slash
	os.Unsetenv("ISSUEGRAPH_DIR")
	os.Setenv("ISSUEGRAPH_DB", issuegraphDir+string(filepath.Separator))

	result := FindDatabasePath()
	resultResolved, _ := filepath.EvalSymlinks(result)
	derived := filepath.Dir(resultResolved)
	if derived != issuegraphDirResolved {
		t.Errorf("ISSUEGRAPH_DB (trailing slash): filepath.Dir(FindDatabasePath()) = %q, want %q\n"+
			"FindDatabasePath returned %q",
			derived, issuegraphDirResolved, resultResolved)
	}
}

// TestHasIssueGraphProjectFiles verifies that hasIssueGraphProjectFiles correctly
// distinguishes between project directories and daemon-only directories (bd-420)
func TestHasIssueGraphProjectFiles(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected bool
	}{
		{
			name:     "empty directory",
			files:    []string{},
			expected: false,
		},
		{
			name:     "daemon registry only",
			files:    []string{"registry.json", "registry.lock"},
			expected: false,
		},
		{
			name:     "has database",
			files:    []string{"issuegraph.db"},
			expected: true,
		},
		{
			name:     "has metadata.json",
			files:    []string{"metadata.json"},
			expected: true,
		},
		{
			name:     "has config.yaml",
			files:    []string{"config.yaml"},
			expected: true,
		},
		{
			name:     "ignores backup db",
			files:    []string{"issuegraph.backup.db"},
			expected: false,
		},
		{
			name:     "ignores vc.db",
			files:    []string{"vc.db"},
			expected: false,
		},
		{
			name:     "real db with backup",
			files:    []string{"issuegraph.db", "issuegraph.backup.db"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "issuegraph-project-test-*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Create test files
			for _, file := range tt.files {
				path := filepath.Join(tmpDir, file)
				if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
					t.Fatal(err)
				}
			}

			result := hasIssueGraphProjectFiles(tmpDir)
			if result != tt.expected {
				t.Errorf("hasIssueGraphProjectFiles() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestFindIssueGraphDirSkipsDaemonRegistry verifies that FindIssueGraphDir skips
// directories containing only daemon registry files (bd-420)
func TestFindIssueGraphDirSkipsDaemonRegistry(t *testing.T) {
	// Save original state
	originalEnv := os.Getenv("ISSUEGRAPH_DIR")
	defer func() {
		if originalEnv != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalEnv)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	}()
	os.Unsetenv("ISSUEGRAPH_DIR")

	// Create temp directory structure
	tmpDir, err := os.MkdirTemp("", "issuegraph-daemon-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .issuegraph with only daemon registry files (should be skipped)
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(issuegraphDir, "registry.json"), []byte("[]"), 0644); err != nil {
		t.Fatal(err)
	}

	// Change to temp dir
	t.Chdir(tmpDir)

	// Should NOT find the daemon-only directory
	result := FindIssueGraphDir()
	if result != "" {
		// Resolve symlinks for comparison
		resultResolved, _ := filepath.EvalSymlinks(result)
		issuegraphDirResolved, _ := filepath.EvalSymlinks(issuegraphDir)
		if resultResolved == issuegraphDirResolved {
			t.Errorf("FindIssueGraphDir() should skip daemon-only directory, got %q", result)
		}
	}
}

// TestFindIssueGraphDirValidatesIssueGraphDirEnv verifies that ISSUEGRAPH_DIR env var
// is validated for project files (bd-420)
func TestFindIssueGraphDirValidatesIssueGraphDirEnv(t *testing.T) {
	// Save original state
	originalEnv := os.Getenv("ISSUEGRAPH_DIR")
	defer func() {
		if originalEnv != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalEnv)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	}()

	// Create temp directory with only daemon registry files
	tmpDir, err := os.MkdirTemp("", "issuegraph-env-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "registry.json"), []byte("[]"), 0644); err != nil {
		t.Fatal(err)
	}

	// Set ISSUEGRAPH_DIR to daemon-only directory
	os.Setenv("ISSUEGRAPH_DIR", tmpDir)

	// Should NOT return the daemon-only directory
	result := FindIssueGraphDir()
	if result != "" {
		resultResolved, _ := filepath.EvalSymlinks(result)
		tmpDirResolved, _ := filepath.EvalSymlinks(tmpDir)
		if resultResolved == tmpDirResolved {
			t.Errorf("FindIssueGraphDir() should skip ISSUEGRAPH_DIR with only daemon files, got %q", result)
		}
	}

	// Now add a project file
	if err := os.WriteFile(filepath.Join(tmpDir, "issuegraph.db"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	// Should now return the directory
	result = FindIssueGraphDir()
	if result == "" {
		t.Error("FindIssueGraphDir() should return ISSUEGRAPH_DIR with project files")
	}
}

func TestFindIssueGraphDirPrefersBranchWorktreeForDetachedCommitISSUEGRAPH_DIR(t *testing.T) {
	originalEnv := os.Getenv("ISSUEGRAPH_DIR")
	defer func() {
		if originalEnv != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalEnv)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	}()

	detachedIssueGraphDir, mainIssueGraphDir, _ := setupDetachedCommitIssueGraphWorktree(t)
	os.Setenv("ISSUEGRAPH_DIR", detachedIssueGraphDir)

	result := FindIssueGraphDir()

	resultResolved, _ := filepath.EvalSymlinks(result)
	mainResolved, _ := filepath.EvalSymlinks(mainIssueGraphDir)

	if resultResolved != mainResolved {
		t.Errorf("FindIssueGraphDir() = %q, want stable branch worktree %q", result, mainIssueGraphDir)
	}
}

func TestFindDatabasePathPrefersBranchWorktreeForDetachedCommitISSUEGRAPH_DIR(t *testing.T) {
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
	os.Unsetenv("ISSUEGRAPH_DB")

	detachedIssueGraphDir, _, mainDoltDir := setupDetachedCommitIssueGraphWorktree(t)
	os.Setenv("ISSUEGRAPH_DIR", detachedIssueGraphDir)

	result := FindDatabasePath()

	resultResolved, _ := filepath.EvalSymlinks(result)
	mainResolved, _ := filepath.EvalSymlinks(mainDoltDir)

	if resultResolved != mainResolved {
		t.Errorf("FindDatabasePath() = %q, want stable branch worktree db %q", result, mainDoltDir)
	}
}

func TestFindDatabasePathHomeDefault(t *testing.T) {
	// This test verifies that if no database is found, it falls back to home directory
	// We can't reliably test this without modifying the home directory, so we'll skip
	// creating the file and just verify the function doesn't crash

	originalEnv := os.Getenv("ISSUEGRAPH_DB")
	defer func() {
		if originalEnv != "" {
			os.Setenv("ISSUEGRAPH_DB", originalEnv)
		} else {
			os.Unsetenv("ISSUEGRAPH_DB")
		}
	}()

	os.Unsetenv("ISSUEGRAPH_DB")

	// Create an empty temp directory and cd to it
	tmpDir, err := os.MkdirTemp("", "issuegraph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Chdir(tmpDir)

	// Call FindDatabasePath - it might return home dir default or empty string
	result := FindDatabasePath()

	// If result is not empty, verify it contains .issuegraph
	if result != "" && !filepath.IsAbs(result) {
		t.Errorf("Expected absolute path or empty string, got '%s'", result)
	}
}

func setupDetachedCommitIssueGraphWorktree(t *testing.T) (string, string, string) {
	t.Helper()

	tmpDir := t.TempDir()
	storeDir := filepath.Join(tmpDir, "store")
	bareDir := filepath.Join(storeDir, ".bare")
	mainWorktreeDir := filepath.Join(storeDir, "refs", "heads", "main")

	if err := os.MkdirAll(filepath.Dir(mainWorktreeDir), 0755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("git", "init", "--bare", bareDir)
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}
	// Ensure HEAD points to main regardless of init.defaultBranch setting.
	// Without this, worktree add -b main fails on systems where the default
	// branch is not "main". Fix inspired by PR #2565 (cwalv).
	runGitInDir(t, tmpDir, "--git-dir", bareDir, "symbolic-ref", "HEAD", "refs/heads/main")

	// A bare repo starts with no commits, so HEAD is invalid. Create an
	// initial empty commit so "git worktree add -b main" can succeed.
	// We use plumbing commands since "git commit" requires a worktree.
	runGitInDir(t, tmpDir, "--git-dir", bareDir, "config", "user.email", "test@example.com")
	runGitInDir(t, tmpDir, "--git-dir", bareDir, "config", "user.name", "Test User")
	emptyTree := runGitInDir(t, tmpDir, "--git-dir", bareDir, "hash-object", "-t", "tree", "/dev/null")
	initCommit := runGitInDir(t, tmpDir, "--git-dir", bareDir, "commit-tree", "-m", "Initial commit", emptyTree)
	runGitInDir(t, tmpDir, "--git-dir", bareDir, "update-ref", "HEAD", initCommit)

	runGitInDir(t, tmpDir, "--git-dir", bareDir, "worktree", "add", mainWorktreeDir, "main")

	mainIssueGraphDir := filepath.Join(mainWorktreeDir, ".issuegraph")
	mainDoltDir := filepath.Join(mainIssueGraphDir, "dolt")
	if err := os.MkdirAll(mainDoltDir, 0755); err != nil {
		t.Fatal(err)
	}

	head := runGitInDir(t, mainWorktreeDir, "rev-parse", "HEAD")
	detachedWorktreeDir := filepath.Join(storeDir, "refs", "commits", head)
	if err := os.MkdirAll(filepath.Dir(detachedWorktreeDir), 0755); err != nil {
		t.Fatal(err)
	}

	runGitInDir(t, tmpDir, "--git-dir", bareDir, "worktree", "add", "--detach", detachedWorktreeDir, head)

	return filepath.Join(detachedWorktreeDir, ".issuegraph"), mainIssueGraphDir, mainDoltDir
}

func setupRegularWorktreeRepo(t *testing.T) (string, string) {
	t.Helper()

	tmpDir := t.TempDir()
	mainRepoDir := filepath.Join(tmpDir, "main-repo")
	worktreeDir := filepath.Join(tmpDir, "worktree")

	if err := os.MkdirAll(mainRepoDir, 0o755); err != nil {
		t.Fatal(err)
	}

	runGitInDir(t, mainRepoDir, "init")
	runGitInDir(t, mainRepoDir, "config", "user.email", "test@example.com")
	runGitInDir(t, mainRepoDir, "config", "user.name", "Test User")

	if err := os.WriteFile(filepath.Join(mainRepoDir, "README.md"), []byte("# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitInDir(t, mainRepoDir, "add", "README.md")
	runGitInDir(t, mainRepoDir, "commit", "-m", "Initial commit")
	runGitInDir(t, mainRepoDir, "worktree", "add", worktreeDir, "HEAD")

	return mainRepoDir, worktreeDir
}

func runGitInDir(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed in %s: %v\n%s", args, dir, err, output)
	}

	return strings.TrimSpace(string(output))
}

func setupBareParentWorktree(t *testing.T) (string, string) {
	t.Helper()

	tmpDir := t.TempDir()
	bareDir := filepath.Join(tmpDir, "repo.git")
	mainWorktreeDir := filepath.Join(tmpDir, "main")
	featureWorktreeDir := filepath.Join(tmpDir, "feature")

	runGitInDir(t, tmpDir, "init", "--bare", bareDir)
	runGitInDir(t, tmpDir, "--git-dir", bareDir, "symbolic-ref", "HEAD", "refs/heads/main")
	runGitInDir(t, tmpDir, "--git-dir", bareDir, "config", "user.email", "test@example.com")
	runGitInDir(t, tmpDir, "--git-dir", bareDir, "config", "user.name", "Test User")
	emptyTree := runGitInDir(t, tmpDir, "--git-dir", bareDir, "hash-object", "-t", "tree", "/dev/null")
	initCommit := runGitInDir(t, tmpDir, "--git-dir", bareDir, "commit-tree", "-m", "Initial commit", emptyTree)
	runGitInDir(t, tmpDir, "--git-dir", bareDir, "update-ref", "HEAD", initCommit)
	runGitInDir(t, tmpDir, "--git-dir", bareDir, "worktree", "add", mainWorktreeDir, "main")
	runGitInDir(t, mainWorktreeDir, "branch", "feature")
	runGitInDir(t, tmpDir, "--git-dir", bareDir, "worktree", "add", featureWorktreeDir, "feature")

	return bareDir, featureWorktreeDir
}

// TestFollowRedirect tests the redirect file functionality
func TestFollowRedirect(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(t *testing.T, tmpDir string) (stubDir, targetDir string)
		expectRedirect bool
	}{
		{
			name: "no redirect file - returns original",
			setupFunc: func(t *testing.T, tmpDir string) (string, string) {
				issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
				if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
					t.Fatal(err)
				}
				return issuegraphDir, ""
			},
			expectRedirect: false,
		},
		{
			name: "relative path redirect",
			setupFunc: func(t *testing.T, tmpDir string) (string, string) {
				// Create stub .issuegraph with redirect
				stubDir := filepath.Join(tmpDir, "project", ".issuegraph")
				if err := os.MkdirAll(stubDir, 0755); err != nil {
					t.Fatal(err)
				}

				// Create target .issuegraph directory
				targetDir := filepath.Join(tmpDir, "actual", ".issuegraph")
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(targetDir, "issuegraph.db"), []byte{}, 0644); err != nil {
					t.Fatal(err)
				}

				// Write redirect file with relative path
				redirectPath := filepath.Join(stubDir, "redirect")
				if err := os.WriteFile(redirectPath, []byte("../actual/.issuegraph\n"), 0644); err != nil {
					t.Fatal(err)
				}

				return stubDir, targetDir
			},
			expectRedirect: true,
		},
		{
			name: "absolute path redirect",
			setupFunc: func(t *testing.T, tmpDir string) (string, string) {
				// Create stub .issuegraph with redirect
				stubDir := filepath.Join(tmpDir, "project", ".issuegraph")
				if err := os.MkdirAll(stubDir, 0755); err != nil {
					t.Fatal(err)
				}

				// Create target .issuegraph directory
				targetDir := filepath.Join(tmpDir, "actual", ".issuegraph")
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(targetDir, "issuegraph.db"), []byte{}, 0644); err != nil {
					t.Fatal(err)
				}

				// Write redirect file with absolute path
				redirectPath := filepath.Join(stubDir, "redirect")
				if err := os.WriteFile(redirectPath, []byte(targetDir+"\n"), 0644); err != nil {
					t.Fatal(err)
				}

				return stubDir, targetDir
			},
			expectRedirect: true,
		},
		{
			name: "redirect with comments",
			setupFunc: func(t *testing.T, tmpDir string) (string, string) {
				// Create stub .issuegraph with redirect
				stubDir := filepath.Join(tmpDir, "project", ".issuegraph")
				if err := os.MkdirAll(stubDir, 0755); err != nil {
					t.Fatal(err)
				}

				// Create target .issuegraph directory
				targetDir := filepath.Join(tmpDir, "actual", ".issuegraph")
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(targetDir, "issuegraph.db"), []byte{}, 0644); err != nil {
					t.Fatal(err)
				}

				// Write redirect file with comments
				redirectPath := filepath.Join(stubDir, "redirect")
				content := "# Redirect to actual issuegraph location\n# This is a workspace redirect\n" + targetDir + "\n"
				if err := os.WriteFile(redirectPath, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}

				return stubDir, targetDir
			},
			expectRedirect: true,
		},
		{
			name: "redirect to non-existent directory - returns original",
			setupFunc: func(t *testing.T, tmpDir string) (string, string) {
				stubDir := filepath.Join(tmpDir, "project", ".issuegraph")
				if err := os.MkdirAll(stubDir, 0755); err != nil {
					t.Fatal(err)
				}

				// Write redirect to non-existent path
				redirectPath := filepath.Join(stubDir, "redirect")
				if err := os.WriteFile(redirectPath, []byte("/nonexistent/path/.issuegraph\n"), 0644); err != nil {
					t.Fatal(err)
				}

				return stubDir, ""
			},
			expectRedirect: false, // Should fall back to original
		},
		{
			name: "empty redirect file - returns original",
			setupFunc: func(t *testing.T, tmpDir string) (string, string) {
				stubDir := filepath.Join(tmpDir, "project", ".issuegraph")
				if err := os.MkdirAll(stubDir, 0755); err != nil {
					t.Fatal(err)
				}

				// Write empty redirect file
				redirectPath := filepath.Join(stubDir, "redirect")
				if err := os.WriteFile(redirectPath, []byte(""), 0644); err != nil {
					t.Fatal(err)
				}

				return stubDir, ""
			},
			expectRedirect: false,
		},
		{
			name: "redirect file with only comments - returns original",
			setupFunc: func(t *testing.T, tmpDir string) (string, string) {
				stubDir := filepath.Join(tmpDir, "project", ".issuegraph")
				if err := os.MkdirAll(stubDir, 0755); err != nil {
					t.Fatal(err)
				}

				// Write redirect file with only comments
				redirectPath := filepath.Join(stubDir, "redirect")
				if err := os.WriteFile(redirectPath, []byte("# Just a comment\n# Another comment\n"), 0644); err != nil {
					t.Fatal(err)
				}

				return stubDir, ""
			},
			expectRedirect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "issuegraph-redirect-test-*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			stubDir, targetDir := tt.setupFunc(t, tmpDir)

			result := FollowRedirect(stubDir)

			// Resolve symlinks for comparison (macOS uses /private/var)
			resultResolved, _ := filepath.EvalSymlinks(result)
			stubResolved, _ := filepath.EvalSymlinks(stubDir)

			if tt.expectRedirect {
				targetResolved, _ := filepath.EvalSymlinks(targetDir)
				if resultResolved != targetResolved {
					t.Errorf("FollowRedirect() = %q, want %q", result, targetDir)
				}
			} else {
				if resultResolved != stubResolved {
					t.Errorf("FollowRedirect() = %q, want original %q", result, stubDir)
				}
			}
		})
	}
}

// captureStderr redirects os.Stderr for the duration of fn and returns
// whatever was written to it.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("closing stderr pipe writer: %v", err)
	}
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

// TestFollowRedirect_IgnoresInvalidTarget is a regression test for
// borg0ai/dx-issuegraph#4692: a redirect whose target directory exists but has
// no metadata.json and no recognizable database is ignored (the source dir
// is returned unchanged) and a warning naming both paths is emitted once on
// stderr. docs/reference/advanced.md ("Database Redirects") documents this
// contract: "The target directory must exist and contain a valid database."
func TestFollowRedirect_IgnoresInvalidTarget(t *testing.T) {
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_MODE", "")
	t.Setenv("ISSUEGRAPH_DOLT_SHARED_SERVER", "")

	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	sourceDir := filepath.Join(tmpDir, "worktree", ".issuegraph")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}

	targetDir := filepath.Join(tmpDir, "empty-target", ".issuegraph")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	// targetDir exists but has no metadata.json and no database.

	if err := os.WriteFile(filepath.Join(sourceDir, "redirect"), []byte(targetDir+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var result string
	stderr := captureStderr(t, func() {
		result = FollowRedirect(sourceDir)
	})

	resultResolved, _ := filepath.EvalSymlinks(result)
	sourceResolved, _ := filepath.EvalSymlinks(sourceDir)
	if resultResolved != sourceResolved {
		t.Errorf("FollowRedirect() = %q, want source dir %q (invalid target should be ignored)", result, sourceDir)
	}
	if !strings.Contains(stderr, "no database or metadata.json") {
		t.Errorf("expected an invalid-target warning on stderr, got: %q", stderr)
	}
	if !strings.Contains(stderr, sourceDir) || !strings.Contains(stderr, targetDir) {
		t.Errorf("expected warning to name both source and target paths, got: %q", stderr)
	}
}

// TestFollowRedirect_FollowsWhenTargetHasDatabase documents that a redirect
// to a target with a recognizable database (no metadata.json) is followed as
// before -- the #4692 guard is about missing databases, not about source
// mode.
func TestFollowRedirect_FollowsWhenTargetHasDatabase(t *testing.T) {
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_MODE", "")
	t.Setenv("ISSUEGRAPH_DOLT_SHARED_SERVER", "")

	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	sourceDir := filepath.Join(tmpDir, "worktree", ".issuegraph")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	// No metadata.json in source (the common worktree-redirect-to-rig-root
	// pattern).

	targetDir := filepath.Join(tmpDir, "rig", ".issuegraph")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "issuegraph.db"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(sourceDir, "redirect"), []byte(targetDir+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result := FollowRedirect(sourceDir)

	resultResolved, _ := filepath.EvalSymlinks(result)
	targetResolved, _ := filepath.EvalSymlinks(targetDir)
	if resultResolved != targetResolved {
		t.Errorf("FollowRedirect() = %q, want target dir %q", result, targetDir)
	}
}

// TestFollowRedirect_ServerModeSourceStillFollowsValidTarget is the
// regression test for the cross-vendor review finding on this fix: a
// server-mode source rig redirecting to a differently-configured, valid
// shared Gas Town root (the fb51196f7 / docs "Database Redirects" topology)
// must still be FOLLOWED. "Source is server-mode" must never, by itself,
// cause a redirect to be ignored -- only target validity does.
func TestFollowRedirect_ServerModeSourceStillFollowsValidTarget(t *testing.T) {
	t.Setenv("ISSUEGRAPH_DOLT_SERVER_MODE", "")
	t.Setenv("ISSUEGRAPH_DOLT_SHARED_SERVER", "")

	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	sourceDir := filepath.Join(tmpDir, "lola", ".issuegraph")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeMetadataJSON(t, sourceDir, &configfile.Config{
		DoltMode:     "server",
		DoltDatabase: "lola",
	})

	targetDir := filepath.Join(tmpDir, "town", ".issuegraph")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeMetadataJSON(t, targetDir, &configfile.Config{
		DoltMode:     "server",
		DoltDatabase: "hq",
	})

	if err := os.WriteFile(filepath.Join(sourceDir, "redirect"), []byte(targetDir+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result := FollowRedirect(sourceDir)

	resultResolved, _ := filepath.EvalSymlinks(result)
	targetResolved, _ := filepath.EvalSymlinks(targetDir)
	if resultResolved != targetResolved {
		t.Errorf("FollowRedirect() = %q, want target dir %q (server-mode source must still follow a valid redirect)", result, targetDir)
	}
}

// TestFindDatabasePathWithRedirect tests that FindDatabasePath follows redirects
func TestFindDatabasePathWithRedirect(t *testing.T) {
	// Save original state
	originalEnv := os.Getenv("ISSUEGRAPH_DIR")
	defer func() {
		if originalEnv != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalEnv)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	}()
	os.Unsetenv("ISSUEGRAPH_DIR")

	// Create temp directory structure
	tmpDir, err := os.MkdirTemp("", "issuegraph-redirect-finddb-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create stub .issuegraph with redirect
	stubDir := filepath.Join(tmpDir, "project", ".issuegraph")
	if err := os.MkdirAll(stubDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create target .issuegraph directory with actual dolt database
	targetDir := filepath.Join(tmpDir, "actual", ".issuegraph")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	targetDolt := filepath.Join(targetDir, "dolt")
	if err := os.MkdirAll(targetDolt, 0755); err != nil {
		t.Fatal(err)
	}

	// Write redirect file
	redirectPath := filepath.Join(stubDir, "redirect")
	if err := os.WriteFile(redirectPath, []byte("../actual/.issuegraph\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Change to project directory
	projectDir := filepath.Join(tmpDir, "project")
	t.Chdir(projectDir)

	// FindDatabasePath should follow the redirect
	result := FindDatabasePath()

	// Resolve symlinks for comparison
	resultResolved, _ := filepath.EvalSymlinks(result)
	targetDoltResolved, _ := filepath.EvalSymlinks(targetDolt)

	if resultResolved != targetDoltResolved {
		t.Errorf("FindDatabasePath() = %q, want %q (via redirect)", result, targetDolt)
	}
}

// TestFindIssueGraphDirWithRedirect tests that FindIssueGraphDir follows redirects
func TestFindIssueGraphDirWithRedirect(t *testing.T) {
	// Save original state
	originalEnv := os.Getenv("ISSUEGRAPH_DIR")
	defer func() {
		if originalEnv != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalEnv)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	}()
	os.Unsetenv("ISSUEGRAPH_DIR")

	// Create temp directory structure
	tmpDir, err := os.MkdirTemp("", "issuegraph-redirect-finddir-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create stub .issuegraph with redirect
	stubDir := filepath.Join(tmpDir, "project", ".issuegraph")
	if err := os.MkdirAll(stubDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create target .issuegraph directory with project files
	targetDir := filepath.Join(tmpDir, "actual", ".issuegraph")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "metadata.json"), []byte(`{"database":"dolt"}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Write redirect file
	redirectPath := filepath.Join(stubDir, "redirect")
	if err := os.WriteFile(redirectPath, []byte("../actual/.issuegraph\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Change to project directory
	projectDir := filepath.Join(tmpDir, "project")
	t.Chdir(projectDir)

	// FindIssueGraphDir should follow the redirect
	result := FindIssueGraphDir()

	// Resolve symlinks for comparison
	resultResolved, _ := filepath.EvalSymlinks(result)
	targetDirResolved, _ := filepath.EvalSymlinks(targetDir)

	if resultResolved != targetDirResolved {
		t.Errorf("FindIssueGraphDir() = %q, want %q (via redirect)", result, targetDir)
	}
}

// TestFindGitRoot_RegularRepo tests that findGitRoot returns the correct path
// in a regular git repository (not a worktree).
func TestFindGitRoot_RegularRepo(t *testing.T) {
	// Create temporary directory for our test repo
	tmpDir, err := os.MkdirTemp("", "issuegraph-gitroot-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize a git repository
	repoDir := filepath.Join(tmpDir, "main-repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Configure git user for the test repo (required for commits)
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = repoDir
	_ = cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoDir
	_ = cmd.Run()

	// Create a subdirectory and change to it
	subDir := filepath.Join(repoDir, "sub", "nested")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	t.Chdir(subDir)
	git.ResetCaches() // Reset after chdir for caching tests

	// findGitRoot should return the repo root
	result := findGitRoot()

	// Resolve symlinks for comparison (macOS /var -> /private/var)
	resultResolved, _ := filepath.EvalSymlinks(result)
	repoDirResolved, _ := filepath.EvalSymlinks(repoDir)

	if resultResolved != repoDirResolved {
		t.Errorf("findGitRoot() = %q, want %q", result, repoDir)
	}
}

// TestFindGitRoot_Worktree tests that findGitRoot returns the worktree root
// (not the main repository root) when inside a git worktree. This is critical
// for bd-745 - ensuring database discovery works correctly in worktrees.
func TestFindGitRoot_Worktree(t *testing.T) {
	// Create temporary directory for our test
	tmpDir, err := os.MkdirTemp("", "issuegraph-worktree-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize a git repository
	mainRepoDir := filepath.Join(tmpDir, "main-repo")
	if err := os.MkdirAll(mainRepoDir, 0755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Configure git user for the test repo (required for commits)
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = mainRepoDir
	_ = cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = mainRepoDir
	_ = cmd.Run()

	// Create an initial commit (required for worktree)
	dummyFile := filepath.Join(mainRepoDir, "README.md")
	if err := os.WriteFile(dummyFile, []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Create a worktree
	worktreeDir := filepath.Join(tmpDir, "worktree")
	cmd = exec.Command("git", "worktree", "add", worktreeDir, "HEAD")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git worktree add failed: %v", err)
	}
	defer func() {
		// Clean up worktree
		cmd := exec.Command("git", "worktree", "remove", worktreeDir)
		cmd.Dir = mainRepoDir
		_ = cmd.Run()
	}()

	// Change to the worktree directory
	t.Chdir(worktreeDir)
	git.ResetCaches() // Reset after chdir for caching tests

	// findGitRoot should return the WORKTREE root, not the main repo root
	result := findGitRoot()

	// Resolve symlinks for comparison
	resultResolved, _ := filepath.EvalSymlinks(result)
	worktreeDirResolved, _ := filepath.EvalSymlinks(worktreeDir)
	mainRepoDirResolved, _ := filepath.EvalSymlinks(mainRepoDir)

	if resultResolved != worktreeDirResolved {
		t.Errorf("findGitRoot() = %q, want worktree %q (not main repo %q)", result, worktreeDir, mainRepoDir)
	}

	// Additional verification: ensure we're NOT returning the main repo
	if resultResolved == mainRepoDirResolved {
		t.Errorf("findGitRoot() returned main repo %q instead of worktree %q - worktree detection is broken!", mainRepoDir, worktreeDir)
	}
}

// TestFindGitRoot_NotGitRepo tests that findGitRoot returns an empty string
// when not inside a git repository.
func TestFindGitRoot_NotGitRepo(t *testing.T) {
	// Create temporary directory that is NOT a git repo
	tmpDir, err := os.MkdirTemp("", "issuegraph-nogit-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	t.Chdir(tmpDir)
	git.ResetCaches() // Reset after chdir for caching tests

	// findGitRoot should return empty string
	result := findGitRoot()

	if result != "" {
		t.Errorf("findGitRoot() = %q, want empty string (not in git repo)", result)
	}
}

// TestFindIssueGraphDir_Worktree tests that FindIssueGraphDir correctly finds the .issuegraph
// directory within a git worktree, respecting the worktree boundary and not
// searching into the main repository. This is critical for bd-745.
func TestFindIssueGraphDir_Worktree(t *testing.T) {
	// Save original state
	originalEnv := os.Getenv("ISSUEGRAPH_DIR")
	defer func() {
		if originalEnv != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalEnv)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	}()
	os.Unsetenv("ISSUEGRAPH_DIR")

	// Create temporary directory for our test
	tmpDir, err := os.MkdirTemp("", "issuegraph-worktree-finddir-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize main git repository
	mainRepoDir := filepath.Join(tmpDir, "main-repo")
	if err := os.MkdirAll(mainRepoDir, 0755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = mainRepoDir
	_ = cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = mainRepoDir
	_ = cmd.Run()

	// Create .issuegraph directory in main repo with a database
	mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
	if err := os.MkdirAll(mainIssueGraphDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainIssueGraphDir, "issuegraph.db"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	// Create initial commit
	if err := os.WriteFile(filepath.Join(mainRepoDir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("git", "add", "-A")
	cmd.Dir = mainRepoDir
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Create a worktree
	worktreeDir := filepath.Join(tmpDir, "worktree")
	cmd = exec.Command("git", "worktree", "add", worktreeDir, "HEAD")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git worktree add failed: %v", err)
	}
	defer func() {
		cmd := exec.Command("git", "worktree", "remove", worktreeDir)
		cmd.Dir = mainRepoDir
		_ = cmd.Run()
	}()

	// Create .issuegraph directory in worktree with its own database
	worktreeIssueGraphDir := filepath.Join(worktreeDir, ".issuegraph")
	if err := os.MkdirAll(worktreeIssueGraphDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worktreeIssueGraphDir, "issuegraph.db"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	// Change to worktree
	t.Chdir(worktreeDir)
	git.ResetCaches() // Reset after chdir for caching tests

	// FindIssueGraphDir should find the worktree's own .issuegraph when it has project files (GH#2190)
	result := FindIssueGraphDir()

	// Resolve symlinks for comparison
	resultResolved, _ := filepath.EvalSymlinks(result)
	worktreeIssueGraphDirResolved, _ := filepath.EvalSymlinks(worktreeIssueGraphDir)

	if resultResolved != worktreeIssueGraphDirResolved {
		t.Errorf("FindIssueGraphDir() = %q, want worktree .issuegraph %q (separate-DB mode)", result, worktreeIssueGraphDir)
	}
}

// TestFindIssueGraphDir_WorktreeRedirectOverride tests that when a worktree has its
// own .issuegraph/redirect, it takes priority over the main repo's .issuegraph directory.
// This enables per-worktree topic selection via timvisher_EXP_issuegraph_topics set.
func TestFindIssueGraphDir_WorktreeRedirectOverride(t *testing.T) {
	// Save original state
	originalEnv := os.Getenv("ISSUEGRAPH_DIR")
	defer func() {
		if originalEnv != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalEnv)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	}()
	os.Unsetenv("ISSUEGRAPH_DIR")

	// Create temporary directory for our test
	tmpDir, err := os.MkdirTemp("", "issuegraph-worktree-redirect-override-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize main git repository with .issuegraph containing project files
	mainRepoDir := filepath.Join(tmpDir, "main-repo")
	if err := os.MkdirAll(mainRepoDir, 0755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = mainRepoDir
	_ = cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = mainRepoDir
	_ = cmd.Run()

	// Create .issuegraph in main repo with project files (would normally win)
	mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
	if err := os.MkdirAll(mainIssueGraphDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainIssueGraphDir, "issuegraph.db"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	// Create initial commit
	if err := os.WriteFile(filepath.Join(mainRepoDir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("git", "add", "-A")
	cmd.Dir = mainRepoDir
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Create an external topic .issuegraph directory (the redirect target)
	topicIssueGraphDir := filepath.Join(tmpDir, "topics", "my-topic", ".issuegraph")
	if err := os.MkdirAll(topicIssueGraphDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(topicIssueGraphDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"issuegraph_my_topic"}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a worktree
	worktreeDir := filepath.Join(tmpDir, "worktree")
	cmd = exec.Command("git", "worktree", "add", worktreeDir, "HEAD")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git worktree add failed: %v", err)
	}
	defer func() {
		cmd := exec.Command("git", "worktree", "remove", worktreeDir)
		cmd.Dir = mainRepoDir
		_ = cmd.Run()
	}()

	// Create .issuegraph/redirect in the worktree pointing to the external topic
	worktreeIssueGraphDir := filepath.Join(worktreeDir, ".issuegraph")
	if err := os.MkdirAll(worktreeIssueGraphDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Resolve symlinks for the redirect target (macOS /private/var)
	topicIssueGraphDirResolved, _ := filepath.EvalSymlinks(topicIssueGraphDir)
	if err := os.WriteFile(filepath.Join(worktreeIssueGraphDir, "redirect"), []byte(topicIssueGraphDirResolved+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Change to worktree
	t.Chdir(worktreeDir)
	git.ResetCaches()

	// FindIssueGraphDir should follow the worktree redirect, NOT use main repo's .issuegraph
	result := FindIssueGraphDir()

	resultResolved, _ := filepath.EvalSymlinks(result)
	mainIssueGraphDirResolved, _ := filepath.EvalSymlinks(mainIssueGraphDir)

	if resultResolved == mainIssueGraphDirResolved {
		t.Errorf("FindIssueGraphDir() = main repo .issuegraph %q, want topic .issuegraph %q (worktree redirect should override)", result, topicIssueGraphDirResolved)
	}

	if resultResolved != topicIssueGraphDirResolved {
		t.Errorf("FindIssueGraphDir() = %q, want topic .issuegraph %q (via worktree redirect)", result, topicIssueGraphDirResolved)
	}
}

// TestFindDatabasePath_WorktreeRedirectOverride tests that findDatabaseInTree
// follows a worktree's .issuegraph/redirect before falling back to the main repo.
// This is the database-discovery counterpart of TestFindIssueGraphDir_WorktreeRedirectOverride.
func TestFindDatabasePath_WorktreeRedirectOverride(t *testing.T) {
	originalEnv := os.Getenv("ISSUEGRAPH_DIR")
	defer func() {
		if originalEnv != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalEnv)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	}()
	os.Unsetenv("ISSUEGRAPH_DIR")

	tmpDir, err := os.MkdirTemp("", "issuegraph-db-worktree-redirect-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize main git repo with a .issuegraph database
	mainRepoDir := filepath.Join(tmpDir, "main-repo")
	if err := os.MkdirAll(mainRepoDir, 0755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = mainRepoDir
	_ = cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = mainRepoDir
	_ = cmd.Run()

	// Main repo has a dolt database (would normally win)
	mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
	mainDoltDir := filepath.Join(mainIssueGraphDir, "dolt")
	if err := os.MkdirAll(mainDoltDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(mainRepoDir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("git", "add", "-A")
	cmd.Dir = mainRepoDir
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Create external topic .issuegraph with its own dolt database
	topicIssueGraphDir := filepath.Join(tmpDir, "topics", "my-topic", ".issuegraph")
	topicDoltDir := filepath.Join(topicIssueGraphDir, "dolt")
	if err := os.MkdirAll(topicDoltDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a worktree
	worktreeDir := filepath.Join(tmpDir, "worktree")
	cmd = exec.Command("git", "worktree", "add", worktreeDir, "HEAD")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git worktree add failed: %v", err)
	}
	defer func() {
		cmd := exec.Command("git", "worktree", "remove", worktreeDir)
		cmd.Dir = mainRepoDir
		_ = cmd.Run()
	}()

	// Create .issuegraph/redirect in the worktree pointing to the topic
	worktreeIssueGraphDir := filepath.Join(worktreeDir, ".issuegraph")
	if err := os.MkdirAll(worktreeIssueGraphDir, 0755); err != nil {
		t.Fatal(err)
	}
	topicIssueGraphDirResolved, _ := filepath.EvalSymlinks(topicIssueGraphDir)
	if err := os.WriteFile(filepath.Join(worktreeIssueGraphDir, "redirect"), []byte(topicIssueGraphDirResolved+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(worktreeDir)
	git.ResetCaches()

	result := FindDatabasePath()

	mainDoltResolved, _ := filepath.EvalSymlinks(mainDoltDir)
	topicDoltResolved, _ := filepath.EvalSymlinks(topicDoltDir)

	if result == "" {
		t.Fatal("FindDatabasePath() returned empty, want topic dolt path")
	}

	resultResolved, _ := filepath.EvalSymlinks(result)
	if resultResolved == mainDoltResolved {
		t.Errorf("FindDatabasePath() = main repo dolt %q, want topic dolt %q (worktree redirect should override)", result, topicDoltResolved)
	}
	if resultResolved != topicDoltResolved {
		t.Errorf("FindDatabasePath() = %q, want topic dolt %q (via worktree redirect)", result, topicDoltResolved)
	}
}

// TestFindIssueGraphDir_SiblingWorktree tests that FindIssueGraphDir does not escape past
// the worktree boundary when the worktree is a sibling of the main repo (not a
// child). This is the regression test for GH#1653.
func TestFindIssueGraphDir_SiblingWorktree(t *testing.T) {
	// Save original state
	originalEnv := os.Getenv("ISSUEGRAPH_DIR")
	defer func() {
		if originalEnv != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalEnv)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	}()
	os.Unsetenv("ISSUEGRAPH_DIR")

	// Create temporary directory for our test
	tmpDir, err := os.MkdirTemp("", "issuegraph-sibling-worktree-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Resolve symlinks (macOS /var -> /private/var)
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Structure: tmpDir/main-repo  (git repo with .issuegraph/)
	//            tmpDir/sibling-wt (worktree, sibling of main-repo)
	//            tmpDir/.issuegraph/    (UNRELATED issuegraph dir that should NOT be found)

	mainRepoDir := filepath.Join(tmpDir, "main-repo")
	if err := os.MkdirAll(mainRepoDir, 0755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = mainRepoDir
	_ = cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = mainRepoDir
	_ = cmd.Run()

	// Create .issuegraph in main repo
	mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
	if err := os.MkdirAll(mainIssueGraphDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainIssueGraphDir, "issuegraph.db"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	// Initial commit
	if err := os.WriteFile(filepath.Join(mainRepoDir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("git", "add", "-A")
	cmd.Dir = mainRepoDir
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Create sibling worktree (NOT a child of main-repo)
	siblingDir := filepath.Join(tmpDir, "sibling-wt")
	cmd = exec.Command("git", "worktree", "add", siblingDir, "HEAD")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git worktree add failed: %v", err)
	}
	defer func() {
		cmd := exec.Command("git", "worktree", "remove", siblingDir)
		cmd.Dir = mainRepoDir
		_ = cmd.Run()
	}()

	// Remove worktree's .issuegraph/ (came from checkout) so it must fall back to main repo
	// This simulates the real-world case where .issuegraph is in .gitignore
	_ = os.RemoveAll(filepath.Join(siblingDir, ".issuegraph"))

	// Create an UNRELATED .issuegraph/ in the parent directory (tmpDir)
	// Before the fix, the walk would escape past worktreeRoot and find this
	unrelatedIssueGraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(unrelatedIssueGraphDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unrelatedIssueGraphDir, "issuegraph.db"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	// Change to the sibling worktree
	t.Chdir(siblingDir)
	git.ResetCaches()

	result := FindIssueGraphDir()

	resultResolved, _ := filepath.EvalSymlinks(result)
	mainIssueGraphDirResolved, _ := filepath.EvalSymlinks(mainIssueGraphDir)
	unrelatedResolved, _ := filepath.EvalSymlinks(unrelatedIssueGraphDir)

	// Should find main repo's .issuegraph (via the mainRepoRoot check in step 2)
	if resultResolved != mainIssueGraphDirResolved {
		t.Errorf("FindIssueGraphDir() = %q, want main repo .issuegraph %q", result, mainIssueGraphDir)
	}

	// Must NOT find the unrelated parent .issuegraph
	if resultResolved == unrelatedResolved {
		t.Errorf("FindIssueGraphDir() escaped worktree boundary and found unrelated %q", unrelatedIssueGraphDir)
	}
}

// TestFindDatabasePath_Worktree tests that FindDatabasePath correctly finds the
// shared database in the main repository when a worktree does NOT have its own
// .issuegraph directory. This is the key test for bd-745 - worktrees should share
// the same .issuegraph database when the worktree has no separate-DB init.
func TestFindDatabasePath_Worktree(t *testing.T) {
	// Save original state
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

	// Create temporary directory for our test
	tmpDir, err := os.MkdirTemp("", "issuegraph-worktree-finddb-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize main git repository
	mainRepoDir := filepath.Join(tmpDir, "main-repo")
	if err := os.MkdirAll(mainRepoDir, 0755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = mainRepoDir
	_ = cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = mainRepoDir
	_ = cmd.Run()

	// Create .issuegraph directory in main repo with dolt database
	mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
	if err := os.MkdirAll(mainIssueGraphDir, 0755); err != nil {
		t.Fatal(err)
	}
	mainDoltDir := filepath.Join(mainIssueGraphDir, "dolt")
	if err := os.MkdirAll(mainDoltDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create initial commit
	if err := os.WriteFile(filepath.Join(mainRepoDir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("git", "add", "-A")
	cmd.Dir = mainRepoDir
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Create a worktree
	worktreeDir := filepath.Join(tmpDir, "worktree")
	cmd = exec.Command("git", "worktree", "add", worktreeDir, "HEAD")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git worktree add failed: %v", err)
	}
	defer func() {
		cmd := exec.Command("git", "worktree", "remove", worktreeDir)
		cmd.Dir = mainRepoDir
		_ = cmd.Run()
	}()

	// Remove worktree's .issuegraph/ (came from checkout) so it falls back to main repo
	// This simulates the real-world case where .issuegraph is in .gitignore
	_ = os.RemoveAll(filepath.Join(worktreeDir, ".issuegraph"))

	// Change to worktree subdirectory
	worktreeSubDir := filepath.Join(worktreeDir, "sub", "nested")
	if err := os.MkdirAll(worktreeSubDir, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(worktreeSubDir)
	git.ResetCaches() // Reset after chdir for caching tests

	// FindDatabasePath should find the main repo's shared database
	result := FindDatabasePath()

	// Resolve symlinks for comparison
	resultResolved, _ := filepath.EvalSymlinks(result)
	mainDoltResolved, _ := filepath.EvalSymlinks(mainDoltDir)

	if resultResolved != mainDoltResolved {
		t.Errorf("FindDatabasePath() = %q, want main repo shared db %q", result, mainDoltDir)
	}
}

// TestFindDatabasePath_WorktreeSeparateDB tests that FindDatabasePath correctly
// finds the worktree's own .issuegraph database when the worktree has been bd-init'd
// with its own separate database (no redirect file). This is the fix for GH#2190.
func TestFindDatabasePath_WorktreeSeparateDB(t *testing.T) {
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

	tmpDir, err := os.MkdirTemp("", "issuegraph-worktree-separatedb-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize main git repository with its own .issuegraph
	mainRepoDir := filepath.Join(tmpDir, "main-repo")
	if err := os.MkdirAll(mainRepoDir, 0755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = mainRepoDir
	_ = cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = mainRepoDir
	_ = cmd.Run()

	// Main repo .issuegraph with dolt database
	mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
	if err := os.MkdirAll(filepath.Join(mainIssueGraphDir, "dolt"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create initial commit (don't commit .issuegraph - it's normally in .gitignore)
	if err := os.WriteFile(filepath.Join(mainRepoDir, ".gitignore"), []byte(".issuegraph/\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainRepoDir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("git", "add", "-A")
	cmd.Dir = mainRepoDir
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Create a worktree
	worktreeDir := filepath.Join(tmpDir, "worktree")
	cmd = exec.Command("git", "worktree", "add", worktreeDir, "HEAD")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git worktree add failed: %v", err)
	}
	defer func() {
		cmd := exec.Command("git", "worktree", "remove", worktreeDir)
		cmd.Dir = mainRepoDir
		_ = cmd.Run()
	}()

	// Simulate "issuegraph init" in the worktree: create .issuegraph with its own dolt database
	worktreeIssueGraphDir := filepath.Join(worktreeDir, ".issuegraph")
	worktreeDoltDir := filepath.Join(worktreeIssueGraphDir, "dolt")
	if err := os.MkdirAll(worktreeDoltDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worktreeIssueGraphDir, "metadata.json"),
		[]byte(`{"backend":"dolt","dolt_database":"issuegraph_worktree"}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Change to worktree
	t.Chdir(worktreeDir)
	git.ResetCaches()

	// FindDatabasePath should find the worktree's own database, NOT the main repo's
	result := FindDatabasePath()

	resultResolved, _ := filepath.EvalSymlinks(result)
	worktreeDoltResolved, _ := filepath.EvalSymlinks(worktreeDoltDir)
	mainDoltResolved, _ := filepath.EvalSymlinks(filepath.Join(mainIssueGraphDir, "dolt"))

	if resultResolved == mainDoltResolved {
		t.Errorf("FindDatabasePath() = main repo db %q, want worktree db %q (separate-DB mode)", result, worktreeDoltDir)
	}

	if resultResolved != worktreeDoltResolved {
		t.Errorf("FindDatabasePath() = %q, want worktree db %q (separate-DB mode)", result, worktreeDoltDir)
	}
}

// TestFindDatabasePath_WorktreeNoLocalDB tests that when a worktree does NOT have
// its own .issuegraph directory, FindDatabasePath finds the shared database in the main
// repository. This tests the "shared database" behavior for worktrees.
func TestFindDatabasePath_WorktreeNoLocalDB(t *testing.T) {
	// Save original state
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

	// Create temporary directory for our test
	tmpDir, err := os.MkdirTemp("", "issuegraph-worktree-nodb-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize main git repository
	mainRepoDir := filepath.Join(tmpDir, "main-repo")
	if err := os.MkdirAll(mainRepoDir, 0755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = mainRepoDir
	_ = cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = mainRepoDir
	_ = cmd.Run()

	// Create .issuegraph directory in main repo with dolt database
	mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
	if err := os.MkdirAll(mainIssueGraphDir, 0755); err != nil {
		t.Fatal(err)
	}
	mainDoltDir := filepath.Join(mainIssueGraphDir, "dolt")
	if err := os.MkdirAll(mainDoltDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create initial commit
	if err := os.WriteFile(filepath.Join(mainRepoDir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("git", "add", "-A")
	cmd.Dir = mainRepoDir
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Create a worktree WITHOUT a .issuegraph directory
	worktreeDir := filepath.Join(tmpDir, "worktree")
	cmd = exec.Command("git", "worktree", "add", worktreeDir, "HEAD")
	cmd.Dir = mainRepoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git worktree add failed: %v", err)
	}
	defer func() {
		cmd := exec.Command("git", "worktree", "remove", worktreeDir)
		cmd.Dir = mainRepoDir
		_ = cmd.Run()
	}()

	// Note: We do NOT create .issuegraph in the worktree
	// The worktree got .issuegraph from the commit, so we need to remove it
	worktreeIssueGraphDir := filepath.Join(worktreeDir, ".issuegraph")
	if err := os.RemoveAll(worktreeIssueGraphDir); err != nil {
		// May not exist, that's fine
	}

	// Change to worktree
	t.Chdir(worktreeDir)
	git.ResetCaches() // Reset after chdir for caching tests

	// FindDatabasePath should find the main repo's shared database
	result := FindDatabasePath()

	// Resolve symlinks for comparison
	resultResolved, _ := filepath.EvalSymlinks(result)
	mainDoltResolved, _ := filepath.EvalSymlinks(mainDoltDir)

	if resultResolved != mainDoltResolved {
		t.Errorf("FindDatabasePath() = %q, want main repo shared db %q", result, mainDoltDir)
	}
}

// writeMetadataJSON writes a metadata.json file to the given .issuegraph directory.
func writeMetadataJSON(t *testing.T, issuegraphDir string, cfg *configfile.Config) {
	t.Helper()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal metadata.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(issuegraphDir, "metadata.json"), data, 0644); err != nil {
		t.Fatalf("failed to write metadata.json: %v", err)
	}
}

// TestResolveRedirect_PreservesSourceDatabase tests that ResolveRedirect captures
// the source rig's dolt_database from metadata.json before following a redirect.
// When a source directory has a redirect to a shared directory with a different
// dolt_database, the source database name must be preserved.
func TestResolveRedirect_PreservesSourceDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	// Resolve symlinks for macOS /private/var path consistency
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	// Create source .issuegraph with dolt_database: "lola"
	sourceDir := filepath.Join(tmpDir, "lola", ".issuegraph")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeMetadataJSON(t, sourceDir, &configfile.Config{
		Database:     "issuegraph.db",
		DoltMode:     "server",
		DoltDatabase: "lola",
	})

	// Create target (shared) .issuegraph with dolt_database: "hq"
	targetDir := filepath.Join(tmpDir, "town", ".issuegraph")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeMetadataJSON(t, targetDir, &configfile.Config{
		Database:     "issuegraph.db",
		DoltMode:     "server",
		DoltDatabase: "hq",
	})

	// Write redirect from source to target
	if err := os.WriteFile(filepath.Join(sourceDir, "redirect"), []byte(targetDir+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	info := ResolveRedirect(sourceDir)

	// Verify redirect was followed
	if !info.WasRedirected {
		t.Error("expected WasRedirected=true")
	}

	// Verify source database is preserved
	if info.SourceDatabase != "lola" {
		t.Errorf("SourceDatabase = %q, want %q", info.SourceDatabase, "lola")
	}

	// Verify source and target dirs are correct
	sourceResolved, _ := filepath.EvalSymlinks(sourceDir)
	targetResolved, _ := filepath.EvalSymlinks(targetDir)
	infoSourceResolved, _ := filepath.EvalSymlinks(info.SourceDir)
	infoTargetResolved, _ := filepath.EvalSymlinks(info.TargetDir)

	if infoSourceResolved != sourceResolved {
		t.Errorf("SourceDir = %q, want %q", info.SourceDir, sourceDir)
	}
	if infoTargetResolved != targetResolved {
		t.Errorf("TargetDir = %q, want %q", info.TargetDir, targetDir)
	}
}

// TestResolveRedirect_NoRedirect tests that ResolveRedirect works correctly when
// there is no redirect file (source and target are the same).
func TestResolveRedirect_NoRedirect(t *testing.T) {
	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeMetadataJSON(t, issuegraphDir, &configfile.Config{
		Database:     "issuegraph.db",
		DoltDatabase: "mydb",
	})

	info := ResolveRedirect(issuegraphDir)

	if info.WasRedirected {
		t.Error("expected WasRedirected=false when no redirect file exists")
	}
	if info.SourceDatabase != "mydb" {
		t.Errorf("SourceDatabase = %q, want %q", info.SourceDatabase, "mydb")
	}

	issuegraphDirResolved, _ := filepath.EvalSymlinks(issuegraphDir)
	infoSourceResolved, _ := filepath.EvalSymlinks(info.SourceDir)
	infoTargetResolved, _ := filepath.EvalSymlinks(info.TargetDir)

	if infoSourceResolved != issuegraphDirResolved {
		t.Errorf("SourceDir = %q, want %q", info.SourceDir, issuegraphDir)
	}
	if infoTargetResolved != issuegraphDirResolved {
		t.Errorf("TargetDir = %q, want same as source %q when no redirect", info.TargetDir, issuegraphDir)
	}
}

// TestResolveRedirect_NoSourceMetadata tests that ResolveRedirect handles a source
// directory with no metadata.json (SourceDatabase should be empty).
func TestResolveRedirect_NoSourceMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	sourceDir := filepath.Join(tmpDir, "rig", ".issuegraph")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	// No metadata.json in source

	targetDir := filepath.Join(tmpDir, "town", ".issuegraph")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeMetadataJSON(t, targetDir, &configfile.Config{
		Database:     "issuegraph.db",
		DoltDatabase: "hq",
	})

	// Write redirect from source to target
	if err := os.WriteFile(filepath.Join(sourceDir, "redirect"), []byte(targetDir+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	info := ResolveRedirect(sourceDir)

	if !info.WasRedirected {
		t.Error("expected WasRedirected=true")
	}
	if info.SourceDatabase != "" {
		t.Errorf("SourceDatabase = %q, want empty string when no source metadata", info.SourceDatabase)
	}
}

// TestResolveRedirect_SourceHasNoDoltDatabase tests that ResolveRedirect handles
// a source whose metadata.json exists but has no dolt_database field.
func TestResolveRedirect_SourceHasNoDoltDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	sourceDir := filepath.Join(tmpDir, "rig", ".issuegraph")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Source metadata has no dolt_database
	writeMetadataJSON(t, sourceDir, &configfile.Config{
		Database: "issuegraph.db",
	})

	targetDir := filepath.Join(tmpDir, "town", ".issuegraph")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeMetadataJSON(t, targetDir, &configfile.Config{
		Database:     "issuegraph.db",
		DoltDatabase: "hq",
	})

	if err := os.WriteFile(filepath.Join(sourceDir, "redirect"), []byte(targetDir+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	info := ResolveRedirect(sourceDir)

	if !info.WasRedirected {
		t.Error("expected WasRedirected=true")
	}
	// No dolt_database in source => SourceDatabase should be empty
	// This means the target's dolt_database will be used (no override)
	if info.SourceDatabase != "" {
		t.Errorf("SourceDatabase = %q, want empty string when source has no dolt_database", info.SourceDatabase)
	}
}

// TestResolveRedirect_SourceDatabaseAvailableForRouting tests that ResolveRedirect
// captures the source database so callers (like routing code) can set the env var.
// The env var is NOT set by FollowRedirect itself (that caused hangs from circular
// configfile.Load calls). Instead, routing callers use ResolveRedirect and set
// ISSUEGRAPH_DOLT_SERVER_DATABASE explicitly.
func TestResolveRedirect_SourceDatabaseAvailableForRouting(t *testing.T) {
	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	// Create source .issuegraph with dolt_database: "lola"
	sourceDir := filepath.Join(tmpDir, "lola", ".issuegraph")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeMetadataJSON(t, sourceDir, &configfile.Config{
		DoltDatabase: "lola",
	})

	// Create target (shared) .issuegraph with dolt_database: "hq"
	targetDir := filepath.Join(tmpDir, "town", ".issuegraph")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeMetadataJSON(t, targetDir, &configfile.Config{
		DoltDatabase: "hq",
	})

	// Write redirect
	if err := os.WriteFile(filepath.Join(sourceDir, "redirect"), []byte(targetDir+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	info := ResolveRedirect(sourceDir)

	// Source database should be available for routing callers to use
	if info.SourceDatabase != "lola" {
		t.Errorf("SourceDatabase = %q, want %q", info.SourceDatabase, "lola")
	}
	if !info.WasRedirected {
		t.Error("expected WasRedirected=true")
	}
}

func TestFindIssueGraphDir_BareParentWorktreeFallback(t *testing.T) {
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

	bareDir, worktreeDir := setupBareParentWorktree(t)
	bareIssueGraphDir := filepath.Join(bareDir, ".issuegraph")
	if err := os.MkdirAll(filepath.Join(bareIssueGraphDir, "dolt"), 0755); err != nil {
		t.Fatal(err)
	}

	t.Chdir(worktreeDir)
	git.ResetCaches()

	result := FindIssueGraphDir()
	resultResolved, _ := filepath.EvalSymlinks(result)
	bareIssueGraphResolved, _ := filepath.EvalSymlinks(bareIssueGraphDir)
	if resultResolved != bareIssueGraphResolved {
		t.Errorf("FindIssueGraphDir() = %q, want bare parent .issuegraph %q", result, bareIssueGraphDir)
	}
}

func TestFindDatabasePath_BareParentWorktreeFallback(t *testing.T) {
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

	bareDir, worktreeDir := setupBareParentWorktree(t)
	bareDoltDir := filepath.Join(bareDir, ".issuegraph", "dolt")
	if err := os.MkdirAll(bareDoltDir, 0755); err != nil {
		t.Fatal(err)
	}

	t.Chdir(worktreeDir)
	git.ResetCaches()

	result := FindDatabasePath()
	resultResolved, _ := filepath.EvalSymlinks(result)
	bareDoltResolved, _ := filepath.EvalSymlinks(bareDoltDir)
	if resultResolved != bareDoltResolved {
		t.Errorf("FindDatabasePath() = %q, want bare parent db %q", result, bareDoltDir)
	}
}

func TestResolveIssueGraphDirForRepo(t *testing.T) {
	t.Run("local issuegraph dir preferred for plain repo path", func(t *testing.T) {
		repoDir := t.TempDir()
		localIssueGraphDir := filepath.Join(repoDir, ".issuegraph")
		if err := os.MkdirAll(localIssueGraphDir, 0o755); err != nil {
			t.Fatal(err)
		}

		got := ResolveIssueGraphDirForRepo(repoDir)
		want := utils.CanonicalizePath(localIssueGraphDir)
		if got != want {
			t.Fatalf("ResolveIssueGraphDirForRepo() = %q, want %q", got, want)
		}
	})

	t.Run("regular worktree falls back to main repo issuegraph dir", func(t *testing.T) {
		mainRepoDir, worktreeDir := setupRegularWorktreeRepo(t)
		mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
		if err := os.MkdirAll(mainIssueGraphDir, 0o755); err != nil {
			t.Fatal(err)
		}

		got := ResolveIssueGraphDirForRepo(worktreeDir)
		want := utils.CanonicalizePath(mainIssueGraphDir)
		if got != want {
			t.Fatalf("ResolveIssueGraphDirForRepo() = %q, want %q", got, want)
		}
	})

	t.Run("worktree local issuegraph dir overrides shared fallback", func(t *testing.T) {
		mainRepoDir, worktreeDir := setupRegularWorktreeRepo(t)
		mainIssueGraphDir := filepath.Join(mainRepoDir, ".issuegraph")
		if err := os.MkdirAll(mainIssueGraphDir, 0o755); err != nil {
			t.Fatal(err)
		}

		worktreeIssueGraphDir := filepath.Join(worktreeDir, ".issuegraph")
		if err := os.MkdirAll(worktreeIssueGraphDir, 0o755); err != nil {
			t.Fatal(err)
		}

		got := ResolveIssueGraphDirForRepo(worktreeDir)
		want := utils.CanonicalizePath(worktreeIssueGraphDir)
		if got != want {
			t.Fatalf("ResolveIssueGraphDirForRepo() = %q, want %q", got, want)
		}
	})

	t.Run("no issuegraph anywhere falls back to local repo path", func(t *testing.T) {
		repoDir := filepath.Join(t.TempDir(), "repo")
		if err := os.MkdirAll(repoDir, 0o755); err != nil {
			t.Fatal(err)
		}
		runGitInDir(t, repoDir, "init")
		runGitInDir(t, repoDir, "config", "user.email", "test@example.com")
		runGitInDir(t, repoDir, "config", "user.name", "Test User")

		got := ResolveIssueGraphDirForRepo(repoDir)
		want := filepath.Join(utils.CanonicalizePath(repoDir), ".issuegraph")
		if got != want {
			t.Fatalf("ResolveIssueGraphDirForRepo() = %q, want %q", got, want)
		}
	})
}

func TestWorktreeFallbackIssueGraphDirForRepo(t *testing.T) {
	t.Run("plain repo returns empty string", func(t *testing.T) {
		repoDir := filepath.Join(t.TempDir(), "repo")
		if err := os.MkdirAll(repoDir, 0o755); err != nil {
			t.Fatal(err)
		}
		runGitInDir(t, repoDir, "init")
		runGitInDir(t, repoDir, "config", "user.email", "test@example.com")
		runGitInDir(t, repoDir, "config", "user.name", "Test User")

		got := worktreeFallbackIssueGraphDirForRepo(repoDir)
		if got != "" {
			t.Fatalf("worktreeFallbackIssueGraphDirForRepo() = %q, want empty string", got)
		}
	})

	t.Run("regular worktree uses parent repo issuegraph dir", func(t *testing.T) {
		mainRepoDir, worktreeDir := setupRegularWorktreeRepo(t)

		got := worktreeFallbackIssueGraphDirForRepo(worktreeDir)
		mainRepoResolved, err := filepath.EvalSymlinks(mainRepoDir)
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(mainRepoResolved, ".issuegraph")
		if got != want {
			t.Fatalf("worktreeFallbackIssueGraphDirForRepo() = %q, want %q", got, want)
		}
	})

	t.Run("bare-parent worktree uses common-dir issuegraph dir", func(t *testing.T) {
		bareDir, worktreeDir := setupBareParentWorktree(t)

		got := worktreeFallbackIssueGraphDirForRepo(worktreeDir)
		bareDirResolved, err := filepath.EvalSymlinks(bareDir)
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(bareDirResolved, ".issuegraph")
		if got != want {
			t.Fatalf("worktreeFallbackIssueGraphDirForRepo() = %q, want %q", got, want)
		}
	})
}

func TestGitPathForRepo(t *testing.T) {
	repoDir := t.TempDir()

	t.Run("empty path returns empty", func(t *testing.T) {
		if got := gitPathForRepo(repoDir, ""); got != "" {
			t.Fatalf("gitPathForRepo() = %q, want empty string", got)
		}
	})

	t.Run("relative path is resolved under repo", func(t *testing.T) {
		got := gitPathForRepo(repoDir, ".git/worktrees/feature")
		want := utils.CanonicalizePath(filepath.Join(repoDir, ".git/worktrees/feature"))
		if got != want {
			t.Fatalf("gitPathForRepo() = %q, want %q", got, want)
		}
	})

	t.Run("absolute path is canonicalized unchanged", func(t *testing.T) {
		absPath := filepath.Join(repoDir, ".git")
		got := gitPathForRepo(repoDir, absPath)
		want := utils.CanonicalizePath(absPath)
		if got != want {
			t.Fatalf("gitPathForRepo() = %q, want %q", got, want)
		}
	})
}

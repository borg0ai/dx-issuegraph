package issuegraph

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/git"
)

// TestFindDatabaseInIssueGraphDir_DoltDir tests finding the dolt database directory
func TestFindDatabaseInIssueGraphDir_DoltDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create dolt directory (the canonical database location)
	doltDir := filepath.Join(tmpDir, "dolt")
	if err := os.MkdirAll(doltDir, 0755); err != nil {
		t.Fatal(err)
	}

	result := findDatabaseInIssueGraphDir(tmpDir, false)

	resultResolved, _ := filepath.EvalSymlinks(result)
	expectedResolved, _ := filepath.EvalSymlinks(doltDir)
	if resultResolved != expectedResolved {
		t.Errorf("findDatabaseInIssueGraphDir() = %q, want %q", result, doltDir)
	}
}

// TestFindDatabaseInIssueGraphDir_NoDatabase tests empty directory
func TestFindDatabaseInIssueGraphDir_NoDatabase(t *testing.T) {
	tmpDir := t.TempDir()

	result := findDatabaseInIssueGraphDir(tmpDir, false)
	if result != "" {
		t.Errorf("findDatabaseInIssueGraphDir() = %q, want empty string", result)
	}
}

// TestFindDatabaseInIssueGraphDir_DoltFallback tests fallback to dolt dir without metadata.json
func TestFindDatabaseInIssueGraphDir_DoltFallback(t *testing.T) {
	tmpDir := t.TempDir()

	// Create dolt directory without metadata.json
	doltDir := filepath.Join(tmpDir, "dolt")
	if err := os.MkdirAll(doltDir, 0755); err != nil {
		t.Fatal(err)
	}

	result := findDatabaseInIssueGraphDir(tmpDir, false)

	resultResolved, _ := filepath.EvalSymlinks(result)
	expectedResolved, _ := filepath.EvalSymlinks(doltDir)
	if resultResolved != expectedResolved {
		t.Errorf("findDatabaseInIssueGraphDir() = %q, want %q", result, doltDir)
	}
}

// TestFindDatabaseInIssueGraphDir_DoltBackend tests the Dolt backend path in metadata.json
func TestFindDatabaseInIssueGraphDir_DoltBackend(t *testing.T) {
	tmpDir := t.TempDir()

	// Create metadata.json with dolt backend
	metadataContent := `{"backend": "dolt"}`
	if err := os.WriteFile(filepath.Join(tmpDir, "metadata.json"), []byte(metadataContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create the dolt directory
	doltDir := filepath.Join(tmpDir, "dolt")
	if err := os.MkdirAll(doltDir, 0755); err != nil {
		t.Fatal(err)
	}

	result := findDatabaseInIssueGraphDir(tmpDir, false)

	resultResolved, _ := filepath.EvalSymlinks(result)
	expectedResolved, _ := filepath.EvalSymlinks(doltDir)
	if resultResolved != expectedResolved {
		t.Errorf("findDatabaseInIssueGraphDir() with dolt = %q, want %q", result, doltDir)
	}
}

// TestFindAllDatabases_Unit tests FindAllDatabases without the integration tag
func TestFindAllDatabases_Unit(t *testing.T) {
	// Save original env
	originalDir := os.Getenv("ISSUEGRAPH_DIR")
	t.Cleanup(func() {
		if originalDir != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalDir)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
		git.ResetCaches()
	})
	os.Unsetenv("ISSUEGRAPH_DIR")

	t.Run("finds closest database", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpDir, _ = filepath.EvalSymlinks(tmpDir)

		issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
		if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
			t.Fatal(err)
		}
		// Create metadata.json so hasIssueGraphProjectFiles returns true
		if err := os.WriteFile(filepath.Join(issuegraphDir, "metadata.json"), []byte(`{"backend":"dolt"}`), 0644); err != nil {
			t.Fatal(err)
		}
		// Create dolt directory (canonical database location)
		doltDir := filepath.Join(issuegraphDir, "dolt")
		if err := os.MkdirAll(doltDir, 0755); err != nil {
			t.Fatal(err)
		}

		t.Chdir(tmpDir)
		git.ResetCaches()

		databases := FindAllDatabases()
		if len(databases) != 1 {
			t.Fatalf("expected 1 database, got %d", len(databases))
		}
		if databases[0].IssueGraphDir != issuegraphDir {
			t.Errorf("IssueGraphDir = %q, want %q", databases[0].IssueGraphDir, issuegraphDir)
		}
	})

	t.Run("no databases returns empty slice", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Chdir(tmpDir)
		git.ResetCaches()

		databases := FindAllDatabases()
		if databases == nil {
			t.Error("FindAllDatabases() should never return nil")
		}
		if len(databases) != 0 {
			t.Errorf("expected 0 databases, got %d", len(databases))
		}
	})
}

// TestFindLocalIssueGraphDir tests the findLocalIssueGraphDir function
func TestFindLocalIssueGraphDir_WithISSUEGRAPH_DIR(t *testing.T) {
	tmpDir := t.TempDir()

	// Save and set ISSUEGRAPH_DIR
	originalDir := os.Getenv("ISSUEGRAPH_DIR")
	t.Cleanup(func() {
		if originalDir != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalDir)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
		git.ResetCaches()
	})

	os.Setenv("ISSUEGRAPH_DIR", tmpDir)

	result := findLocalIssueGraphDir()
	resultResolved, _ := filepath.EvalSymlinks(result)
	expectedResolved, _ := filepath.EvalSymlinks(tmpDir)
	if resultResolved != expectedResolved {
		t.Errorf("findLocalIssueGraphDir() = %q, want %q", result, tmpDir)
	}
}

// TestFindLocalIssueGraphDir_WalksUpTree tests that findLocalIssueGraphDir walks up the directory tree
func TestFindLocalIssueGraphDir_WalksUpTree(t *testing.T) {
	originalDir := os.Getenv("ISSUEGRAPH_DIR")
	t.Cleanup(func() {
		if originalDir != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalDir)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
		git.ResetCaches()
	})
	os.Unsetenv("ISSUEGRAPH_DIR")

	tmpDir := t.TempDir()

	// Create .issuegraph dir at root
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create nested subdir
	subDir := filepath.Join(tmpDir, "a", "b", "c")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	t.Chdir(subDir)
	git.ResetCaches()

	result := findLocalIssueGraphDir()

	resultResolved, _ := filepath.EvalSymlinks(result)
	expectedResolved, _ := filepath.EvalSymlinks(issuegraphDir)
	if resultResolved != expectedResolved {
		t.Errorf("findLocalIssueGraphDir() = %q, want %q", result, issuegraphDir)
	}
}

// TestRepoContext_GitCmdCWD tests the GitCmdCWD method
func TestRepoContext_GitCmdCWD(t *testing.T) {
	tmpDir := t.TempDir()
	if err := initGitRepo(tmpDir); err != nil {
		t.Skipf("git not available: %v", err)
	}

	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create metadata.json so hasIssueGraphProjectFiles returns true
	if err := os.WriteFile(filepath.Join(issuegraphDir, "metadata.json"), []byte(`{"backend":"dolt"}`), 0644); err != nil {
		t.Fatal(err)
	}
	// Create dolt directory for database detection
	if err := os.MkdirAll(filepath.Join(issuegraphDir, "dolt"), 0755); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		ResetCaches()
		git.ResetCaches()
	})

	rc, err := GetRepoContextForWorkspace(tmpDir)
	if err != nil {
		t.Fatalf("GetRepoContextForWorkspace failed: %v", err)
	}

	// GitCmdCWD should create a command with Dir set to CWDRepoRoot
	cmd := rc.GitCmdCWD(t.Context(), "status")
	if cmd.Dir == "" {
		t.Error("GitCmdCWD().Dir should not be empty when CWDRepoRoot is set")
	}

	resolvedDir, _ := filepath.EvalSymlinks(cmd.Dir)
	resolvedExpected, _ := filepath.EvalSymlinks(rc.CWDRepoRoot)
	if resolvedDir != resolvedExpected {
		t.Errorf("GitCmdCWD().Dir = %q, want %q", cmd.Dir, rc.CWDRepoRoot)
	}

	// The command should actually run
	output, err := cmd.Output()
	if err != nil {
		t.Errorf("GitCmdCWD command failed: %v", err)
	}
	_ = output
}

// TestRepoContext_GitCmdCWD_NoCWDRepo tests GitCmdCWD when CWDRepoRoot is empty
func TestRepoContext_GitCmdCWD_NoCWDRepo(t *testing.T) {
	rc := &RepoContext{
		IssueGraphDir: "/some/path/.issuegraph",
		RepoRoot:      "/some/path",
		CWDRepoRoot:   "", // Not in a git repo
	}

	cmd := rc.GitCmdCWD(t.Context(), "status")
	if cmd.Dir != "" {
		t.Errorf("GitCmdCWD().Dir should be empty when CWDRepoRoot is empty, got %q", cmd.Dir)
	}
}

// TestRepoContext_RelPath tests the RelPath method
func TestRepoContext_RelPath(t *testing.T) {
	rc := &RepoContext{
		RepoRoot: "/home/user/project",
	}

	tests := []struct {
		name     string
		absPath  string
		expected string
		wantErr  bool
	}{
		{
			name:     "file in .issuegraph",
			absPath:  "/home/user/project/.issuegraph/issuegraph.db",
			expected: ".issuegraph/issuegraph.db",
		},
		{
			name:     "file in subdirectory",
			absPath:  "/home/user/project/src/main.go",
			expected: "src/main.go",
		},
		{
			name:     "repo root itself",
			absPath:  "/home/user/project",
			expected: ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := rc.RelPath(tt.absPath)
			if tt.wantErr {
				if err == nil {
					t.Error("RelPath() should have returned error")
				}
				return
			}
			if err != nil {
				t.Fatalf("RelPath() returned error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("RelPath(%q) = %q, want %q", tt.absPath, result, tt.expected)
			}
		})
	}
}

// TestFindDatabasePath_ISSUEGRAPH_DIR_NoDatabase tests FindDatabasePath when ISSUEGRAPH_DIR is set
// but no database exists in it (--no-db mode)
func TestFindDatabasePath_ISSUEGRAPH_DIR_NoDatabase(t *testing.T) {
	originalDir := os.Getenv("ISSUEGRAPH_DIR")
	originalDB := os.Getenv("ISSUEGRAPH_DB")
	t.Cleanup(func() {
		if originalDir != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalDir)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
		if originalDB != "" {
			os.Setenv("ISSUEGRAPH_DB", originalDB)
		} else {
			os.Unsetenv("ISSUEGRAPH_DB")
		}
	})
	os.Unsetenv("ISSUEGRAPH_DB")

	tmpDir := t.TempDir()
	// Set ISSUEGRAPH_DIR to an empty directory (no database)
	os.Setenv("ISSUEGRAPH_DIR", tmpDir)

	t.Chdir(tmpDir)

	result := FindDatabasePath()
	// Should return empty when ISSUEGRAPH_DIR has no database
	if result != "" {
		t.Errorf("FindDatabasePath() = %q, want empty string for no-db mode", result)
	}
}

// TestCheckRedirectInDir tests the checkRedirectInDir helper
func TestCheckRedirectInDir(t *testing.T) {
	t.Run("no redirect file", func(t *testing.T) {
		tmpDir := t.TempDir()
		issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
		if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
			t.Fatal(err)
		}

		info := checkRedirectInDir(issuegraphDir)
		if info.IsRedirected {
			t.Error("IsRedirected should be false without redirect file")
		}
		if info.LocalDir != issuegraphDir {
			t.Errorf("LocalDir = %q, want %q", info.LocalDir, issuegraphDir)
		}
	})

	t.Run("valid redirect", func(t *testing.T) {
		tmpDir := t.TempDir()

		stubDir := filepath.Join(tmpDir, "stub", ".issuegraph")
		if err := os.MkdirAll(stubDir, 0755); err != nil {
			t.Fatal(err)
		}

		targetDir := filepath.Join(tmpDir, "target", ".issuegraph")
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			t.Fatal(err)
		}
		// Target must contain a recognizable database (or metadata.json) for
		// FollowRedirect to honor the redirect (borg0ai/dx-issuegraph#4692 guard).
		if err := os.WriteFile(filepath.Join(targetDir, "issuegraph.db"), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}

		// Write redirect
		if err := os.WriteFile(filepath.Join(stubDir, "redirect"), []byte(targetDir+"\n"), 0644); err != nil {
			t.Fatal(err)
		}

		info := checkRedirectInDir(stubDir)
		if !info.IsRedirected {
			t.Error("IsRedirected should be true with valid redirect")
		}

		targetResolved, _ := filepath.EvalSymlinks(targetDir)
		infoResolved, _ := filepath.EvalSymlinks(info.TargetDir)
		if infoResolved != targetResolved {
			t.Errorf("TargetDir = %q, want %q", info.TargetDir, targetDir)
		}
	})

	t.Run("invalid redirect target", func(t *testing.T) {
		tmpDir := t.TempDir()
		issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
		if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Write redirect to non-existent path
		if err := os.WriteFile(filepath.Join(issuegraphDir, "redirect"), []byte("/nonexistent/path\n"), 0644); err != nil {
			t.Fatal(err)
		}

		info := checkRedirectInDir(issuegraphDir)
		if info.IsRedirected {
			t.Error("IsRedirected should be false for invalid redirect target")
		}
	})
}

// TestFollowRedirect_ChainPrevention tests that redirect chains are not followed
func TestFollowRedirect_ChainPrevention(t *testing.T) {
	tmpDir := t.TempDir()

	// Create first .issuegraph that redirects to second
	dir1 := filepath.Join(tmpDir, "a", ".issuegraph")
	dir2 := filepath.Join(tmpDir, "b", ".issuegraph")
	if err := os.MkdirAll(dir1, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir2, 0755); err != nil {
		t.Fatal(err)
	}
	// dir2 must contain a recognizable database for FollowRedirect to honor
	// the redirect from dir1 (borg0ai/dx-issuegraph#4692 guard).
	if err := os.WriteFile(filepath.Join(dir2, "issuegraph.db"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	// dir1 redirects to dir2
	if err := os.WriteFile(filepath.Join(dir1, "redirect"), []byte(dir2+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// dir2 also has a redirect (chain)
	if err := os.WriteFile(filepath.Join(dir2, "redirect"), []byte("/some/other/path\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Should follow to dir2 but not further (and log warning about chain)
	result := FollowRedirect(dir1)
	resultResolved, _ := filepath.EvalSymlinks(result)
	dir2Resolved, _ := filepath.EvalSymlinks(dir2)
	if resultResolved != dir2Resolved {
		t.Errorf("FollowRedirect() = %q, want %q (chain should stop at first redirect)", result, dir2)
	}
}

// TestFindDatabaseInTree_WithISSUEGRAPH_DIR tests findDatabaseInTree with ISSUEGRAPH_DIR set
func TestFindDatabaseInTree_WithISSUEGRAPH_DIR(t *testing.T) {
	originalDir := os.Getenv("ISSUEGRAPH_DIR")
	originalDB := os.Getenv("ISSUEGRAPH_DB")
	t.Cleanup(func() {
		if originalDir != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalDir)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
		if originalDB != "" {
			os.Setenv("ISSUEGRAPH_DB", originalDB)
		} else {
			os.Unsetenv("ISSUEGRAPH_DB")
		}
		git.ResetCaches()
	})
	os.Unsetenv("ISSUEGRAPH_DB")
	os.Unsetenv("ISSUEGRAPH_DIR")

	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	// Init git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Create .issuegraph with dolt directory
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatal(err)
	}
	doltDir := filepath.Join(issuegraphDir, "dolt")
	if err := os.MkdirAll(doltDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create subdirectory and cd to it
	subDir := filepath.Join(tmpDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(subDir)
	git.ResetCaches()

	result := findDatabaseInTree()
	if result == "" {
		t.Error("findDatabaseInTree() returned empty string, expected to find db")
	}

	resultResolved, _ := filepath.EvalSymlinks(result)
	expectedResolved, _ := filepath.EvalSymlinks(doltDir)
	if resultResolved != expectedResolved {
		t.Errorf("findDatabaseInTree() = %q, want %q", result, doltDir)
	}
}

// TestFindIssueGraphDir_ISSUEGRAPH_DIR_WithProjectFiles tests FindIssueGraphDir with ISSUEGRAPH_DIR env var
func TestFindIssueGraphDir_ISSUEGRAPH_DIR_WithProjectFiles(t *testing.T) {
	originalDir := os.Getenv("ISSUEGRAPH_DIR")
	t.Cleanup(func() {
		if originalDir != "" {
			os.Setenv("ISSUEGRAPH_DIR", originalDir)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	})

	tmpDir := t.TempDir()

	// Create metadata.json (project file that indicates a issuegraph directory)
	if err := os.WriteFile(filepath.Join(tmpDir, "metadata.json"), []byte(`{"backend":"dolt"}`), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("ISSUEGRAPH_DIR", tmpDir)

	result := FindIssueGraphDir()
	resultResolved, _ := filepath.EvalSymlinks(result)
	expectedResolved, _ := filepath.EvalSymlinks(tmpDir)
	if resultResolved != expectedResolved {
		t.Errorf("FindIssueGraphDir() = %q, want %q", result, tmpDir)
	}
}

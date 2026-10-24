package issuegraph

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestFindIssueGraphDir_CwdPriority verifies that a .issuegraph/ directory in cwd takes
// priority over a .issuegraph/ directory at the git worktree root.
//
// Scenario: A "rig" subdirectory has its own .issuegraph/ inside a git worktree
// that also has .issuegraph/ at its root. Before this fix, step 2b
// (git.GetRepoRoot → check .issuegraph/) fired before the cwd walk, grabbing
// the worktree root's .issuegraph/ instead of the rig's local one.
func TestFindIssueGraphDir_CwdPriority(t *testing.T) {
	// Save and restore env
	origIssueGraphDir := os.Getenv("ISSUEGRAPH_DIR")
	t.Cleanup(func() {
		if origIssueGraphDir != "" {
			os.Setenv("ISSUEGRAPH_DIR", origIssueGraphDir)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	})
	os.Unsetenv("ISSUEGRAPH_DIR")

	tmpDir := t.TempDir()

	// Create a git repo (simulating the worktree root)
	cmd := exec.Command("git", "init", tmpDir)
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Create root-level .issuegraph/ with project files (the "wrong" one)
	rootIssueGraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(rootIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootIssueGraphDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"root_db"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootIssueGraphDir, "config.yaml"), []byte("issue_prefix: root\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a rig subdirectory with its own .issuegraph/ (the "right" one)
	rigDir := filepath.Join(tmpDir, "my-rig")
	rigIssueGraphDir := filepath.Join(rigDir, ".issuegraph")
	if err := os.MkdirAll(rigIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rigIssueGraphDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"rig_db"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rigIssueGraphDir, "config.yaml"), []byte("issue_prefix: rig\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// cd into the rig directory
	t.Chdir(rigDir)

	result := FindIssueGraphDir()

	resultResolved, _ := filepath.EvalSymlinks(result)
	expectedResolved, _ := filepath.EvalSymlinks(rigIssueGraphDir)
	if resultResolved != expectedResolved {
		t.Errorf("FindIssueGraphDir() = %q, want %q (rig's .issuegraph should win over root's)", result, rigIssueGraphDir)
	}
}

// TestFindDatabasePath_CwdPriority verifies FindDatabasePath (the database
// discovery path) also prefers cwd's .issuegraph/ over the git worktree root's.
func TestFindDatabasePath_CwdPriority(t *testing.T) {
	origIssueGraphDir := os.Getenv("ISSUEGRAPH_DIR")
	origIssueGraphDB := os.Getenv("ISSUEGRAPH_DB")
	t.Cleanup(func() {
		if origIssueGraphDir != "" {
			os.Setenv("ISSUEGRAPH_DIR", origIssueGraphDir)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
		if origIssueGraphDB != "" {
			os.Setenv("ISSUEGRAPH_DB", origIssueGraphDB)
		} else {
			os.Unsetenv("ISSUEGRAPH_DB")
		}
	})
	os.Unsetenv("ISSUEGRAPH_DIR")
	os.Unsetenv("ISSUEGRAPH_DB")

	tmpDir := t.TempDir()

	// Create a git repo
	cmd := exec.Command("git", "init", tmpDir)
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Create root-level .issuegraph/ with a dolt dir (the "wrong" one)
	rootIssueGraphDir := filepath.Join(tmpDir, ".issuegraph")
	rootDoltDir := filepath.Join(rootIssueGraphDir, "dolt")
	if err := os.MkdirAll(rootDoltDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootIssueGraphDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"root_db"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create rig subdirectory with its own .issuegraph/ and dolt dir
	rigDir := filepath.Join(tmpDir, "my-rig")
	rigIssueGraphDir := filepath.Join(rigDir, ".issuegraph")
	rigDoltDir := filepath.Join(rigIssueGraphDir, "dolt")
	if err := os.MkdirAll(rigDoltDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rigIssueGraphDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"rig_db"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// cd into the rig directory
	t.Chdir(rigDir)

	result := FindDatabasePath()

	// The database path should be under the rig's .issuegraph/, not the root's
	if result == "" {
		t.Fatal("FindDatabasePath() returned empty, expected rig's database path")
	}

	resultResolved, _ := filepath.EvalSymlinks(result)
	rigDirResolved, _ := filepath.EvalSymlinks(rigDir)
	rootIssueGraphDirResolved, _ := filepath.EvalSymlinks(rootIssueGraphDir)
	if !isUnder(resultResolved, rigDirResolved) {
		t.Errorf("FindDatabasePath() = %q, want path under %q (rig's .issuegraph should win)", result, rigDirResolved)
	}
	if isUnder(resultResolved, rootIssueGraphDirResolved) {
		t.Errorf("FindDatabasePath() = %q, should NOT be under root's .issuegraph %q", result, rootIssueGraphDir)
	}
}

// TestFindIssueGraphDir_CwdWithoutIssueGraph_FallsBackToWalk verifies that when cwd
// has no .issuegraph/, the normal walk-up behavior still works.
func TestFindIssueGraphDir_CwdWithoutIssueGraph_FallsBackToWalk(t *testing.T) {
	origIssueGraphDir := os.Getenv("ISSUEGRAPH_DIR")
	t.Cleanup(func() {
		if origIssueGraphDir != "" {
			os.Setenv("ISSUEGRAPH_DIR", origIssueGraphDir)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	})
	os.Unsetenv("ISSUEGRAPH_DIR")

	tmpDir := t.TempDir()

	// Create a git repo
	cmd := exec.Command("git", "init", tmpDir)
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Create root-level .issuegraph/ only (no rig-level .issuegraph/)
	rootIssueGraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(rootIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootIssueGraphDir, "metadata.json"), []byte(`{"backend":"dolt"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a subdirectory WITHOUT .issuegraph/
	subDir := filepath.Join(tmpDir, "some", "deep", "subdir")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Chdir(subDir)

	result := FindIssueGraphDir()

	resultResolved, _ := filepath.EvalSymlinks(result)
	expectedResolved, _ := filepath.EvalSymlinks(rootIssueGraphDir)
	if resultResolved != expectedResolved {
		t.Errorf("FindIssueGraphDir() = %q, want %q (should fall back to root when cwd has no .issuegraph/)", result, rootIssueGraphDir)
	}
}

// TestFindIssueGraphDir_CwdIssueGraphDirWithRedirect verifies that cwd's .issuegraph/
// redirect is followed when the cwd check fires.
func TestFindIssueGraphDir_CwdIssueGraphDirWithRedirect(t *testing.T) {
	origIssueGraphDir := os.Getenv("ISSUEGRAPH_DIR")
	t.Cleanup(func() {
		if origIssueGraphDir != "" {
			os.Setenv("ISSUEGRAPH_DIR", origIssueGraphDir)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	})
	os.Unsetenv("ISSUEGRAPH_DIR")

	tmpDir := t.TempDir()

	// Create a git repo
	cmd := exec.Command("git", "init", tmpDir)
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Create root-level .issuegraph/
	rootIssueGraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(rootIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootIssueGraphDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"root_db"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a redirect target
	targetIssueGraphDir := filepath.Join(tmpDir, "shared-issuegraph")
	if err := os.MkdirAll(targetIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetIssueGraphDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"shared_db"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create rig subdirectory with .issuegraph/ that has a redirect
	rigDir := filepath.Join(tmpDir, "my-rig")
	rigIssueGraphDir := filepath.Join(rigDir, ".issuegraph")
	if err := os.MkdirAll(rigIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Write redirect file pointing to the shared target
	if err := os.WriteFile(filepath.Join(rigIssueGraphDir, "redirect"), []byte(targetIssueGraphDir), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(rigDir)

	result := FindIssueGraphDir()

	resultResolved, _ := filepath.EvalSymlinks(result)
	expectedResolved, _ := filepath.EvalSymlinks(targetIssueGraphDir)
	if resultResolved != expectedResolved {
		t.Errorf("FindIssueGraphDir() = %q, want %q (cwd .issuegraph/ redirect should be followed)", result, targetIssueGraphDir)
	}
}

// TestFindIssueGraphDir_ISSUEGRAPH_DIR_StillTakesPriority verifies that ISSUEGRAPH_DIR env
// var still takes priority over the cwd check.
func TestFindIssueGraphDir_ISSUEGRAPH_DIR_StillTakesPriority(t *testing.T) {
	origIssueGraphDir := os.Getenv("ISSUEGRAPH_DIR")
	t.Cleanup(func() {
		if origIssueGraphDir != "" {
			os.Setenv("ISSUEGRAPH_DIR", origIssueGraphDir)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	})

	tmpDir := t.TempDir()

	// Create an explicit ISSUEGRAPH_DIR target
	explicitIssueGraphDir := filepath.Join(tmpDir, "explicit-issuegraph")
	if err := os.MkdirAll(explicitIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(explicitIssueGraphDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"explicit_db"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	os.Setenv("ISSUEGRAPH_DIR", explicitIssueGraphDir)

	// Create cwd with its own .issuegraph/
	cwdDir := filepath.Join(tmpDir, "cwd-project")
	cwdIssueGraphDir := filepath.Join(cwdDir, ".issuegraph")
	if err := os.MkdirAll(cwdIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cwdIssueGraphDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"cwd_db"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(cwdDir)

	result := FindIssueGraphDir()

	resultResolved, _ := filepath.EvalSymlinks(result)
	expectedResolved, _ := filepath.EvalSymlinks(explicitIssueGraphDir)
	if resultResolved != expectedResolved {
		t.Errorf("FindIssueGraphDir() = %q, want %q (ISSUEGRAPH_DIR should still take priority over cwd)", result, explicitIssueGraphDir)
	}
}

// TestFindIssueGraphDir_CwdEmptyIssueGraphDir_SkipsToCwdWalk verifies that when cwd
// has a .issuegraph/ directory without any project files, it's skipped and the
// normal walk-up behavior continues.
func TestFindIssueGraphDir_CwdEmptyIssueGraphDir_SkipsToCwdWalk(t *testing.T) {
	origIssueGraphDir := os.Getenv("ISSUEGRAPH_DIR")
	t.Cleanup(func() {
		if origIssueGraphDir != "" {
			os.Setenv("ISSUEGRAPH_DIR", origIssueGraphDir)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	})
	os.Unsetenv("ISSUEGRAPH_DIR")

	tmpDir := t.TempDir()

	// Create a git repo
	cmd := exec.Command("git", "init", tmpDir)
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Create root-level .issuegraph/ with project files
	rootIssueGraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(rootIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootIssueGraphDir, "metadata.json"), []byte(`{"backend":"dolt"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create rig with empty .issuegraph/ (no project files)
	rigDir := filepath.Join(tmpDir, "empty-rig")
	rigIssueGraphDir := filepath.Join(rigDir, ".issuegraph")
	if err := os.MkdirAll(rigIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// No metadata.json, no config.yaml, no dolt/ — empty dir

	t.Chdir(rigDir)

	result := FindIssueGraphDir()

	// Should fall through to the root's .issuegraph/ since rig's is empty
	resultResolved, _ := filepath.EvalSymlinks(result)
	expectedResolved, _ := filepath.EvalSymlinks(rootIssueGraphDir)
	if resultResolved != expectedResolved {
		t.Errorf("FindIssueGraphDir() = %q, want %q (empty cwd .issuegraph/ should be skipped)", result, rootIssueGraphDir)
	}
}

// TestFindIssueGraphDir_RigSubdir_WalksUpToRigIssueGraph verifies that when CWD is a
// subdirectory within a rig, the walk-up finds the rig's .issuegraph/ (not the
// git root's .issuegraph/). This is the primary fix for GH#3027 — the old step 1b
// only checked CWD directly, missing rig subdirectories.
func TestFindIssueGraphDir_RigSubdir_WalksUpToRigIssueGraph(t *testing.T) {
	origIssueGraphDir := os.Getenv("ISSUEGRAPH_DIR")
	t.Cleanup(func() {
		if origIssueGraphDir != "" {
			os.Setenv("ISSUEGRAPH_DIR", origIssueGraphDir)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	})
	os.Unsetenv("ISSUEGRAPH_DIR")

	tmpDir := t.TempDir()

	// Create a git repo (simulating the main repo root)
	cmd := exec.Command("git", "init", tmpDir)
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Create root-level .issuegraph/ with project files (the "wrong" one)
	rootIssueGraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(rootIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootIssueGraphDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"root_db"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootIssueGraphDir, "config.yaml"), []byte("issue_prefix: root\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a rig subdirectory with its own .issuegraph/ (the "right" one)
	rigDir := filepath.Join(tmpDir, "my-rig")
	rigIssueGraphDir := filepath.Join(rigDir, ".issuegraph")
	if err := os.MkdirAll(rigIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rigIssueGraphDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"rig_db"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rigIssueGraphDir, "config.yaml"), []byte("issue_prefix: rig\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a deep subdirectory within the rig (CWD will be here)
	deepDir := filepath.Join(rigDir, "polecats", "furiosa", "src")
	if err := os.MkdirAll(deepDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// cd into the deep subdirectory within the rig
	t.Chdir(deepDir)

	result := FindIssueGraphDir()

	resultResolved, _ := filepath.EvalSymlinks(result)
	expectedResolved, _ := filepath.EvalSymlinks(rigIssueGraphDir)
	if resultResolved != expectedResolved {
		t.Errorf("FindIssueGraphDir() = %q, want %q (should walk up from rig subdir to find rig's .issuegraph/)", result, rigIssueGraphDir)
	}
}

// TestFindIssueGraphDir_DeepSubdir_NoRig_WalksToGitRoot verifies that when CWD is
// a deep subdirectory with no rig .issuegraph/ in between, the walk-up finds the
// git root's .issuegraph/.
func TestFindIssueGraphDir_DeepSubdir_NoRig_WalksToGitRoot(t *testing.T) {
	origIssueGraphDir := os.Getenv("ISSUEGRAPH_DIR")
	t.Cleanup(func() {
		if origIssueGraphDir != "" {
			os.Setenv("ISSUEGRAPH_DIR", origIssueGraphDir)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	})
	os.Unsetenv("ISSUEGRAPH_DIR")

	tmpDir := t.TempDir()

	// Create a git repo
	cmd := exec.Command("git", "init", tmpDir)
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Create root-level .issuegraph/ only
	rootIssueGraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(rootIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootIssueGraphDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"root_db"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a deep subdirectory without any .issuegraph/ in between
	deepDir := filepath.Join(tmpDir, "a", "b", "c", "d")
	if err := os.MkdirAll(deepDir, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Chdir(deepDir)

	result := FindIssueGraphDir()

	resultResolved, _ := filepath.EvalSymlinks(result)
	expectedResolved, _ := filepath.EvalSymlinks(rootIssueGraphDir)
	if resultResolved != expectedResolved {
		t.Errorf("FindIssueGraphDir() = %q, want %q (should walk up to git root's .issuegraph/)", result, rootIssueGraphDir)
	}
}

// TestFindIssueGraphDir_NestedRigs_FindsNearest verifies that with multiple nested
// rigs, the walk-up finds the nearest (innermost) rig's .issuegraph/.
func TestFindIssueGraphDir_NestedRigs_FindsNearest(t *testing.T) {
	origIssueGraphDir := os.Getenv("ISSUEGRAPH_DIR")
	t.Cleanup(func() {
		if origIssueGraphDir != "" {
			os.Setenv("ISSUEGRAPH_DIR", origIssueGraphDir)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	})
	os.Unsetenv("ISSUEGRAPH_DIR")

	tmpDir := t.TempDir()

	// Create a git repo
	cmd := exec.Command("git", "init", tmpDir)
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Root .issuegraph/
	rootIssueGraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(rootIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootIssueGraphDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"root_db"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Outer rig .issuegraph/
	outerRigDir := filepath.Join(tmpDir, "outer-rig")
	outerIssueGraphDir := filepath.Join(outerRigDir, ".issuegraph")
	if err := os.MkdirAll(outerIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outerIssueGraphDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"outer_db"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Inner rig .issuegraph/ (nested inside outer)
	innerRigDir := filepath.Join(outerRigDir, "inner-rig")
	innerIssueGraphDir := filepath.Join(innerRigDir, ".issuegraph")
	if err := os.MkdirAll(innerIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(innerIssueGraphDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"inner_db"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// CWD is a subdirectory of inner rig
	deepDir := filepath.Join(innerRigDir, "src", "pkg")
	if err := os.MkdirAll(deepDir, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Chdir(deepDir)

	result := FindIssueGraphDir()

	resultResolved, _ := filepath.EvalSymlinks(result)
	expectedResolved, _ := filepath.EvalSymlinks(innerIssueGraphDir)
	if resultResolved != expectedResolved {
		t.Errorf("FindIssueGraphDir() = %q, want %q (should find nearest/innermost rig's .issuegraph/)", result, innerIssueGraphDir)
	}
}

// TestFindIssueGraphDir_RigSubdirWithRedirect verifies that when CWD is a subdirectory
// within a rig and the rig's .issuegraph/ has a redirect, the redirect is followed.
func TestFindIssueGraphDir_RigSubdirWithRedirect(t *testing.T) {
	origIssueGraphDir := os.Getenv("ISSUEGRAPH_DIR")
	t.Cleanup(func() {
		if origIssueGraphDir != "" {
			os.Setenv("ISSUEGRAPH_DIR", origIssueGraphDir)
		} else {
			os.Unsetenv("ISSUEGRAPH_DIR")
		}
	})
	os.Unsetenv("ISSUEGRAPH_DIR")

	tmpDir := t.TempDir()

	// Create a git repo
	cmd := exec.Command("git", "init", tmpDir)
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Create root-level .issuegraph/
	rootIssueGraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(rootIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootIssueGraphDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"root_db"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create redirect target
	targetIssueGraphDir := filepath.Join(tmpDir, "shared-issuegraph")
	if err := os.MkdirAll(targetIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetIssueGraphDir, "metadata.json"), []byte(`{"backend":"dolt","dolt_database":"shared_db"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create rig with .issuegraph/ that has a redirect
	rigDir := filepath.Join(tmpDir, "my-rig")
	rigIssueGraphDir := filepath.Join(rigDir, ".issuegraph")
	if err := os.MkdirAll(rigIssueGraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rigIssueGraphDir, "redirect"), []byte(targetIssueGraphDir), 0o644); err != nil {
		t.Fatal(err)
	}

	// CWD is a subdirectory within the rig
	subDir := filepath.Join(rigDir, "polecats", "nux")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Chdir(subDir)

	result := FindIssueGraphDir()

	resultResolved, _ := filepath.EvalSymlinks(result)
	expectedResolved, _ := filepath.EvalSymlinks(targetIssueGraphDir)
	if resultResolved != expectedResolved {
		t.Errorf("FindIssueGraphDir() = %q, want %q (should follow redirect from rig .issuegraph/ found via walk-up)", result, targetIssueGraphDir)
	}
}

// isUnder returns true if child is under parent in the directory tree.
func isUnder(child, parent string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	// rel should not start with ".." (going up) and should not be absolute
	return !filepath.IsAbs(rel) && (rel == "." || (len(rel) >= 2 && rel[:2] != ".."))
}

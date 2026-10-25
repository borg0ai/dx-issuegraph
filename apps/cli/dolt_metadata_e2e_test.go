//go:build cgo && integration

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestE2E_InitDoltMetadataRoundtrip verifies that issuegraph init --backend dolt writes
// metadata that issuegraph doctor can validate without warnings.
// Covers FR-018 (e2e init->doctor roundtrip).
func TestE2E_InitDoltMetadataRoundtrip(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("dolt metadata e2e test not supported on windows")
	}

	tmpDir := createTempDirWithCleanup(t)

	// Set up a real git repo so repo_id can be computed
	if err := runCommandInDir(tmpDir, "git", "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	_ = runCommandInDir(tmpDir, "git", "config", "user.email", "test@example.com")
	_ = runCommandInDir(tmpDir, "git", "config", "user.name", "Test User")
	_ = runCommandInDir(tmpDir, "git", "config", "remote.origin.url", "https://github.com/test/repo.git")

	env := append(os.Environ(),
		"ISSUEGRAPH_TEST_MODE=1",
	)

	// Init dolt backend
	initOut, initErr := runIssueGraphExecAllowErrorWithEnv(t, tmpDir, env, "init", "--backend", "dolt", "--prefix", "test", "--quiet")
	if initErr != nil {
		lower := strings.ToLower(initOut)
		if strings.Contains(lower, "dolt") && (strings.Contains(lower, "not supported") || strings.Contains(lower, "not available") || strings.Contains(lower, "unknown")) {
			t.Skipf("dolt backend not available: %s", initOut)
		}
		t.Fatalf("issuegraph init --backend dolt failed: %v\n%s", initErr, initOut)
	}

	// Run doctor and verify no metadata warnings
	doctorOut, _ := runIssueGraphExecAllowErrorWithEnv(t, tmpDir, env, "doctor")

	// Doctor should NOT report missing metadata
	metadataWarnings := []string{
		"Missing metadata",
		"issuegraph_version",
		"repo_id not set",
		"clone_id not set",
	}
	for _, warning := range metadataWarnings {
		if strings.Contains(doctorOut, warning) {
			t.Errorf("issuegraph doctor reported metadata warning %q after init; output:\n%s", warning, doctorOut)
		}
	}

	// Sanity check: doctor should mention dolt
	if !strings.Contains(strings.ToLower(doctorOut), "dolt") {
		t.Logf("Note: doctor output did not mention dolt; output:\n%s", doctorOut)
	}

	// Verify no SQLite database was created (regression check)
	if _, err := os.Stat(filepath.Join(tmpDir, ".issuegraph", "issuegraph.db")); err == nil {
		t.Errorf("unexpected sqlite database created in dolt mode")
	}
}

// TestE2E_DoctorFixMetadataRoundtrip verifies that issuegraph doctor --fix repairs
// missing metadata on an existing Dolt database, and a subsequent issuegraph doctor
// reports a clean state.
// Covers FR-019 (e2e doctor fix cycle).
func TestE2E_DoctorFixMetadataRoundtrip(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("dolt metadata e2e test not supported on windows")
	}
	if testDoltServerPort == 0 {
		t.Skip("skipping: Dolt test container not available")
	}

	tmpDir := createTempDirWithCleanup(t)

	// Set up a real git repo so repo_id can be computed
	if err := runCommandInDir(tmpDir, "git", "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	_ = runCommandInDir(tmpDir, "git", "config", "user.email", "test@example.com")
	_ = runCommandInDir(tmpDir, "git", "config", "user.name", "Test User")
	_ = runCommandInDir(tmpDir, "git", "config", "remote.origin.url", "https://github.com/test/repo.git")

	env := append(os.Environ(),
		"ISSUEGRAPH_TEST_MODE=1",
	)

	// Init dolt backend (which now writes metadata via Phase 1)
	initOut, initErr := runIssueGraphExecAllowErrorWithEnv(t, tmpDir, env, "init", "--backend", "dolt", "--prefix", "test", "--quiet")
	if initErr != nil {
		lower := strings.ToLower(initOut)
		if strings.Contains(lower, "dolt") && (strings.Contains(lower, "not supported") || strings.Contains(lower, "not available") || strings.Contains(lower, "unknown")) {
			t.Skipf("dolt backend not available: %s", initOut)
		}
		t.Fatalf("issuegraph init --backend dolt failed: %v\n%s", initErr, initOut)
	}

	// Delete metadata to simulate a pre-Phase-1 database
	// issuegraph_version is now in local_metadata (dolt-ignored), repo_id/clone_id remain in metadata
	sqlOut, sqlErr := runIssueGraphExecAllowErrorWithEnv(t, tmpDir, env, "sql",
		"DELETE FROM local_metadata WHERE `key` = 'issuegraph_version'; DELETE FROM metadata WHERE `key` IN ('repo_id', 'clone_id')")
	if sqlErr != nil {
		t.Fatalf("issuegraph sql DELETE failed: %v\n%s", sqlErr, sqlOut)
	}

	// Verify doctor detects the missing metadata
	doctorOut1, _ := runIssueGraphExecAllowErrorWithEnv(t, tmpDir, env, "doctor")
	if !strings.Contains(doctorOut1, "doctor --fix") {
		t.Logf("Note: first doctor did not suggest 'doctor --fix'; output:\n%s", doctorOut1)
	}

	// Run doctor --fix to repair
	fixOut, fixErr := runIssueGraphExecAllowErrorWithEnv(t, tmpDir, env, "doctor", "--fix", "--yes")
	if fixErr != nil {
		t.Fatalf("issuegraph doctor --fix failed: %v\n%s", fixErr, fixOut)
	}

	// Run doctor again and verify no metadata warnings
	doctorOut2, _ := runIssueGraphExecAllowErrorWithEnv(t, tmpDir, env, "doctor")
	metadataWarnings := []string{
		"Missing metadata",
		"missing version metadata",
		"Missing repo fingerprint",
	}
	for _, warning := range metadataWarnings {
		if strings.Contains(doctorOut2, warning) {
			t.Errorf("issuegraph doctor still reports metadata warning %q after fix; output:\n%s", warning, doctorOut2)
		}
	}
}

// TestE2E_MigrateDoltMetadata verifies that issuegraph migrate on a Dolt database
// sets all 3 metadata fields (issuegraph_version, repo_id, clone_id), including
// the case where issuegraph_version is already set but identity fields are missing.
// Covers SC-005 (migrate sets repo_id and clone_id).
func TestE2E_MigrateDoltMetadata(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("dolt metadata e2e test not supported on windows")
	}
	if testDoltServerPort == 0 {
		t.Skip("skipping: Dolt test container not available")
	}

	tmpDir := createTempDirWithCleanup(t)

	// Set up a real git repo so repo_id can be computed
	if err := runCommandInDir(tmpDir, "git", "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	_ = runCommandInDir(tmpDir, "git", "config", "user.email", "test@example.com")
	_ = runCommandInDir(tmpDir, "git", "config", "user.name", "Test User")
	_ = runCommandInDir(tmpDir, "git", "config", "remote.origin.url", "https://github.com/test/repo.git")

	env := append(os.Environ(),
		"ISSUEGRAPH_TEST_MODE=1",
	)

	// Init dolt backend (writes all metadata via Phase 1)
	initOut, initErr := runIssueGraphExecAllowErrorWithEnv(t, tmpDir, env, "init", "--backend", "dolt", "--prefix", "test", "--quiet")
	if initErr != nil {
		lower := strings.ToLower(initOut)
		if strings.Contains(lower, "dolt") && (strings.Contains(lower, "not supported") || strings.Contains(lower, "not available") || strings.Contains(lower, "unknown")) {
			t.Skipf("dolt backend not available: %s", initOut)
		}
		t.Fatalf("issuegraph init --backend dolt failed: %v\n%s", initErr, initOut)
	}

	// Delete repo_id and clone_id to simulate a pre-Phase-3 database
	// (issuegraph_version is set by init, but identity fields are missing)
	sqlOut, sqlErr := runIssueGraphExecAllowErrorWithEnv(t, tmpDir, env, "sql",
		"DELETE FROM metadata WHERE key IN ('repo_id', 'clone_id')")
	if sqlErr != nil {
		t.Fatalf("issuegraph sql DELETE failed: %v\n%s", sqlErr, sqlOut)
	}

	// Run issuegraph migrate — should detect missing repo_id/clone_id and set them
	migrateOut, migrateErr := runIssueGraphExecAllowErrorWithEnv(t, tmpDir, env, "migrate")
	if migrateErr != nil {
		t.Fatalf("issuegraph migrate failed: %v\n%s", migrateErr, migrateOut)
	}

	// Verify migrate output mentions setting identity fields
	if !strings.Contains(migrateOut, "repo_id") && !strings.Contains(migrateOut, "All metadata") {
		t.Logf("Note: migrate output did not mention repo_id; output:\n%s", migrateOut)
	}

	// Run doctor to verify clean state after migrate
	doctorOut, _ := runIssueGraphExecAllowErrorWithEnv(t, tmpDir, env, "doctor")

	// Doctor should NOT report missing metadata
	metadataWarnings := []string{
		"Missing metadata",
		"repo_id not set",
		"Missing repo fingerprint",
	}
	for _, warning := range metadataWarnings {
		if strings.Contains(doctorOut, warning) {
			t.Errorf("issuegraph doctor reported metadata warning %q after migrate; output:\n%s", warning, doctorOut)
		}
	}
}

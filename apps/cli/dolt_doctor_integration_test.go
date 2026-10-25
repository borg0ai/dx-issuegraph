//go:build cgo && integration

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDoltDoctor_NoSQLiteWarningsAfterInitAndCreate(t *testing.T) {
	if runtime.GOOS == windowsOS {
		t.Skip("dolt doctor integration test not supported on windows")
	}

	tmpDir := createTempDirWithCleanup(t)

	// Set up a real git repo so init/create/doctor behave normally.
	if err := runCommandInDir(tmpDir, "git", "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	_ = runCommandInDir(tmpDir, "git", "config", "user.email", "test@example.com")
	_ = runCommandInDir(tmpDir, "git", "config", "user.name", "Test User")

	if testDoltServerPort == 0 {
		t.Skip("skipping: Dolt test container not available")
	}

	env := append(os.Environ(), "ISSUEGRAPH_TEST_MODE=1")

	// Init dolt backend.
	initOut, initErr := runIssueGraphExecAllowErrorWithEnv(t, tmpDir, env, "init", "--backend", "dolt", "--prefix", "test", "--quiet")
	if initErr != nil {
		// If dolt backend isn't available in this build, skip rather than fail.
		lower := strings.ToLower(initOut)
		if strings.Contains(lower, "dolt") && (strings.Contains(lower, "not supported") || strings.Contains(lower, "not available") || strings.Contains(lower, "unknown")) {
			t.Skipf("dolt backend not available: %s", initOut)
		}
		t.Fatalf("issuegraph init --backend dolt failed: %v\n%s", initErr, initOut)
	}

	// Create one issue so the store is definitely initialized.
	createOut, createErr := runIssueGraphExecAllowErrorWithEnv(t, tmpDir, env, "create", "doctor dolt smoke", "--json")
	if createErr != nil {
		t.Fatalf("issuegraph create failed: %v\n%s", createErr, createOut)
	}

	// Run doctor; it may return non-zero for unrelated warnings (upstream, claude, etc),
	// but it should NOT include SQLite-only failures on dolt.
	doctorOut, _ := runIssueGraphExecAllowErrorWithEnv(t, tmpDir, env, "doctor")

	// Also include stderr-like output if doctor wrote it to stdout in some modes.
	// (CombinedOutput already captures both.)
	for _, forbidden := range []string{
		"No issuegraph.db found",
		"Unable to read database version",
		"Legacy database",
	} {
		if strings.Contains(doctorOut, forbidden) {
			t.Fatalf("issuegraph doctor printed sqlite-specific warning %q in dolt mode; output:\n%s", forbidden, doctorOut)
		}
	}

	// Sanity check: doctor should mention dolt somewhere so we know we exercised the right path.
	if !strings.Contains(strings.ToLower(doctorOut), "dolt") {
		// Some doctor output is terse depending on flags; don't be too strict, but
		// if it's completely missing, that usually means we didn't use dolt config.
		t.Fatalf("issuegraph doctor output did not mention dolt; output:\n%s", doctorOut)
	}

	// Regression check: dolt init must NOT create a SQLite database file.
	if _, err := os.Stat(filepath.Join(tmpDir, ".issuegraph", "issuegraph.db")); err == nil {
		t.Fatalf("unexpected sqlite database created in dolt mode: %s", filepath.Join(tmpDir, ".issuegraph", "issuegraph.db"))
	}
}

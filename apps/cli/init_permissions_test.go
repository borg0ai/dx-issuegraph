//go:build !windows

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var (
	initPermissionTestBD             string
	initPermissionTestIssueGraphOnce sync.Once
	initPermissionTestIssueGraphErr  error
)

func buildIssueGraphForInitPermissionTests(t *testing.T) string {
	t.Helper()
	initPermissionTestIssueGraphOnce.Do(func() {
		prebuilt, err := findPrebuiltIssueGraphBinary()
		if err != nil {
			initPermissionTestIssueGraphErr = err
			return
		}
		if prebuilt != "" {
			initPermissionTestBD = prebuilt
			return
		}
		tmpDir, err := testTempDir("issuegraph-init-permissions-test-*")
		if err != nil {
			initPermissionTestIssueGraphErr = fmt.Errorf("failed to create temp dir: %w", err)
			return
		}
		initPermissionTestBD = filepath.Join(tmpDir, "issuegraph")
		cmd := exec.Command("go", "build", "-tags", "gms_pure_go", "-o", initPermissionTestBD, ".")
		if out, err := cmd.CombinedOutput(); err != nil {
			initPermissionTestIssueGraphErr = fmt.Errorf("go build failed: %v\n%s", err, out)
		}
	})
	if initPermissionTestIssueGraphErr != nil {
		t.Fatalf("failed to build issuegraph binary: %v", initPermissionTestIssueGraphErr)
	}
	return initPermissionTestBD
}

// TestInitRepairsPermissiveIssueGraphDir is the init-path regression test for
// GH#3391: a pre-existing .issuegraph/ directory with permissive bits
// (e.g. 0755 from a permissive umask) must be repaired to 0700 during
// issuegraph init.
//
// The test creates a real git repo with a pre-existing .issuegraph/ at 0755,
// runs issuegraph init, and asserts that permissions are repaired. The init may
// fail later (e.g. no Dolt server), but the permission fix happens early
// enough that the assertion is valid regardless of exit code.
func TestInitRepairsPermissiveIssueGraphDir(t *testing.T) {
	issuegraphBin := buildIssueGraphForInitPermissionTests(t)

	repoDir := newGitRepo(t)

	issuegraphDir := filepath.Join(repoDir, ".issuegraph")
	if err := os.Mkdir(issuegraphDir, 0755); err != nil {
		t.Fatalf("failed to create .issuegraph: %v", err)
	}
	if err := os.Chmod(issuegraphDir, 0755); err != nil {
		t.Fatalf("failed to chmod .issuegraph: %v", err)
	}

	// Verify starting permissions.
	info, err := os.Stat(issuegraphDir)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got&0007 == 0 {
		t.Fatalf("precondition failed: .issuegraph should have world bits, got %04o", got)
	}

	cmd := exec.Command(issuegraphBin, "init", "--prefix", "issuegraph",
		"--non-interactive", "--skip-hooks", "--skip-agents")
	cmd.Dir = repoDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// We don't check the exit code — init may fail later for reasons
	// unrelated to permissions (no Dolt, etc.). The permission fix runs
	// before database creation.
	_ = cmd.Run()

	// Assert: permissions must be repaired to the same mode issuegraph uses when
	// creating .issuegraph/ itself.
	info, err = os.Stat(issuegraphDir)
	if err != nil {
		t.Fatalf("Stat(.issuegraph) after init: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0700 {
		t.Errorf(".issuegraph permissions after init = %04o, want 0700", perm)
	}

	// Assert: the fix was announced on stderr.
	if !strings.Contains(stderr.String(), "Fixed .issuegraph permissions to 0700") {
		t.Errorf("expected permission-fix message on stderr, got:\n%s", stderr.String())
	}
}

// TestInitPreservesSecureIssueGraphDir verifies that issuegraph init does NOT touch a
// .issuegraph/ directory that already has secure permissions (0700).
func TestInitPreservesSecureIssueGraphDir(t *testing.T) {
	issuegraphBin := buildIssueGraphForInitPermissionTests(t)

	repoDir := newGitRepo(t)

	issuegraphDir := filepath.Join(repoDir, ".issuegraph")
	if err := os.Mkdir(issuegraphDir, 0700); err != nil {
		t.Fatalf("failed to create .issuegraph: %v", err)
	}

	cmd := exec.Command(issuegraphBin, "init", "--prefix", "issuegraph",
		"--non-interactive", "--skip-hooks", "--skip-agents")
	cmd.Dir = repoDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	_ = cmd.Run()

	// Permissions should remain 0700.
	info, err := os.Stat(issuegraphDir)
	if err != nil {
		t.Fatalf("Stat(.issuegraph) after init: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0700 {
		t.Errorf(".issuegraph permissions after init = %04o, want 0700", perm)
	}

	// No fix message expected.
	if strings.Contains(stderr.String(), "Fixed .issuegraph permissions") {
		t.Errorf("unexpected permission-fix message for already-secure .issuegraph/:\n%s", stderr.String())
	}
}

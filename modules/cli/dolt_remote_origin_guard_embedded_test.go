//go:build embeddeddolt && cgo

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Integration tests for be-7eu1d: git-origin collision guard in issuegraph dolt remote
// add and issuegraph doctor (FR-06 to FR-09).
//
// TDD gate tests — these fail before the implementation exists. Run with
// ISSUEGRAPH_TEST_EMBEDDED_DOLT=1.

// remoteAdd runs "issuegraph dolt remote add <args>" returning (stdout, stderr, error)
// with separate buffers. Delegates to issuegraphDoltSeparate (dolt_local_only_embedded_test.go).
func remoteAdd(t *testing.T, issuegraph, dir string, args ...string) (string, string, error) {
	t.Helper()
	return issuegraphDoltSeparate(t, issuegraph, dir, append([]string{"remote", "add"}, args...)...)
}

// setGitOrigin adds a git remote named "origin" to the directory.
func setGitOrigin(t *testing.T, dir, url string) {
	t.Helper()
	cmd := exec.Command("git", "remote", "add", "origin", url)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote add origin %s: %v\n%s", url, err, out)
	}
}

// issuegraphDoctorSeparate runs "issuegraph doctor <args>" returning (stdout, stderr, error).
func issuegraphDoctorSeparate(t *testing.T, issuegraph, dir string, args ...string) (string, string, error) {
	t.Helper()
	fullArgs := append([]string{"doctor"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// -- issuegraph dolt remote add: collision with git origin --

// TestDoltRemoteAdd_GitOriginCollision_Exits1 asserts that adding a Dolt remote
// whose URL matches the git origin exits non-zero and prints the refusal message.
func TestDoltRemoteAdd_GitOriginCollision_Exits1(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded integration tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "rg")

	originURL := "https://github.com/org/repo.git"
	setGitOrigin(t, dir, originURL)

	_, stderr, err := remoteAdd(t, issuegraph, dir, "origin", originURL)
	if err == nil {
		t.Fatalf("expected issuegraph dolt remote add to exit non-zero on git-origin collision, but succeeded\nstderr: %s", stderr)
	}
	if !strings.Contains(stderr, "refusing to add") {
		t.Errorf("expected 'refusing to add' in stderr\ngot:\n%s", stderr)
	}
	if !strings.Contains(stderr, originURL) {
		t.Errorf("expected rejected URL %q in stderr\ngot:\n%s", originURL, stderr)
	}
}

// TestDoltRemoteAdd_GitOriginCollision_RefusalMessageCopy verifies exact phrases
// from the UX spec (be-fxwbm §2.1) appear in the refusal output.
func TestDoltRemoteAdd_GitOriginCollision_RefusalMessageCopy(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded integration tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "rm")

	originURL := "https://github.com/org/repo.git"
	setGitOrigin(t, dir, originURL)

	_, stderr, _ := remoteAdd(t, issuegraph, dir, "origin", originURL)

	wantPhrases := []string{
		"refusing to add",    // §2.1 first line
		"git origin",         // §2.1 first line
		"--allow-git-origin", // §2.1 recovery option
		"dolt.local-only",    // §2.1 opt-out option
	}
	for _, phrase := range wantPhrases {
		if !strings.Contains(stderr, phrase) {
			t.Errorf("refusal message missing %q\nfull stderr:\n%s", phrase, stderr)
		}
	}
}

// TestDoltRemoteAdd_AllowGitOrigin_Exits0 asserts that --allow-git-origin
// bypasses the collision guard and exits 0.
func TestDoltRemoteAdd_AllowGitOrigin_Exits0(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded integration tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "ag")

	originURL := "file:///tmp/fake-git-remote"
	setGitOrigin(t, dir, originURL)

	_, _, err := remoteAdd(t, issuegraph, dir, "--allow-git-origin", "dolt-origin", originURL)
	if err != nil {
		t.Errorf("expected issuegraph dolt remote add --allow-git-origin to exit 0 on collision, got error: %v", err)
	}
}

// TestDoltRemoteAdd_AllowGitOrigin_WarningOnStderr asserts the one-liner warning
// from be-fxwbm §2.2 appears on stderr when --allow-git-origin is set.
func TestDoltRemoteAdd_AllowGitOrigin_WarningOnStderr(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded integration tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "aw")

	originURL := "file:///tmp/fake-git-remote2"
	setGitOrigin(t, dir, originURL)

	_, stderr, _ := remoteAdd(t, issuegraph, dir, "--allow-git-origin", "dolt-origin", originURL)

	// §2.2: "Warning: '<url>' matches the git origin — proceeding because --allow-git-origin is set."
	if !strings.Contains(stderr, "Warning:") {
		t.Errorf("expected 'Warning:' on stderr when --allow-git-origin is set\ngot:\n%s", stderr)
	}
	if !strings.Contains(stderr, "--allow-git-origin") {
		t.Errorf("expected '--allow-git-origin' in warning\ngot:\n%s", stderr)
	}
}

// TestDoltRemoteAdd_NoGitOrigin_ProceedsNormally asserts no refusal fires when
// the git repo has no 'origin' remote.
func TestDoltRemoteAdd_NoGitOrigin_ProceedsNormally(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded integration tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "no")
	// issuegraphInit creates a git repo, but no origin is set.

	remoteDir := t.TempDir()
	_, stderr, err := remoteAdd(t, issuegraph, dir, "dolt-origin", "file://"+remoteDir)
	if err != nil {
		t.Errorf("expected issuegraph dolt remote add to succeed when no git origin, got error: %v\nstderr: %s", err, stderr)
	}
	if strings.Contains(stderr, "refusing to add") {
		t.Errorf("unexpected refusal when no git origin\nstderr:\n%s", stderr)
	}
}

// TestDoltRemoteAdd_DifferentURL_ProceedsNormally asserts the guard does not
// fire when the Dolt remote URL is different from the git origin.
func TestDoltRemoteAdd_DifferentURL_ProceedsNormally(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded integration tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "du")

	setGitOrigin(t, dir, "https://github.com/org/repo.git")

	// Different URL — should not trigger guard.
	remoteDir := t.TempDir()
	_, stderr, err := remoteAdd(t, issuegraph, dir, "dolt-origin", "file://"+remoteDir)
	if err != nil {
		t.Errorf("expected issuegraph dolt remote add with different URL to succeed, got error: %v\nstderr: %s", err, stderr)
	}
	if strings.Contains(stderr, "refusing to add") {
		t.Errorf("unexpected refusal when URL doesn't match git origin\nstderr:\n%s", stderr)
	}
}

// TestDoltRemoteAdd_LocalOnly_FiresBeforeGuard verifies that when
// dolt.local-only=true is set, the local-only refusal fires before the
// git-origin guard — the local-only guard takes priority (be-3v2ou ordering).
func TestDoltRemoteAdd_LocalOnly_FiresBeforeGuard(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded integration tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "lo")
	issuegraphConfig(t, issuegraph, dir, "set", "dolt.local-only", "true")

	originURL := "https://github.com/org/repo.git"
	setGitOrigin(t, dir, originURL)

	_, stderr, err := remoteAdd(t, issuegraph, dir, "origin", originURL)
	if err == nil {
		t.Fatal("expected issuegraph dolt remote add to fail with dolt.local-only=true")
	}
	// Must mention local-only, not git-origin collision.
	if strings.Contains(stderr, "refusing to add") {
		t.Errorf("git-origin guard fired before local-only guard\nstderr:\n%s", stderr)
	}
	if !strings.Contains(stderr, "local-only") && !strings.Contains(stderr, "dolt.local-only") {
		t.Errorf("expected local-only message, got:\n%s", stderr)
	}
}

// -- issuegraph doctor: Dolt remote matches git origin --

// TestDoctor_DoltRemoteMatchesGitOrigin_Warning verifies that issuegraph doctor emits a
// warning when a Dolt remote URL matches the git origin.
func TestDoctor_DoltRemoteMatchesGitOrigin_Warning(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded integration tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "dw")

	originURL := "file:///tmp/origin-collision-for-doctor"
	setGitOrigin(t, dir, originURL)

	// Add a Dolt remote that matches the git origin (using --allow-git-origin to bypass the guard).
	issuegraphDolt(t, issuegraph, dir, "remote", "add", "--allow-git-origin", "origin", originURL)

	stdout, _, _ := issuegraphDoctorSeparate(t, issuegraph, dir)
	combined := stdout

	// Doctor must surface the warning check.
	if !strings.Contains(combined, "warning") && !strings.Contains(combined, "Warning") {
		t.Errorf("issuegraph doctor: expected warning when Dolt remote matches git origin\noutput:\n%s", combined)
	}
	if !strings.Contains(combined, "git origin") {
		t.Errorf("issuegraph doctor: expected 'git origin' in warning output\noutput:\n%s", combined)
	}
}

// TestDoctor_DoltRemoteNoMatch_OK verifies that issuegraph doctor does NOT emit a
// git-origin warning when the Dolt remote URL differs from the git origin.
func TestDoctor_DoltRemoteNoMatch_OK(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded integration tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "dn")

	setGitOrigin(t, dir, "https://github.com/org/repo.git")
	remoteDir := t.TempDir()
	issuegraphDolt(t, issuegraph, dir, "remote", "add", "dolt-origin", "file://"+remoteDir)

	stdout, _, _ := issuegraphDoctorSeparate(t, issuegraph, dir)
	if strings.Contains(stdout, "Dolt remote matches git origin") {
		t.Errorf("issuegraph doctor: unexpected git-origin warning when URLs differ\noutput:\n%s", stdout)
	}
}

// TestDoctor_NoGitOrigin_OK verifies that issuegraph doctor shows OK for the git-origin
// check when no git origin is configured.
func TestDoctor_NoGitOrigin_OK(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded integration tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "ng")
	// issuegraphInit creates a git repo but no origin.

	remoteDir := t.TempDir()
	issuegraphDolt(t, issuegraph, dir, "remote", "add", "dolt-origin", "file://"+remoteDir)

	stdout, _, _ := issuegraphDoctorSeparate(t, issuegraph, dir)
	// Should not emit a warning for the git-origin check.
	if strings.Contains(stdout, "Dolt remote matches git origin") {
		t.Errorf("issuegraph doctor: unexpected git-origin warning when no git origin\noutput:\n%s", stdout)
	}
}

// TestDoctor_JSON_DoltRemoteMatchesGitOrigin_Warning verifies that
// issuegraph doctor --json exposes the git-origin warning check with "status":"warning".
func TestDoctor_JSON_DoltRemoteMatchesGitOrigin_Warning(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded integration tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "dj")

	originURL := "file:///tmp/origin-collision-for-json"
	setGitOrigin(t, dir, originURL)
	issuegraphDolt(t, issuegraph, dir, "remote", "add", "--allow-git-origin", "origin", originURL)

	stdout, _, _ := issuegraphDoctorSeparate(t, issuegraph, dir, "--json")

	// Output should be JSON array of checks.
	var checks []map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &checks); err != nil {
		// May be wrapped in an outer object — try extracting the checks array.
		var outer map[string]interface{}
		if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &outer); jsonErr != nil {
			t.Fatalf("issuegraph doctor --json: not valid JSON: %v\noutput:\n%s", err, stdout)
		}
		if arr, ok := outer["checks"].([]interface{}); ok {
			data, _ := json.Marshal(arr)
			_ = json.Unmarshal(data, &checks)
		}
	}

	found := false
	for _, check := range checks {
		name, _ := check["name"].(string)
		if strings.Contains(name, "git origin") || strings.Contains(name, "Dolt remote") {
			found = true
			status, _ := check["status"].(string)
			if status != "warning" {
				t.Errorf("doctor JSON check %q: status = %q, want \"warning\"", name, status)
			}
			if _, hasFix := check["fix"]; !hasFix {
				t.Errorf("doctor JSON check %q: missing \"fix\" field", name)
			}
			break
		}
	}
	if !found {
		t.Errorf("issuegraph doctor --json: no git-origin check found in output\noutput:\n%s", stdout)
	}
}

// TestDoltRemoteAdd_JSON_AllowGitOrigin_WarningOnStderr asserts that even in
// --json mode, the override warning still appears on stderr (not stdout).
func TestDoltRemoteAdd_JSON_AllowGitOrigin_WarningOnStderr(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded integration tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "jw")

	originURL := "file:///tmp/origin-collision-json-mode"
	setGitOrigin(t, dir, originURL)

	// Run with --json and --allow-git-origin on collision — warning must go to stderr.
	cmd := exec.Command(issuegraph, "dolt", "--json", "remote", "add", "--allow-git-origin", "origin", originURL)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run()

	if !strings.Contains(stderr.String(), "Warning:") {
		t.Errorf("--json --allow-git-origin: expected Warning on stderr, got:\nstderr: %s\nstdout: %s",
			stderr.String(), stdout.String())
	}
}

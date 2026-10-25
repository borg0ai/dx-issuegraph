// Tests for multi-ID `issuegraph update` exit-code semantics (GH audit: multi-ID
// update exits 0 after a mid-batch failure, so callers cannot detect that
// some IDs were not updated).
//
// Contract under test:
//   - `issuegraph update` applies updates per issue ID, not atomically across IDs:
//     successful IDs stay applied even when other IDs fail.
//   - Any per-ID failure makes the command exit nonzero with a per-ID error
//     report on stderr.
//   - In --json mode stdout keeps the existing success shape (array of
//     updated issues) and the failure report is emitted as a single JSON
//     line on stderr listing which IDs failed.
//   - The all-good path is unchanged: exit 0, clean stderr.
//
// This file MUST NOT carry a cgo build tag: it exercises the default sqlite
// backend via a issuegraph binary built with the gms_pure_go tag.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// multiIDUpdateEnv returns a hermetic environment for issuegraph subprocess runs:
// no inherited ISSUEGRAPH_* variables, HOME and USERPROFILE pinned to the test dir,
// metrics and daemons disabled.
func multiIDUpdateEnv(dir string) []string {
	var env []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "ISSUEGRAPH_") {
			continue
		}
		env = append(env, e)
	}
	return append(env,
		"HOME="+dir,
		"USERPROFILE="+dir,
		"ISSUEGRAPH_NON_INTERACTIVE=1",
		"ISSUEGRAPH_DISABLE_METRICS=1",
		"ISSUEGRAPH_DISABLE_EVENT_FLUSH=1",
		"ISSUEGRAPH_NO_DAEMON=1",
		"ISSUEGRAPH_DOLT_AUTO_START=0",
	)
}

// runIssueGraphMultiID runs the issuegraph binary and returns stdout, stderr, and the exit
// code. Only a failure to launch the process fails the test; nonzero exits
// are returned to the caller for assertion.
func runIssueGraphMultiID(t *testing.T, issuegraph, dir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(issuegraph, args...)
	cmd.Dir = dir
	cmd.Env = multiIDUpdateEnv(dir)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			t.Fatalf("issuegraph %v did not run: %v", args, err)
		}
		return outBuf.String(), errBuf.String(), ee.ExitCode()
	}
	return outBuf.String(), errBuf.String(), 0
}

// setupMultiIDUpdateDB builds issuegraph and initializes a fresh sqlite-backed
// database in a temp dir.
func setupMultiIDUpdateDB(t *testing.T) (issuegraph, dir string) {
	t.Helper()
	issuegraph = buildIssueGraphForInitTests(t)
	dir = t.TempDir()
	runGitForBootstrapTest(t, dir, "init", "-q")
	runGitForBootstrapTest(t, dir, "config", "core.hooksPath", ".git/hooks")
	stdout, stderr, code := runIssueGraphMultiID(t, issuegraph, dir,
		"init", "--prefix", "test", "--quiet", "--non-interactive", "--skip-hooks", "--skip-agents")
	if code != 0 {
		t.Fatalf("issuegraph init failed (exit %d):\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	return issuegraph, dir
}

// createMultiIDUpdateIssue creates an issue and returns its ID.
func createMultiIDUpdateIssue(t *testing.T, issuegraph, dir, title string) string {
	t.Helper()
	stdout, stderr, code := runIssueGraphMultiID(t, issuegraph, dir, "create", title, "-p", "2", "--json")
	if code != 0 {
		t.Fatalf("issuegraph create failed (exit %d):\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	var issue struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(stdout), &issue); err != nil {
		t.Fatalf("parsing create --json output: %v\n%s", err, stdout)
	}
	if issue.ID == "" {
		t.Fatalf("issuegraph create --json returned no id:\n%s", stdout)
	}
	return issue.ID
}

// showMultiIDUpdatePriority fetches an issue's priority via issuegraph show --json.
func showMultiIDUpdatePriority(t *testing.T, issuegraph, dir, id string) int {
	t.Helper()
	stdout, stderr, code := runIssueGraphMultiID(t, issuegraph, dir, "show", id, "--json")
	if code != 0 {
		t.Fatalf("issuegraph show %s failed (exit %d):\nstdout:\n%s\nstderr:\n%s", id, code, stdout, stderr)
	}
	var details []struct {
		ID       string `json:"id"`
		Priority int    `json:"priority"`
	}
	if err := json.Unmarshal([]byte(stdout), &details); err != nil {
		t.Fatalf("parsing show --json output for %s: %v\n%s", id, err, stdout)
	}
	if len(details) != 1 || details[0].ID != id {
		t.Fatalf("show --json for %s returned unexpected issues:\n%s", id, stdout)
	}
	return details[0].Priority
}

// bogusMultiIDUpdateID is an ID that cannot collide with generated issue IDs
// (issuegraph resolves fuzzy/substring matches, so it must not be a substring of a
// real ID either).
const bogusMultiIDUpdateID = "test-zzzzzzzzzz"

func TestMultiIDUpdatePartialFailureExitsNonzero(t *testing.T) {
	issuegraph, dir := setupMultiIDUpdateDB(t)
	id1 := createMultiIDUpdateIssue(t, issuegraph, dir, "first issue")
	id2 := createMultiIDUpdateIssue(t, issuegraph, dir, "second issue")

	stdout, stderr, code := runIssueGraphMultiID(t, issuegraph, dir, "update", id1, bogusMultiIDUpdateID, id2, "--priority", "0")
	if code == 0 {
		t.Errorf("issuegraph update with a bogus ID mid-batch exited 0, want nonzero\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(stderr, bogusMultiIDUpdateID) {
		t.Errorf("stderr does not mention the failed ID %s:\n%s", bogusMultiIDUpdateID, stderr)
	}

	// The command is per-ID, not atomic: both good IDs must still be applied.
	for _, id := range []string{id1, id2} {
		if got := showMultiIDUpdatePriority(t, issuegraph, dir, id); got != 0 {
			t.Errorf("issue %s priority = %d, want 0 (successful IDs must stay applied)", id, got)
		}
	}
}

func TestMultiIDUpdatePartialFailureJSONReportsFailedIDs(t *testing.T) {
	issuegraph, dir := setupMultiIDUpdateDB(t)
	id1 := createMultiIDUpdateIssue(t, issuegraph, dir, "first issue")
	id2 := createMultiIDUpdateIssue(t, issuegraph, dir, "second issue")

	stdout, stderr, code := runIssueGraphMultiID(t, issuegraph, dir, "update", id1, bogusMultiIDUpdateID, id2, "--priority", "0", "--json")
	if code == 0 {
		t.Errorf("issuegraph update --json with a bogus ID mid-batch exited 0, want nonzero\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}

	// stdout keeps the existing success shape: an array of the successfully
	// updated issues.
	var updated []struct {
		ID       string `json:"id"`
		Priority int    `json:"priority"`
	}
	if err := json.Unmarshal([]byte(stdout), &updated); err != nil {
		t.Fatalf("stdout is not the array-of-issues success shape: %v\n%s", err, stdout)
	}
	gotIDs := map[string]bool{}
	for _, u := range updated {
		gotIDs[u.ID] = true
		if u.Priority != 0 {
			t.Errorf("updated issue %s priority = %d, want 0", u.ID, u.Priority)
		}
	}
	if !gotIDs[id1] || !gotIDs[id2] || len(updated) != 2 {
		t.Errorf("stdout array = %v, want exactly [%s %s]", updated, id1, id2)
	}

	// The last stderr line is a machine-parseable failure report naming the
	// failed ID.
	lines := strings.Split(strings.TrimSpace(stderr), "\n")
	last := lines[len(lines)-1]
	var report struct {
		Error  string `json:"error"`
		Failed []struct {
			ID    string `json:"id"`
			Error string `json:"error"`
		} `json:"failed"`
	}
	if err := json.Unmarshal([]byte(last), &report); err != nil {
		t.Fatalf("last stderr line is not a JSON failure report: %v\nstderr:\n%s", err, stderr)
	}
	if report.Error == "" {
		t.Errorf("JSON failure report has empty error message: %s", last)
	}
	if len(report.Failed) != 1 || report.Failed[0].ID != bogusMultiIDUpdateID {
		t.Errorf("JSON failure report failed list = %+v, want exactly one entry for %s", report.Failed, bogusMultiIDUpdateID)
	}
	if len(report.Failed) == 1 && report.Failed[0].Error == "" {
		t.Errorf("JSON failure report entry has empty per-ID error: %s", last)
	}
}

func TestMultiIDUpdateAllFailStillExitsNonzero(t *testing.T) {
	issuegraph, dir := setupMultiIDUpdateDB(t)

	stdout, stderr, code := runIssueGraphMultiID(t, issuegraph, dir, "update", bogusMultiIDUpdateID, "--priority", "0")
	if code == 0 {
		t.Errorf("issuegraph update with only a bogus ID exited 0, want nonzero\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(stderr, bogusMultiIDUpdateID) {
		t.Errorf("stderr does not mention the failed ID %s:\n%s", bogusMultiIDUpdateID, stderr)
	}
}

func TestMultiIDUpdateAllGoodPathUnchanged(t *testing.T) {
	issuegraph, dir := setupMultiIDUpdateDB(t)
	id1 := createMultiIDUpdateIssue(t, issuegraph, dir, "first issue")
	id2 := createMultiIDUpdateIssue(t, issuegraph, dir, "second issue")

	stdout, stderr, code := runIssueGraphMultiID(t, issuegraph, dir, "update", id1, id2, "--priority", "1", "--json")
	if code != 0 {
		t.Fatalf("all-good issuegraph update --json exited %d, want 0\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	if stderr != "" {
		t.Errorf("all-good issuegraph update --json wrote to stderr (success path must stay byte-identical):\n%s", stderr)
	}
	var updated []struct {
		ID       string `json:"id"`
		Priority int    `json:"priority"`
	}
	if err := json.Unmarshal([]byte(stdout), &updated); err != nil {
		t.Fatalf("stdout is not the array-of-issues success shape: %v\n%s", err, stdout)
	}
	if len(updated) != 2 {
		t.Fatalf("updated %d issues, want 2:\n%s", len(updated), stdout)
	}
	for _, u := range updated {
		if u.Priority != 1 {
			t.Errorf("updated issue %s priority = %d, want 1", u.ID, u.Priority)
		}
	}
}

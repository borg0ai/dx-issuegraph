//go:build cgo

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Integration tests for be-3v2ou: dolt.local-only enforcement (FR-01 to FR-05).
//
// TDD gate tests: these fail before the implementation because the guards
// don't exist yet.  Run with ISSUEGRAPH_TEST_EMBEDDED_DOLT=1.

// issuegraphLocalOnlySetup inits a project and enables dolt.local-only=true.
func issuegraphLocalOnlySetup(t *testing.T, issuegraph string) (dir string) {
	t.Helper()
	dir, _, _ = issuegraphInit(t, issuegraph, "--prefix", "lo")
	issuegraphConfig(t, issuegraph, dir, "set", "dolt.local-only", "true")
	return dir
}

// issuegraphDoltSeparate runs "issuegraph dolt <args>" and returns (stdout, stderr, err)
// with separate buffers.  Used when tests must distinguish stdout from stderr.
func issuegraphDoltSeparate(t *testing.T, issuegraph, dir string, args ...string) (string, string, error) {
	t.Helper()
	fullArgs := append([]string{"dolt"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// -- push --

func TestDoltLocalOnly_Push_ExitsZeroWithMessage(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run dolt.local-only embedded tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	dir := issuegraphLocalOnlySetup(t, issuegraph)

	// issuegraphDolt fatals on non-zero exit — this asserts exit 0.
	out := issuegraphDolt(t, issuegraph, dir, "push")

	for _, want := range []string{
		"Remote sync is disabled for this project (dolt.local-only=true).",
		"issuegraph config unset dolt.local-only",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("push stdout missing %q\ngot:\n%s", want, out)
		}
	}
}

func TestDoltLocalOnly_Push_LocalStoreCopy(t *testing.T) {
	// The push no-op message must include the "Your issues are stored locally" line.
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run dolt.local-only embedded tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	dir := issuegraphLocalOnlySetup(t, issuegraph)

	out := issuegraphDolt(t, issuegraph, dir, "push")
	if !strings.Contains(out, "stored locally") {
		t.Errorf("push no-op missing 'stored locally'\ngot:\n%s", out)
	}
}

func TestDoltLocalOnly_Push_JSON_DisabledStatus(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run dolt.local-only embedded tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	dir := issuegraphLocalOnlySetup(t, issuegraph)

	out := issuegraphDolt(t, issuegraph, dir, "--json", "push")

	var got map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &got); err != nil {
		t.Fatalf("issuegraph dolt --json push: stdout not valid JSON: %v\nout: %s", err, out)
	}
	if got["status"] != "disabled" {
		t.Errorf("JSON[status] = %q, want \"disabled\"", got["status"])
	}
	if got["reason"] != "dolt.local-only=true" {
		t.Errorf("JSON[reason] = %q, want \"dolt.local-only=true\"", got["reason"])
	}
}

// -- pull --

func TestDoltLocalOnly_Pull_ExitsZeroWithMessage(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run dolt.local-only embedded tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	dir := issuegraphLocalOnlySetup(t, issuegraph)

	out := issuegraphDolt(t, issuegraph, dir, "pull")

	for _, want := range []string{
		"Remote sync is disabled for this project (dolt.local-only=true).",
		"Nothing to pull",
		"issuegraph config unset dolt.local-only",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("pull stdout missing %q\ngot:\n%s", want, out)
		}
	}
}

func TestDoltLocalOnly_Pull_JSON_DisabledStatus(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run dolt.local-only embedded tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	dir := issuegraphLocalOnlySetup(t, issuegraph)

	out := issuegraphDolt(t, issuegraph, dir, "--json", "pull")

	var got map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &got); err != nil {
		t.Fatalf("issuegraph dolt --json pull: stdout not valid JSON: %v\nout: %s", err, out)
	}
	if got["status"] != "disabled" {
		t.Errorf("JSON[status] = %q, want \"disabled\"", got["status"])
	}
	if got["reason"] != "dolt.local-only=true" {
		t.Errorf("JSON[reason] = %q, want \"dolt.local-only=true\"", got["reason"])
	}
}

// -- remote add --

func TestDoltLocalOnly_RemoteAdd_RefusesExitOne(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run dolt.local-only embedded tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	dir := issuegraphLocalOnlySetup(t, issuegraph)

	stdout, stderr, err := issuegraphDoltSeparate(t, issuegraph, dir, "remote", "add", "origin", "https://doltremoteapi.dolthub.com/test/repo")
	if err == nil {
		t.Fatalf("issuegraph dolt remote add: expected exit non-zero, but succeeded\nstdout:\n%s", stdout)
	}
	for _, want := range []string{
		"cannot add Dolt remote",
		"dolt.local-only",
		"issuegraph config unset dolt.local-only",
	} {
		if !strings.Contains(strings.ToLower(stderr+stdout), strings.ToLower(want)) {
			t.Errorf("remote-add output missing %q\nstdout: %s\nstderr: %s", want, stdout, stderr)
		}
	}
}

func TestDoltLocalOnly_RemoteAdd_ErrorOnStderr(t *testing.T) {
	// The refusal must go to stderr (not stdout) so agents can distinguish errors.
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run dolt.local-only embedded tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	dir := issuegraphLocalOnlySetup(t, issuegraph)

	stdout, stderr, _ := issuegraphDoltSeparate(t, issuegraph, dir, "remote", "add", "origin", "https://doltremoteapi.dolthub.com/test/repo")
	if !strings.Contains(strings.ToLower(stderr), "disabled") {
		t.Errorf("remote-add refusal not on stderr\nstderr: %s", stderr)
	}
	if strings.Contains(strings.ToLower(stdout), "disabled") {
		t.Errorf("remote-add refusal leaked onto stdout\nstdout: %s", stdout)
	}
}

// -- config round-trip --

func TestDoltLocalOnly_ConfigRoundTrip(t *testing.T) {
	// issuegraph config set dolt.local-only true / unset must persist via config.yaml.
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run dolt.local-only embedded tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "cr")

	// Set: push becomes a no-op.
	issuegraphConfig(t, issuegraph, dir, "set", "dolt.local-only", "true")
	out := issuegraphDolt(t, issuegraph, dir, "push")
	if !strings.Contains(out, "dolt.local-only=true") {
		t.Errorf("after set: push output = %q, want to mention dolt.local-only=true", out)
	}

	// Unset: key must be absent from config. issuegraph config get returns "(not set ...)" for missing keys.
	issuegraphConfig(t, issuegraph, dir, "unset", "dolt.local-only")
	val := issuegraphConfig(t, issuegraph, dir, "get", "dolt.local-only")
	if !strings.Contains(val, "not set") {
		t.Errorf("after unset: issuegraph config get dolt.local-only = %q, want 'not set'", val)
	}
}

// -- issuegraph dolt status --

func TestDoltLocalOnly_DoltStatus_ShowsDisabled(t *testing.T) {
	// When dolt.local-only=true, issuegraph dolt status must include the indicator line.
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run dolt.local-only embedded tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	dir := issuegraphLocalOnlySetup(t, issuegraph)

	out := issuegraphDolt(t, issuegraph, dir, "status")
	want := "Remote sync: disabled (dolt.local-only=true)"
	if !strings.Contains(out, want) {
		t.Errorf("dolt status missing %q\ngot:\n%s", want, out)
	}
}

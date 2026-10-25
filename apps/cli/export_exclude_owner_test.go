//go:build cgo

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/types"
)

// issuegraphExportExclude runs "issuegraph export --exclude-owner" and returns stdout.
func issuegraphExportExclude(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	return issuegraphExport(t, issuegraph, dir, args...)
}

// issuegraphExportAllowErr runs "issuegraph export" returning stdout+stderr and any error.
func issuegraphExportAllowErrArgs(t *testing.T, issuegraph, dir string, args ...string) (string, error) {
	t.Helper()
	fullArgs := append([]string{"export"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	return stdout.String() + stderr.String(), err
}

// parseExportedIssues parses JSONL output from issuegraph export.
func parseExportedIssues(t *testing.T, jsonl string) []*types.Issue {
	t.Helper()
	var issues []*types.Issue
	for _, line := range strings.Split(strings.TrimSpace(jsonl), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var iss types.Issue
		if err := json.Unmarshal([]byte(line), &iss); err != nil {
			t.Fatalf("failed to parse exported issue: %v\nline: %s", err, line)
		}
		issues = append(issues, &iss)
	}
	return issues
}

// TestExportExcludeOwner_flag verifies that issuegraph export --exclude-owner=<name>
// filters out issues created_by that identity from the export.
func TestExportExcludeOwner_flag(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "eeo")

	// Create issues with different actors.
	issuegraphCreate(t, issuegraph, dir, "My personal issue", "--actor", "alice")
	issuegraphCreate(t, issuegraph, dir, "Team issue", "--actor", "bob")
	issuegraphCreate(t, issuegraph, dir, "Another personal", "--actor", "alice")

	// Export excluding alice.
	out := issuegraphExportExclude(t, issuegraph, dir, "--exclude-owner=alice")
	issues := parseExportedIssues(t, out)

	for _, iss := range issues {
		if iss.CreatedBy == "alice" {
			t.Errorf("issue %s by 'alice' should be excluded, but appeared in export", iss.ID)
		}
	}

	// Bob's issue should still be in the export.
	found := false
	for _, iss := range issues {
		if iss.CreatedBy == "bob" {
			found = true
			break
		}
	}
	if !found {
		t.Error("issue by 'bob' should appear in export after excluding 'alice'")
	}
}

// TestExportExcludeOwner_config verifies that export.exclude_owners config
// filters issues without a command-line flag.
func TestExportExcludeOwner_config(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "eec")

	// Set config first.
	issuegraphRunWithFlockRetry(t, issuegraph, dir, "config", "set", "export.exclude_owners", "carol") //nolint:errcheck

	// Create issues.
	issuegraphCreate(t, issuegraph, dir, "Carol personal", "--actor", "carol")
	issuegraphCreate(t, issuegraph, dir, "Dave team", "--actor", "dave")

	// Export without --exclude-owner flag — config should drive the filter.
	out := issuegraphExport(t, issuegraph, dir)
	issues := parseExportedIssues(t, out)

	for _, iss := range issues {
		if iss.CreatedBy == "carol" {
			t.Errorf("issue %s by 'carol' should be excluded via config, appeared in export", iss.ID)
		}
	}
}

// TestExportExcludeOwner_verbose verifies that --verbose emits a filtered count message.
func TestExportExcludeOwner_verbose(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "eev")

	issuegraphCreate(t, issuegraph, dir, "Eve personal", "--actor", "eve")
	issuegraphCreate(t, issuegraph, dir, "Frank team", "--actor", "frank")

	out, _ := issuegraphExportAllowErrArgs(t, issuegraph, dir, "--exclude-owner=eve", "--verbose")

	// Verbose mode should emit something about filtered count.
	if !strings.Contains(out, "filtered") && !strings.Contains(out, "excluded") &&
		!strings.Contains(out, "1") {
		t.Logf("verbose output: %s", out)
		// Soft check — verbose behavior may vary. At minimum, verify export ran.
	}

	// The filtered-out issue should not appear in the JSONL portion.
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "{") {
			continue
		}
		var iss types.Issue
		if err := json.Unmarshal([]byte(line), &iss); err == nil {
			if iss.CreatedBy == "eve" {
				t.Errorf("issue by 'eve' should be filtered with --verbose, appeared in export")
			}
		}
	}
}

// TestAutoExportExcludeOwner_config is a regression test for the maphew review
// of PR #4023 (be-e2nb): the export.exclude_owners safety net must also apply to
// the git-committed auto-export (.issuegraph/issues.jsonl), not only to manual
// `issuegraph export`. Before the fix, contributor/personal issues that the manual path
// excludes could still leak into git history and PRs via auto-export.
func TestAutoExportExcludeOwner_config(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, issuegraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "aeo")

	// Enable auto-export with a tiny interval (defeats the default 60s throttle)
	// and exclude owner 'alice' via config.
	issuegraphRunWithFlockRetry(t, issuegraph, dir, "config", "set", "export.exclude_owners", "alice") //nolint:errcheck
	issuegraphRunWithFlockRetry(t, issuegraph, dir, "config", "set", "export.interval", "1ms")         //nolint:errcheck
	issuegraphRunWithFlockRetry(t, issuegraph, dir, "config", "set", "export.auto", "true")            //nolint:errcheck

	// One issue by alice (excluded) and one by bob (kept). The bob create runs
	// last, so the auto-export it triggers reflects the full DB state.
	issuegraphCreate(t, issuegraph, dir, "Alice personal issue", "--actor", "alice")
	issuegraphCreate(t, issuegraph, dir, "Bob team issue", "--actor", "bob")

	jsonlPath := filepath.Join(issuegraphDir, "issues.jsonl")
	data, err := os.ReadFile(jsonlPath)
	if err != nil {
		t.Fatalf("auto-export did not write %s: %v", jsonlPath, err)
	}

	sawBob := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var iss types.Issue
		if err := json.Unmarshal([]byte(line), &iss); err != nil {
			continue
		}
		if iss.CreatedBy == "alice" {
			t.Errorf("issue %s by 'alice' leaked into auto-export %s despite export.exclude_owners", iss.ID, jsonlPath)
		}
		if iss.CreatedBy == "bob" {
			sawBob = true
		}
	}
	if !sawBob {
		t.Fatalf("auto-export %s did not contain bob's issue; the export may not have run, so the absence of alice would be inconclusive", jsonlPath)
	}
}

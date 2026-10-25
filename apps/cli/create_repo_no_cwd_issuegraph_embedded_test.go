//go:build cgo

package main

import (
	"os"
	"strings"
	"testing"
)

// TestEmbeddedCreateRepoFromNonIssueGraphCwd reproduces GH#3686: running
// `issuegraph create --repo=<local path>` from a directory that has no .issuegraph/
// workspace of its own must resolve the target repo's workspace instead of
// failing with "no issuegraph database found".
//
// Before the fix, PersistentPreRun exited early with that error because the
// current directory had no discoverable database, so create.go's --repo
// handling never ran. The reproduction, contributor bug report, and expected
// behavior are due to kevglynn (GH#3774).
func TestEmbeddedCreateRepoFromNonIssueGraphCwd(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt create tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)

	t.Run("repo_flag_resolves_target_workspace", func(t *testing.T) {
		// Target repo with a real .issuegraph/ workspace.
		targetDir, targetIssueGraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "rp")

		// A separate directory with NO .issuegraph/ workspace (and no .issuegraph
		// ancestor, since it is an independent temp dir).
		noIssueGraphCwd := t.TempDir()

		// Sanity: the cwd genuinely has no .issuegraph workspace.
		if _, err := os.Stat(noIssueGraphCwd + "/.issuegraph"); err == nil {
			t.Fatalf("test setup: %s unexpectedly has a .issuegraph dir", noIssueGraphCwd)
		}

		// Create from the non-issuegraph cwd, targeting the other repo. Before the
		// fix this failed with "no issuegraph database found".
		issue := issuegraphCreate(t, issuegraph, noIssueGraphCwd, "Routed from non-issuegraph cwd", "--repo", targetDir)
		if issue.ID == "" {
			t.Fatal("expected issue ID")
		}
		if !strings.HasPrefix(issue.ID, "rp-") {
			t.Errorf("ID should have target prefix rp-, got %q", issue.ID)
		}
		if issue.Title != "Routed from non-issuegraph cwd" {
			t.Errorf("title: got %q, want %q", issue.Title, "Routed from non-issuegraph cwd")
		}

		// The issue must land in the target repo's store.
		assertIssueInStore(t, targetIssueGraphDir, "rp", issue.ID)
	})

	t.Run("no_repo_flag_still_errors_in_non_issuegraph_cwd", func(t *testing.T) {
		// Regression guard: the no-database-found error must still fire for an
		// ordinary create with no --repo when the cwd has no workspace, so the
		// fix does not swallow the diagnostic for the common mistake.
		noIssueGraphCwd := t.TempDir()
		out := issuegraphCreateFail(t, issuegraph, noIssueGraphCwd, "should fail")
		if !strings.Contains(out, "no issuegraph database found") {
			t.Errorf("expected 'no issuegraph database found' error, got:\n%s", out)
		}
	})
}

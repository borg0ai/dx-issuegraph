//go:build cgo

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// issuegraphPrune runs "issuegraph prune" with the given args and returns stdout.
func issuegraphPrune(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"prune"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph prune %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphPruneFail runs "issuegraph prune" expecting failure and returns combined output.
func issuegraphPruneFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"prune"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected issuegraph prune %s to fail, got success:\n%s", strings.Join(args, " "), out)
	}
	return string(out)
}

// createAndClose creates a non-ephemeral task issue and closes it. Returns the ID.
func createAndClose(t *testing.T, issuegraph, dir, title string) string {
	t.Helper()
	issue := issuegraphCreate(t, issuegraph, dir, title)
	issuegraphClose(t, issuegraph, dir, issue.ID)
	return issue.ID
}

func TestEmbeddedPrune(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)

	// ===== Safety gate =====

	t.Run("prune_requires_older_than_or_pattern", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "pr")
		createAndClose(t, issuegraph, dir, "Some closed task")

		// `issuegraph prune --force` alone should fail — the gate blocks it.
		out := issuegraphPruneFail(t, issuegraph, dir, "--force")
		if !strings.Contains(out, "--older-than or --pattern") {
			t.Errorf("expected safety-gate error to mention --older-than or --pattern, got: %s", out)
		}

		// Confirm the issue wasn't deleted.
		listing := issuegraphList(t, issuegraph, dir, "--status=closed", "--json")
		if !strings.Contains(listing, "Some closed task") {
			t.Error("issuegraph prune --force without a filter must be a no-op, but the issue was deleted")
		}
	})

	// ===== Scope: non-ephemeral closed only =====

	t.Run("prune_deletes_closed_non_ephemeral", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "pnx")
		target := createAndClose(t, issuegraph, dir, "Closed non-ephemeral task")

		// --pattern is a scope filter AND the safety gate.
		out := issuegraphPrune(t, issuegraph, dir, "--pattern", "pnx-*", "--force")
		if !strings.Contains(out, "Pruned") && !strings.Contains(out, "pruned") {
			t.Errorf("expected success message, got: %s", out)
		}

		// The target should be gone from the closed list.
		listing := issuegraphList(t, issuegraph, dir, "--status=closed", "--json")
		if strings.Contains(listing, target) {
			t.Errorf("expected %s to be deleted, still present in: %s", target, listing)
		}
	})

	t.Run("prune_last_issue_removes_gitignored_auto_export", func(t *testing.T) {
		dir, issuegraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "pjx")
		if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".issuegraph/\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		for _, args := range [][]string{
			{"config", "set", "export.auto", "true"},
			{"config", "set", "export.git-add", "true"},
		} {
			cmd := exec.Command(issuegraph, args...)
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("issuegraph %s failed: %v\n%s", strings.Join(args, " "), err, out)
			}
		}

		target := createAndClose(t, issuegraph, dir, "Closed exported task")
		jsonlPath := filepath.Join(issuegraphDir, "issues.jsonl")
		cmd := exec.Command(issuegraph, "export", "-o", jsonlPath)
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("issuegraph export failed: %v\n%s", err, out)
		}
		if _, err := os.Stat(jsonlPath); err != nil {
			t.Fatalf("expected stale JSONL before prune: %v", err)
		}
		if err := os.Remove(filepath.Join(issuegraphDir, exportAutoStateFile)); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}

		issuegraphPrune(t, issuegraph, dir, "--pattern", target, "--force")
		if _, err := os.Stat(jsonlPath); !os.IsNotExist(err) {
			t.Fatalf("expected prune of last issue to remove stale JSONL, stat err=%v", err)
		}
	})

	// ===== Scope: must NOT touch ephemeral =====

	t.Run("prune_leaves_closed_ephemeral_for_purge", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "pw")
		wispID := createAndCloseEphemeral(t, issuegraph, dir, "Closed wisp")

		out := issuegraphPrune(t, issuegraph, dir, "--pattern", "pw-*", "--force")
		// Prune's non-ephemeral scope must not match the wisp at all.
		if strings.Contains(out, wispID) {
			t.Errorf("prune wrongly touched ephemeral %s: %s", wispID, out)
		}
		if !strings.Contains(strings.ToLower(out), "no closed issuegraph to prune") {
			t.Errorf("expected prune to skip closed ephemeral %s; got: %s", wispID, out)
		}
	})

	// ===== Scope: must NOT touch open issuegraph =====

	t.Run("prune_leaves_open_issuegraph_alone", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "po")
		openID := issuegraphCreate(t, issuegraph, dir, "Still-open task").ID

		out := issuegraphPrune(t, issuegraph, dir, "--pattern", "po-*", "--force")
		if strings.Contains(out, openID) {
			t.Errorf("prune wrongly touched open issue %s: %s", openID, out)
		}
	})

	// ===== Dry-run shows stats, doesn't touch data =====

	t.Run("prune_dry_run_reports_without_deleting", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "pd")
		target := createAndClose(t, issuegraph, dir, "For dry-run")

		out := issuegraphPrune(t, issuegraph, dir, "--pattern", "pd-*", "--dry-run")
		if !strings.Contains(out, "Would prune") {
			t.Errorf("expected 'Would prune' in dry-run output, got: %s", out)
		}

		// The issue must still exist.
		listing := issuegraphList(t, issuegraph, dir, "--status=closed", "--json")
		if !strings.Contains(listing, target) {
			t.Errorf("dry-run deleted %s; listing: %s", target, listing)
		}
	})

	// ===== Preview mode (no --force) fails without deleting =====

	t.Run("prune_preview_fails_without_force", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "pp")
		target := createAndClose(t, issuegraph, dir, "For preview")

		// Passing a scope filter but no --force: preview-and-abort path.
		out := issuegraphPruneFail(t, issuegraph, dir, "--pattern", "pp-*")
		if !strings.Contains(out, "would prune") && !strings.Contains(out, "Found") {
			t.Errorf("expected preview output, got: %s", out)
		}

		listing := issuegraphList(t, issuegraph, dir, "--status=closed", "--json")
		if !strings.Contains(listing, target) {
			t.Errorf("preview deleted %s; listing: %s", target, listing)
		}
	})

	// ===== --older-than excludes recent issuegraph =====

	t.Run("prune_older_than_skips_recent", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "pot")
		// Just-closed — should NOT match --older-than 30d.
		target := createAndClose(t, issuegraph, dir, "Recent close")

		out := issuegraphPrune(t, issuegraph, dir, "--older-than", "30d", "--force")
		if strings.Contains(out, target) {
			t.Errorf("prune with --older-than 30d wrongly touched just-closed %s: %s", target, out)
		}

		// A "no issuegraph to prune" message (or a zero-count result) is the
		// expected outcome.
		lower := strings.ToLower(out)
		if !strings.Contains(lower, "no ") && !strings.Contains(lower, "0 ") {
			// Not fatal — wording may vary. Ensure target survives.
			listing := issuegraphList(t, issuegraph, dir, "--status=closed", "--json")
			if !strings.Contains(listing, target) {
				t.Errorf("--older-than 30d deleted recent issue %s; listing: %s", target, listing)
			}
		}
	})
}

// TestPrune_protectsIssueCitedByCustomStatusIssue is a regression test for the
// maphew review of PR #4023 (be-e2nb): reference-aware prune must scan issuegraph in
// active *custom* statuses, not only the built-in non-closed statuses. A closed
// issue cited by a issue in an active custom status must be protected from prune,
// exactly like one cited by a built-in open issue.
func TestPrune_protectsIssueCitedByCustomStatusIssue(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "rcs")

	// Define an active custom status.
	issuegraphRunWithFlockRetry(t, issuegraph, dir, "config", "set", "status.custom", "in_review:active") //nolint:errcheck

	// A closed issue that will be cited, and a closed issue nobody cites.
	cited := createAndClose(t, issuegraph, dir, "Closed issue cited by a reviewer issue")
	orphan := createAndClose(t, issuegraph, dir, "Closed orphan issue")

	// An open issue moved into the active custom status, whose description cites
	// the closed issue. With the fix this protects `cited`; without it `cited`
	// is pruned because the custom status is never scanned.
	citing := issuegraphCreate(t, issuegraph, dir, "Reviewer issue", "--description", "depends on "+cited)
	issuegraphRunWithFlockRetry(t, issuegraph, dir, "update", citing.ID, "--status", "in_review") //nolint:errcheck

	// Guard the test's own premise: the citing issue must really be in the custom
	// status, not a built-in non-closed status (which would protect `cited`
	// regardless of the fix and make this test vacuous).
	if got := issuegraphShow(t, issuegraph, dir, citing.ID); got.Status != "in_review" {
		t.Fatalf("setup: expected citing issue %s in custom status in_review, got %q", citing.ID, got.Status)
	}

	issuegraphPrune(t, issuegraph, dir, "--pattern", "rcs-*", "--force")

	// `cited` must survive (protected by the in_review reference); `orphan` must
	// be gone (proving prune actually deleted something).
	if got := issuegraphShow(t, issuegraph, dir, cited); got.ID != cited {
		t.Errorf("expected cited issue %s to survive prune, got %q", cited, got.ID)
	}
	issuegraphShowFail(t, issuegraph, dir, orphan)
}

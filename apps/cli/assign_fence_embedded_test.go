//go:build cgo

package main

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/types"
)

// issuegraphRunFailCode runs an arbitrary issuegraph command expecting failure and returns
// combined output plus the process exit code.
func issuegraphRunFailCode(t *testing.T, issuegraph, dir string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(issuegraph, args...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected issuegraph %s to fail, but it succeeded:\n%s", strings.Join(args, " "), out)
	}
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("issuegraph %s failed without an exit code: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out), ee.ExitCode()
}

// issuegraphRunOK runs an arbitrary issuegraph command expecting success.
func issuegraphRunOK(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command(issuegraph, args...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("issuegraph %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// TestReassignFenceCLI drives the bd-98s5c live-claim reassign fence
// end-to-end: plain `issuegraph update -a` and `issuegraph assign` were the last unfenced
// cross-actor takeover path (--claim, unclaim, and close all refuse without
// --force). The fence must refuse ONLY the silent steamroll — a different
// actor overwriting a live in_progress claim with a third assignee — and must
// leave routine dispatch untouched: open-issue reassigns, self-edits,
// idempotent re-asserts, pool-alias takes, and holder-aware --if-assignee
// transfers all proceed without --force.
func TestReassignFenceCLI(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "rf")

	claimAs := func(t *testing.T, holder string) *types.Issue {
		t.Helper()
		issue := issuegraphCreate(t, issuegraph, dir, "Held work", "--type", "task")
		issuegraphUpdate(t, issuegraph, dir, issue.ID, "--actor", holder, "--assignee", holder, "--status", "in_progress")
		return issue
	}

	t.Run("update_refuses_live_foreign_claim", func(t *testing.T) {
		issue := claimAs(t, "holder")

		out, code := issuegraphUpdateFailCode(t, issuegraph, dir, issue.ID, "--actor", "thief", "--assignee", "thief")
		if code != 1 {
			t.Errorf("fence refusal exit code = %d, want 1 (policy refusal, never 13/guard-mismatch)\n%s", code, out)
		}
		for _, frag := range []string{"holder", "held by", "--force"} {
			if !strings.Contains(out, frag) {
				t.Errorf("refusal should contain %q, got:\n%s", frag, out)
			}
		}
		// The old --claim refusal copy taught the steamroller ("use issuegraph
		// unclaim"); the fence copy must never reintroduce that shape.
		if strings.Contains(out, "use issuegraph unclaim") {
			t.Errorf("refusal copy teaches the unclaim steamroller:\n%s", out)
		}

		got := issuegraphShow(t, issuegraph, dir, issue.ID)
		if got.Assignee != "holder" || got.Status != types.StatusInProgress {
			t.Errorf("refused reassign clobbered the row: assignee=%q status=%s, want holder/in_progress", got.Assignee, got.Status)
		}

		// --force is the sanctioned override for abandoned claims.
		issuegraphUpdate(t, issuegraph, dir, issue.ID, "--actor", "thief", "--assignee", "thief", "--force")
		if got := issuegraphShow(t, issuegraph, dir, issue.ID); got.Assignee != "thief" {
			t.Errorf("--force reassign did not apply: assignee=%q, want thief", got.Assignee)
		}
	})

	t.Run("assign_refuses_live_foreign_claim", func(t *testing.T) {
		issue := claimAs(t, "holder")

		out, code := issuegraphRunFailCode(t, issuegraph, dir, "assign", issue.ID, "thief", "--actor", "thief")
		if code != 1 {
			t.Errorf("assign fence exit code = %d, want 1\n%s", code, out)
		}
		if !strings.Contains(out, "holder") {
			t.Errorf("assign refusal should name the holder, got:\n%s", out)
		}
		if got := issuegraphShow(t, issuegraph, dir, issue.ID); got.Assignee != "holder" {
			t.Errorf("refused assign clobbered the row: assignee=%q, want holder", got.Assignee)
		}

		issuegraphRunOK(t, issuegraph, dir, "assign", issue.ID, "thief", "--actor", "thief", "--force")
		if got := issuegraphShow(t, issuegraph, dir, issue.ID); got.Assignee != "thief" {
			t.Errorf("assign --force did not apply: assignee=%q, want thief", got.Assignee)
		}
	})

	t.Run("unassign_of_live_foreign_claim_refused", func(t *testing.T) {
		issue := claimAs(t, "holder")

		// Stripping a live claim via -a "" is unclaim in disguise; same fence.
		_, code := issuegraphUpdateFailCode(t, issuegraph, dir, issue.ID, "--actor", "thief", "--assignee", "")
		if code != 1 {
			t.Errorf("unassign fence exit code = %d, want 1", code)
		}
		out, _ := issuegraphRunFailCode(t, issuegraph, dir, "assign", issue.ID, "", "--actor", "thief")
		if !strings.Contains(out, "holder") {
			t.Errorf("assign-to-empty refusal should name the holder, got:\n%s", out)
		}
		if got := issuegraphShow(t, issuegraph, dir, issue.ID); got.Assignee != "holder" {
			t.Errorf("refused unassign clobbered the row: assignee=%q, want holder", got.Assignee)
		}
	})

	// NEGATIVE regression fences for the load-bearing status==in_progress
	// clause and the self/idempotent exemptions: routine dispatch must stay
	// frictionless or every dispatcher grows a habitual --force.
	t.Run("open_assigned_reassign_needs_no_force", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Open dispatch", "--type", "task", "--assignee", "alice")
		issuegraphUpdate(t, issuegraph, dir, issue.ID, "--actor", "dispatcher", "--assignee", "bob")
		if got := issuegraphShow(t, issuegraph, dir, issue.ID); got.Assignee != "bob" {
			t.Errorf("open-issue reassign should not be fenced: assignee=%q, want bob", got.Assignee)
		}
	})

	t.Run("holder_edits_own_claim_freely", func(t *testing.T) {
		issue := claimAs(t, "holder")
		issuegraphUpdate(t, issuegraph, dir, issue.ID, "--actor", "holder", "--assignee", "successor")
		if got := issuegraphShow(t, issuegraph, dir, issue.ID); got.Assignee != "successor" {
			t.Errorf("self-reassign should not be fenced: assignee=%q, want successor", got.Assignee)
		}
	})

	t.Run("idempotent_reassert_of_holder_passes", func(t *testing.T) {
		issue := claimAs(t, "holder")
		issuegraphUpdate(t, issuegraph, dir, issue.ID, "--actor", "replayer", "--assignee", "holder")
		if got := issuegraphShow(t, issuegraph, dir, issue.ID); got.Assignee != "holder" {
			t.Errorf("idempotent re-assert should pass: assignee=%q, want holder", got.Assignee)
		}
	})

	t.Run("if_assignee_transfer_needs_no_force", func(t *testing.T) {
		// The holder-aware CAS (bd-wsqvw park transition) names the holder
		// explicitly — nothing silent — and --force/--if-assignee are
		// mutually exclusive, so the fence must not fire under the guard.
		issue := claimAs(t, "holder")
		issuegraphUpdate(t, issuegraph, dir, issue.ID, "--actor", "supervisor", "--if-assignee", "holder", "--assignee", "parked")
		if got := issuegraphShow(t, issuegraph, dir, issue.ID); got.Assignee != "parked" {
			t.Errorf("guarded park should pass without --force: assignee=%q, want parked", got.Assignee)
		}
	})

	t.Run("force_and_if_assignee_mutually_exclusive", func(t *testing.T) {
		issue := claimAs(t, "holder")
		out, _ := issuegraphUpdateFailCode(t, issuegraph, dir, issue.ID, "--force", "--if-assignee", "holder", "--assignee", "x")
		if !strings.Contains(out, "force") || !strings.Contains(out, "if-assignee") {
			t.Errorf("expected flag-exclusion error naming both flags, got:\n%s", out)
		}
	})

	t.Run("claim_conflict_copy_preserved_under_claim_flag", func(t *testing.T) {
		// --claim -a X on a foreign live claim must keep failing through the
		// claim CAS with the canonical "already claimed" copy — the fence
		// defers to it (the CAS is itself the anti-steal gate, and existing
		// automation keys on that message).
		issue := claimAs(t, "holder")
		out, _ := issuegraphUpdateFailCode(t, issuegraph, dir, issue.ID, "--actor", "thief", "--claim", "--assignee", "thief")
		if !strings.Contains(out, "already claimed") && !strings.Contains(out, "already assigned") {
			t.Errorf("expected the claim CAS conflict copy, got:\n%s", out)
		}
		if got := issuegraphShow(t, issuegraph, dir, issue.ID); got.Assignee != "holder" {
			t.Errorf("claim+assignee combo clobbered the row: assignee=%q, want holder", got.Assignee)
		}
	})

	t.Run("pool_alias_holder_is_takeable", func(t *testing.T) {
		// The claim.pools carve-out: --claim deliberately treats a
		// pool-assigned issue as claimable by any actor; without the same
		// carve-out, --claim and -a would give opposite answers on the same
		// issue — and the refusal would say "coordinate with the holder" when
		// the holder is a queue alias with nobody behind it.
		issuegraphRunOK(t, issuegraph, dir, "config", "set", "claim.pools", "fable-crew")
		issue := issuegraphCreate(t, issuegraph, dir, "Queue item", "--type", "task")
		issuegraphUpdate(t, issuegraph, dir, issue.ID, "--actor", "dispatcher", "--assignee", "fable-crew", "--status", "in_progress")

		issuegraphUpdate(t, issuegraph, dir, issue.ID, "--actor", "worker-1", "--assignee", "worker-1")
		if got := issuegraphShow(t, issuegraph, dir, issue.ID); got.Assignee != "worker-1" {
			t.Errorf("pool-alias take should pass without --force: assignee=%q, want worker-1", got.Assignee)
		}
	})
}

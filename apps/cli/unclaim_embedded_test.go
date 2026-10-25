//go:build cgo

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/types"
)

// issuegraphUnclaim runs "issuegraph unclaim" with the given args and returns stdout.
// Retries on flock contention.
func issuegraphUnclaim(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"unclaim"}, args...)
	out, err := issuegraphRunWithFlockRetry(t, issuegraph, dir, fullArgs...)
	if err != nil {
		t.Fatalf("issuegraph unclaim %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// issuegraphUnclaimFail runs "issuegraph unclaim" expecting failure.
func issuegraphUnclaimFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"unclaim"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected issuegraph unclaim %s to fail, but it succeeded:\n%s", strings.Join(args, " "), out)
	}
	return string(out)
}

func TestEmbeddedUnclaim(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "tu")

	// ===== Basic Success =====

	t.Run("unclaim_from_in_progress", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Unclaim test", "--type", "task")
		issuegraphUpdate(t, issuegraph, dir, issue.ID, "--claim")
		issuegraphUnclaim(t, issuegraph, dir, issue.ID)
		got := issuegraphShow(t, issuegraph, dir, issue.ID)
		if got.Assignee != "" {
			t.Errorf("expected assignee to be empty after unclaim, got %q", got.Assignee)
		}
		if got.Status != types.StatusOpen {
			t.Errorf("expected status open after unclaim, got %s", got.Status)
		}
	})

	t.Run("unclaim_from_open_stuck_claim", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Stuck claim test", "--type", "task")
		// Manually set assignee without changing status (simulates stuck claim).
		issuegraphUpdate(t, issuegraph, dir, issue.ID, "--assignee", "alice")
		// Release as the owner: ownership enforcement (fe908d40b) rejects a
		// non-owner release without --force.
		issuegraphUnclaim(t, issuegraph, dir, issue.ID, "--actor", "alice")
		got := issuegraphShow(t, issuegraph, dir, issue.ID)
		if got.Assignee != "" {
			t.Errorf("expected assignee to be empty after unclaim, got %q", got.Assignee)
		}
		if got.Status != types.StatusOpen {
			t.Errorf("expected status open after unclaim, got %s", got.Status)
		}
	})

	// ===== With Reason =====

	t.Run("unclaim_with_reason", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Reason test", "--type", "task")
		issuegraphUpdate(t, issuegraph, dir, issue.ID, "--claim")
		out := issuegraphUnclaim(t, issuegraph, dir, issue.ID, "--reason", "Agent crashed")
		if !strings.Contains(out, "Agent crashed") {
			t.Errorf("expected reason in output, got: %s", out)
		}
		got := issuegraphShow(t, issuegraph, dir, issue.ID)
		if got.Assignee != "" {
			t.Errorf("expected assignee to be empty after unclaim, got %q", got.Assignee)
		}
		if got.Status != types.StatusOpen {
			t.Errorf("expected status open after unclaim, got %s", got.Status)
		}
	})

	// ===== JSON Output =====

	t.Run("unclaim_json_output", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "JSON test", "--type", "task")
		issuegraphUpdate(t, issuegraph, dir, issue.ID, "--claim")
		out := issuegraphUnclaim(t, issuegraph, dir, issue.ID, "--json")
		// issuegraph unclaim --json emits an array of the affected issues. A cleared
		// assignee is empty and omitempty drops it from the payload, so parse
		// and assert on the decoded fields rather than raw substrings.
		var unclaimed []struct {
			Assignee string `json:"assignee"`
			Status   string `json:"status"`
		}
		if err := json.Unmarshal([]byte(out), &unclaimed); err != nil {
			t.Fatalf("failed to parse unclaim --json output: %v\n%s", err, out)
		}
		if len(unclaimed) != 1 {
			t.Fatalf("expected 1 issue in unclaim --json output, got %d:\n%s", len(unclaimed), out)
		}
		if unclaimed[0].Assignee != "" {
			t.Errorf("expected empty assignee after unclaim, got %q", unclaimed[0].Assignee)
		}
		if unclaimed[0].Status != "open" {
			t.Errorf("expected status open after unclaim, got %q", unclaimed[0].Status)
		}
	})

	// ===== Multiple IDs =====

	t.Run("unclaim_multiple_ids", func(t *testing.T) {
		issue1 := issuegraphCreate(t, issuegraph, dir, "Multi test 1", "--type", "task")
		issue2 := issuegraphCreate(t, issuegraph, dir, "Multi test 2", "--type", "task")
		issuegraphUpdate(t, issuegraph, dir, issue1.ID, "--claim")
		issuegraphUpdate(t, issuegraph, dir, issue2.ID, "--claim")
		issuegraphUnclaim(t, issuegraph, dir, issue1.ID, issue2.ID)
		got1 := issuegraphShow(t, issuegraph, dir, issue1.ID)
		got2 := issuegraphShow(t, issuegraph, dir, issue2.ID)
		if got1.Assignee != "" {
			t.Errorf("expected assignee to be empty for issue1, got %q", got1.Assignee)
		}
		if got2.Assignee != "" {
			t.Errorf("expected assignee to be empty for issue2, got %q", got2.Assignee)
		}
		if got1.Status != types.StatusOpen {
			t.Errorf("expected status open for issue1, got %s", got1.Status)
		}
		if got2.Status != types.StatusOpen {
			t.Errorf("expected status open for issue2, got %s", got2.Status)
		}
	})

	// ===== Error Cases =====

	t.Run("unclaim_non_existent_issue", func(t *testing.T) {
		out := issuegraphUnclaimFail(t, issuegraph, dir, "nonexistent")
		if !strings.Contains(out, "not found") && !strings.Contains(out, "Error") {
			t.Errorf("expected error about non-existent issue, got: %s", out)
		}
	})

	t.Run("unclaim_closed_issue", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Closed test", "--type", "task")
		issuegraphUpdate(t, issuegraph, dir, issue.ID, "--claim")
		// Close the issue first
		issuegraphClose(t, issuegraph, dir, issue.ID)
		out := issuegraphUnclaimFail(t, issuegraph, dir, issue.ID)
		if !strings.Contains(out, "cannot unclaim closed issue") {
			t.Errorf("expected error about closed issue, got: %s", out)
		}
	})

	t.Run("unclaim_unassigned_issue", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Unassigned test", "--type", "task")
		out := issuegraphUnclaimFail(t, issuegraph, dir, issue.ID)
		if !strings.Contains(out, "is not assigned") {
			t.Errorf("expected error about unassigned issue, got: %s", out)
		}
	})

	// ===== Ownership =====

	// A non-owner may not release another agent's claim: the release is rejected
	// with ErrNotOwner (non-zero exit) and the claim is left intact.
	t.Run("unclaim_non_owner_rejected", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Owned by alice", "--type", "task")
		issuegraphUpdate(t, issuegraph, dir, issue.ID, "--claim", "--actor", "alice")

		out := issuegraphUnclaimFail(t, issuegraph, dir, issue.ID, "--actor", "bob")
		// ErrNotOwner surfaces as "issue claimed by a different actor: ... held by alice".
		if !strings.Contains(out, "held by alice") && !strings.Contains(out, "claimed by a different actor") {
			t.Errorf("expected ownership-rejection error, got: %s", out)
		}
		// The rejection should frame --force as an abandoned-claim escape
		// hatch, not a routine override (bd-at6rc / wy-zs5s2).
		if !strings.Contains(out, "coordinate with the holder") {
			t.Errorf("rejection should say coordinate-with-holder, got: %s", out)
		}

		// The claim must be untouched after the rejected release.
		got := issuegraphShow(t, issuegraph, dir, issue.ID)
		if got.Assignee != "alice" {
			t.Errorf("expected assignee to remain alice after rejected unclaim, got %q", got.Assignee)
		}
		if got.Status != types.StatusInProgress {
			t.Errorf("expected status to remain in_progress after rejected unclaim, got %s", got.Status)
		}
	})

	// --force lets a non-owner (admin/reaper) release someone else's claim.
	t.Run("unclaim_non_owner_force_succeeds", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Force release", "--type", "task")
		issuegraphUpdate(t, issuegraph, dir, issue.ID, "--claim", "--actor", "alice")

		issuegraphUnclaim(t, issuegraph, dir, issue.ID, "--force", "--actor", "bob")

		got := issuegraphShow(t, issuegraph, dir, issue.ID)
		if got.Assignee != "" {
			t.Errorf("expected assignee to be cleared after --force unclaim, got %q", got.Assignee)
		}
		if got.Status != types.StatusOpen {
			t.Errorf("expected status open after --force unclaim, got %s", got.Status)
		}
	})

	// The owner may release its own claim under a different spelling of its own
	// identity (ga-5ksp5): dot vs underscore separator is the same Gas Town
	// identity at two different layers, and both the Go-side ownership
	// precheck and the SQL CAS predicate must recognize that — not just the
	// former, or the UPDATE would silently affect zero rows and the release
	// would fail even though the Go precheck said it should succeed.
	t.Run("unclaim_owner_different_spelling_succeeds", func(t *testing.T) {
		issue := issuegraphCreate(t, issuegraph, dir, "Owner spelling variant", "--type", "task")
		issuegraphUpdate(t, issuegraph, dir, issue.ID, "--claim", "--actor", "gastown.mayor")

		issuegraphUnclaim(t, issuegraph, dir, issue.ID, "--actor", "gastown_mayor")

		got := issuegraphShow(t, issuegraph, dir, issue.ID)
		if got.Assignee != "" {
			t.Errorf("expected assignee to be cleared after same-identity unclaim, got %q", got.Assignee)
		}
		if got.Status != types.StatusOpen {
			t.Errorf("expected status open after same-identity unclaim, got %s", got.Status)
		}
	})
}

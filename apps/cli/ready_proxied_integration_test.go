//go:build cgo

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/storage/depid"
	"github.com/borg0ai/dx-issuegraph/internal/types"
)

func issuegraphProxiedClaimCmd(issuegraph, dir, actor string) *exec.Cmd {
	cmd := exec.Command(issuegraph, "ready", "--claim", "--json", "--label", "atomic")
	cmd.Dir = dir
	env := issuegraphProxiedEnv(dir)
	env = append(env, "ISSUEGRAPH_ACTOR="+actor)
	cmd.Env = env
	return cmd
}

func issuegraphProxiedReadyJSON(t *testing.T, issuegraph string, p proxiedProject, args ...string) []*types.IssueWithCounts {
	t.Helper()
	fullArgs := append([]string{"ready", "--json"}, args...)
	stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, fullArgs...)
	if err != nil {
		t.Fatalf("issuegraph ready --json %s failed: %v\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), err, stdout, stderr)
	}
	s := strings.TrimSpace(stdout)
	start := strings.Index(s, "[")
	if start < 0 {
		t.Fatalf("no JSON array in ready --json output:\n%s", stdout)
	}
	var issues []*types.IssueWithCounts
	if err := json.Unmarshal([]byte(s[start:]), &issues); err != nil {
		t.Fatalf("parse ready JSON: %v\n%s", err, s[start:])
	}
	return issues
}

func issuegraphProxiedReadyCapture(t *testing.T, issuegraph string, p proxiedProject, args ...string) (string, string) {
	t.Helper()
	fullArgs := append([]string{"ready"}, args...)
	stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, fullArgs...)
	if err != nil {
		t.Fatalf("issuegraph ready %s failed: %v\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), err, stdout, stderr)
	}
	return stdout, stderr
}

func issuegraphProxiedReadyFail(t *testing.T, issuegraph string, p proxiedProject, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"ready"}, args...)
	stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, fullArgs...)
	if err == nil {
		t.Fatalf("issuegraph ready %s should have failed; stdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), stdout, stderr)
	}
	return stdout + stderr
}

func TestProxiedServerReady(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)

	t.Run("json_round_trips_issue_with_counts", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rdj1")
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Zero deps", "--label", "rdj1-only")
		ready := issuegraphProxiedReadyJSON(t, issuegraph, p, "--label", "rdj1-only")
		if len(ready) != 1 {
			t.Fatalf("ready count = %d, want 1", len(ready))
		}
		if ready[0].ID != issue.ID {
			t.Fatalf("ready[0].ID = %s, want %s", ready[0].ID, issue.ID)
		}
		if ready[0].DependencyCount != 0 {
			t.Errorf("DependencyCount = %d, want 0", ready[0].DependencyCount)
		}
	})

	t.Run("label_pattern_and_regex_filter", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rdlpr")
		matched := issuegraphProxiedCreate(t, issuegraph, p.dir, "Matched", "--label", "tech-debt")
		issuegraphProxiedCreate(t, issuegraph, p.dir, "Unmatched", "--label", "product")

		for _, args := range [][]string{{"--label-pattern", "tech-*"}, {"--label-regex", "^tech-(debt|legacy)$"}} {
			ready := issuegraphProxiedReadyJSON(t, issuegraph, p, args...)
			if len(ready) != 1 || ready[0].ID != matched.ID {
				t.Errorf("issuegraph ready %s = %#v, want only %s", strings.Join(args, " "), ready, matched.ID)
			}
		}
	})

	t.Run("default_text_output", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rdt")
		issuegraphProxiedCreate(t, issuegraph, p.dir, "Ready title here")
		stdout, _ := issuegraphProxiedReadyCapture(t, issuegraph, p)
		if !strings.Contains(stdout, "Ready title here") {
			t.Errorf("expected issue title in output, got: %s", stdout)
		}
	})

	t.Run("json_validity", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rdv")
		issuegraphProxiedCreate(t, issuegraph, p.dir, "JSON test")
		stdout, _, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "ready", "--json")
		if err != nil {
			t.Fatalf("issuegraph ready --json failed: %v\n%s", err, stdout)
		}
		s := strings.TrimSpace(stdout)
		start := strings.IndexAny(s, "[{")
		if start < 0 {
			t.Fatalf("no JSON in output: %s", s)
		}
		if !json.Valid([]byte(s[start:])) {
			t.Errorf("invalid JSON: %s", s[start:])
		}
	})

	t.Run("limit_zero_returns_full_set_and_suppresses_hint", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rdl0")
		const n = 4
		want := make(map[string]bool, n)
		for i := 0; i < n; i++ {
			issue := issuegraphProxiedCreate(t, issuegraph, p.dir, fmt.Sprintf("L0 item %d", i), "--label", "l0")
			want[issue.ID] = true
		}
		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir,
			"ready", "--json", "--limit", "0", "--label", "l0")
		if err != nil {
			t.Fatalf("issuegraph ready --json --limit 0 failed: %v\nstderr: %s", err, stderr)
		}
		if strings.Contains(stderr, "Use --limit 0 for all") {
			t.Errorf("--limit 0 must not emit truncation hint, got stderr: %s", stderr)
		}
		s := strings.TrimSpace(stdout)
		start := strings.Index(s, "[")
		if start < 0 {
			t.Fatalf("no JSON array in stdout: %s", stdout)
		}
		var got []*types.IssueWithCounts
		if err := json.Unmarshal([]byte(s[start:]), &got); err != nil {
			t.Fatalf("parse JSON: %v\n%s", err, s[start:])
		}
		gotIDs := map[string]bool{}
		for _, r := range got {
			gotIDs[r.ID] = true
		}
		for id := range want {
			if !gotIDs[id] {
				t.Errorf("expected %s in --limit 0 result", id)
			}
		}
		if len(got) < n {
			t.Errorf("got %d issues under --limit 0, want >=%d", len(got), n)
		}
	})

	t.Run("limit_truncation_hint_on_stderr", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rdl")
		for i := 0; i < 4; i++ {
			issuegraphProxiedCreate(t, issuegraph, p.dir, fmt.Sprintf("Ready item %d", i))
		}
		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "ready", "--json", "--limit", "2")
		if err != nil {
			t.Fatalf("issuegraph ready --json --limit 2 failed: %v\nstderr: %s\nstdout: %s", err, stderr, stdout)
		}
		s := strings.TrimSpace(stdout)
		start := strings.Index(s, "[")
		if start < 0 || !json.Valid([]byte(s[start:])) {
			t.Fatalf("stdout JSON should remain parseable, got: %s", stdout)
		}
		if !strings.Contains(stderr, "Use --limit 0 for all") {
			t.Fatalf("expected truncation hint on stderr, got: %q", stderr)
		}
		if !strings.Contains(stderr, "more matched but were hidden by --limit") {
			t.Errorf("expected HasMore-based hint wording on stderr, got: %q", stderr)
		}
	})

	t.Run("pagination_envelope_truncated", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rdpgt")
		for i := 0; i < 4; i++ {
			issuegraphProxiedCreate(t, issuegraph, p.dir, fmt.Sprintf("Pag item %d", i), "--label", "rdpgt")
		}
		stdout, _, err := issuegraphProxiedRunBuffersWithEnv(t, issuegraph, p.dir, []string{"ISSUEGRAPH_JSON_ENVELOPE=1"},
			"ready", "--json", "--limit", "2", "--label", "rdpgt")
		if err != nil {
			t.Fatalf("issuegraph ready --json --limit 2 failed: %v\nstdout: %s", err, stdout)
		}
		var envelope struct {
			Data       []*types.IssueWithCounts `json:"data"`
			Pagination *PaginationMeta          `json:"pagination"`
		}
		s := strings.TrimSpace(stdout)
		start := strings.Index(s, "{")
		if start < 0 {
			t.Fatalf("no JSON envelope object in stdout: %s", stdout)
		}
		if err := json.Unmarshal([]byte(s[start:]), &envelope); err != nil {
			t.Fatalf("parse envelope JSON: %v\n%s", err, s[start:])
		}
		if len(envelope.Data) != 2 {
			t.Fatalf("envelope.data length = %d, want 2", len(envelope.Data))
		}
		if envelope.Pagination == nil {
			t.Fatal("missing 'pagination' key in truncated envelope")
		}
		if !envelope.Pagination.Truncated {
			t.Error("pagination.truncated = false, want true")
		}
		if envelope.Pagination.Returned != 2 {
			t.Errorf("pagination.returned = %d, want 2", envelope.Pagination.Returned)
		}
		// Total is PUBLISHED here, which it was not before the ReadyCounter
		// role: this route had no way to size the ready set, so the direct
		// route's "total" simply had no proxied counterpart and a script that
		// read it got nothing under a proxied workspace. Four rows were seeded
		// and two returned.
		if envelope.Pagination.Total != 4 {
			t.Errorf("pagination.total = %d, want the 4 rows this case seeded: %s",
				envelope.Pagination.Total, s[start:])
		}
	})

	t.Run("pagination_envelope_not_truncated", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rdpgn")
		issuegraphProxiedCreate(t, issuegraph, p.dir, "Pag not truncated", "--label", "rdpgn")
		stdout, _, err := issuegraphProxiedRunBuffersWithEnv(t, issuegraph, p.dir, []string{"ISSUEGRAPH_JSON_ENVELOPE=1"},
			"ready", "--json", "--limit", "10", "--label", "rdpgn")
		if err != nil {
			t.Fatalf("issuegraph ready --json --limit 10 failed: %v\nstdout: %s", err, stdout)
		}
		var envelope map[string]json.RawMessage
		s := strings.TrimSpace(stdout)
		start := strings.Index(s, "{")
		if start < 0 {
			t.Fatalf("no JSON envelope object in stdout: %s", stdout)
		}
		if err := json.Unmarshal([]byte(s[start:]), &envelope); err != nil {
			t.Fatalf("parse envelope JSON: %v\n%s", err, s[start:])
		}
		// Parity with the direct route: no pagination key when not truncated.
		if raw, ok := envelope["pagination"]; ok {
			t.Errorf("unexpected 'pagination' key when not truncated: %s", raw)
		}
	})

	t.Run("offset_with_large_finite_limit", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rdofflz")
		for i := 0; i < 4; i++ {
			issuegraphProxiedCreate(t, issuegraph, p.dir,
				fmt.Sprintf("Lz item %d", i), "--label", "rdofflz")
		}
		got := issuegraphProxiedReadyJSON(t, issuegraph, p,
			"--offset", "1", "--limit", "1000", "--label", "rdofflz")
		if len(got) != 3 {
			t.Errorf("expected 3 items (4 created, offset 1, loose limit), got %d", len(got))
		}
	})

	t.Run("offset_zero_equals_no_offset", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rdoff0")
		for i := 0; i < 3; i++ {
			issuegraphProxiedCreate(t, issuegraph, p.dir,
				fmt.Sprintf("Eq item %d", i), "--label", "rdoff0")
		}
		baseline := issuegraphProxiedReadyJSON(t, issuegraph, p,
			"--limit", "10", "--label", "rdoff0")
		withZero := issuegraphProxiedReadyJSON(t, issuegraph, p,
			"--offset", "0", "--limit", "10", "--label", "rdoff0")
		if len(baseline) != len(withZero) {
			t.Fatalf("--offset 0 must equal no --offset, lengths differ: %d vs %d", len(baseline), len(withZero))
		}
		for i := range baseline {
			if baseline[i].ID != withZero[i].ID {
				t.Errorf("position %d: baseline=%s, withZero=%s", i, baseline[i].ID, withZero[i].ID)
			}
		}
	})

	t.Run("offset_combo_guards", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rdoffg")
		issuegraphProxiedCreate(t, issuegraph, p.dir, "Seed")
		cases := []struct {
			name string
			args []string
		}{
			{"offset_claim", []string{"--offset", "1", "--claim"}},
			{"offset_mol", []string{"--offset", "1", "--mol", "x"}},
			{"offset_gated", []string{"--offset", "1", "--gated"}},
			{"offset_explain", []string{"--offset", "1", "--explain"}},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				out := issuegraphProxiedReadyFail(t, issuegraph, p, c.args...)
				if !strings.Contains(out, "--offset cannot be combined") {
					t.Errorf("expected '--offset cannot be combined' error, got: %s", out)
				}
			})
		}
	})

	// The rejection lives on this route only: outside proxied mode a negative
	// --offset is ignored (see TestEmbeddedReady's
	// negative_offset_ignored_outside_proxied), because --offset is not
	// supported there at all.
	t.Run("negative_offset_rejected", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rdoffneg")
		out := issuegraphProxiedReadyFail(t, issuegraph, p, "--offset", "-1")
		if !strings.Contains(out, "--offset must be >= 0") {
			t.Errorf("expected '--offset must be >= 0' error, got: %s", out)
		}
	})

	t.Run("offset_skips_leading_results", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rdoff")
		ids := make([]string, 4)
		for i := 0; i < 4; i++ {
			issue := issuegraphProxiedCreate(t, issuegraph, p.dir,
				fmt.Sprintf("Off item %d", i), "-p", "1", "--label", "rdoff")
			ids[i] = issue.ID
		}
		got := issuegraphProxiedReadyJSON(t, issuegraph, p,
			"--offset", "2", "--limit", "2", "--label", "rdoff")
		if len(got) != 2 {
			t.Fatalf("expected exactly 2 items after offset+limit, got %d: %+v", len(got), got)
		}
		gotIDs := []string{got[0].ID, got[1].ID}
		first := issuegraphProxiedReadyJSON(t, issuegraph, p, "--limit", "0", "--label", "rdoff")
		if len(first) < 4 {
			t.Fatalf("baseline returned %d, need 4 to assert ordering", len(first))
		}
		wantA, wantB := first[2].ID, first[3].ID
		if gotIDs[0] != wantA || gotIDs[1] != wantB {
			t.Errorf("offset=2 limit=2 returned %v, want [%s %s] (the 3rd+4th of the unpaginated set)",
				gotIDs, wantA, wantB)
		}
	})

	t.Run("claim_json_no_match_returns_empty_array", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rcn")
		issuegraphProxiedCreate(t, issuegraph, p.dir, "Has wrong label", "--label", "real")
		out, err := issuegraphProxiedRun(t, issuegraph, p.dir, "ready", "--claim", "--json", "--label", "missing-label")
		if err != nil {
			t.Fatalf("issuegraph ready --claim --json with no match failed: %v\n%s", err, out)
		}
		var empty []types.IssueWithCounts
		if err := json.Unmarshal(bytes.TrimSpace(out), &empty); err != nil {
			t.Fatalf("parse empty claim JSON: %v\n%s", err, out)
		}
		if len(empty) != 0 {
			t.Fatalf("expected no claimed issues, got %d: %s", len(empty), out)
		}
	})

	t.Run("claim_json_marks_in_progress_and_assignee", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rcm")
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Claim me", "--label", "claim-me")
		out, err := issuegraphProxiedRun(t, issuegraph, p.dir, "ready", "--claim", "--json", "--label", "claim-me")
		if err != nil {
			t.Fatalf("issuegraph ready --claim --json failed: %v\n%s", err, out)
		}
		var claimed []types.IssueWithCounts
		if err := json.Unmarshal(bytes.TrimSpace(out), &claimed); err != nil {
			t.Fatalf("parse claim JSON: %v\n%s", err, out)
		}
		if len(claimed) != 1 {
			t.Fatalf("expected one claimed issue, got %d: %s", len(claimed), out)
		}
		if claimed[0].ID != issue.ID {
			t.Fatalf("claimed ID = %s, want %s", claimed[0].ID, issue.ID)
		}
		if claimed[0].Status != types.StatusInProgress {
			t.Errorf("Status = %s, want %s", claimed[0].Status, types.StatusInProgress)
		}
		if claimed[0].Assignee == "" {
			t.Error("Assignee should be set after claim")
		}
		shown := issuegraphProxiedShow(t, issuegraph, p.dir, issue.ID)
		if shown.Status != types.StatusInProgress {
			t.Errorf("after re-fetch: Status = %s, want %s", shown.Status, types.StatusInProgress)
		}
		if shown.Assignee == "" {
			t.Error("after re-fetch: Assignee empty (commit may not have persisted)")
		}
	})

	t.Run("blocked_issues_excluded_from_default_output", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rdb")
		blocker := issuegraphProxiedCreate(t, issuegraph, p.dir, "Blocker issue")
		issuegraphProxiedCreate(t, issuegraph, p.dir, "Should be hidden because blocked", "--deps", "depends-on:"+blocker.ID)
		stdout, _ := issuegraphProxiedReadyCapture(t, issuegraph, p)
		if strings.Contains(stdout, "Should be hidden because blocked") {
			t.Errorf("blocked issue must not appear in ready output: %s", stdout)
		}
	})

	t.Run("exclude_label_filter", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rdex")
		issuegraphProxiedCreate(t, issuegraph, p.dir, "Triage pending item", "--label", "triage:pending")
		issuegraphProxiedCreate(t, issuegraph, p.dir, "Normal ready item")
		stdout, _ := issuegraphProxiedReadyCapture(t, issuegraph, p, "--exclude-label", "triage:pending")
		if strings.Contains(stdout, "Triage pending item") {
			t.Errorf("triage:pending issue must be excluded: %s", stdout)
		}
		if !strings.Contains(stdout, "Normal ready item") {
			t.Errorf("normal issue must appear: %s", stdout)
		}
	})

	t.Run("explain_json_round_trip", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rde")
		issuegraphProxiedCreate(t, issuegraph, p.dir, "Plain ready")
		blocker := issuegraphProxiedCreate(t, issuegraph, p.dir, "Live blocker")
		issuegraphProxiedCreate(t, issuegraph, p.dir, "Blocked dependent", "--deps", "depends-on:"+blocker.ID)
		stdout, _, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "ready", "--explain", "--json")
		if err != nil {
			t.Fatalf("issuegraph ready --explain --json failed: %v\n%s", err, stdout)
		}
		s := strings.TrimSpace(stdout)
		start := strings.Index(s, "{")
		if start < 0 {
			t.Fatalf("no JSON in --explain output: %s", stdout)
		}
		var exp types.ReadyExplanation
		if err := json.Unmarshal([]byte(s[start:]), &exp); err != nil {
			t.Fatalf("parse ReadyExplanation: %v\n%s", err, s[start:])
		}
		if exp.Summary.TotalReady < 1 {
			t.Errorf("Summary.TotalReady = %d, want >=1", exp.Summary.TotalReady)
		}
		if exp.Summary.TotalBlocked < 1 {
			t.Errorf("Summary.TotalBlocked = %d, want >=1", exp.Summary.TotalBlocked)
		}
	})

	t.Run("explain_text_header", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rdet")
		issuegraphProxiedCreate(t, issuegraph, p.dir, "Smoke")
		stdout, _, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "ready", "--explain")
		if err != nil {
			t.Fatalf("issuegraph ready --explain failed: %v\n%s", err, stdout)
		}
		if !strings.Contains(stdout, "Ready Work Explanation") {
			t.Errorf("expected explain heading, got: %s", stdout)
		}
	})

	t.Run("explain_detects_cycles", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rdc")
		a := issuegraphProxiedCreate(t, issuegraph, p.dir, "Cycle node A")
		b := issuegraphProxiedCreate(t, issuegraph, p.dir, "Cycle node B")
		db := openProxiedDB(t, p)
		ctx := context.Background()
		if _, err := db.ExecContext(ctx,
			"INSERT INTO dependencies (id, issue_id, depends_on_issue_id, type, created_at, created_by) VALUES (?, ?, ?, ?, NOW(), 'test')",
			depid.New(a.ID, b.ID), a.ID, b.ID, string(types.DepBlocks)); err != nil {
			t.Fatalf("plant a->b edge: %v", err)
		}
		if _, err := db.ExecContext(ctx,
			"INSERT INTO dependencies (id, issue_id, depends_on_issue_id, type, created_at, created_by) VALUES (?, ?, ?, ?, NOW(), 'test')",
			depid.New(b.ID, a.ID), b.ID, a.ID, string(types.DepBlocks)); err != nil {
			t.Fatalf("plant b->a edge: %v", err)
		}
		stdout, _, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "ready", "--explain", "--json")
		if err != nil {
			t.Fatalf("issuegraph ready --explain --json after cycle plant: %v\n%s", err, stdout)
		}
		s := strings.TrimSpace(stdout)
		start := strings.Index(s, "{")
		if start < 0 {
			t.Fatalf("no JSON in --explain output: %s", stdout)
		}
		var exp types.ReadyExplanation
		if err := json.Unmarshal([]byte(s[start:]), &exp); err != nil {
			t.Fatalf("parse ReadyExplanation: %v\n%s", err, s[start:])
		}
		if len(exp.Cycles) == 0 {
			t.Errorf("expected non-empty Cycles after planting a->b->a, got: %+v", exp)
		}
	})

	t.Run("gated_empty_case", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rdge")
		issuegraphProxiedCreate(t, issuegraph, p.dir, "No gates anywhere")
		stdout, _, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "ready", "--gated", "--json")
		if err != nil {
			t.Fatalf("issuegraph ready --gated --json failed: %v\n%s", err, stdout)
		}
		s := strings.TrimSpace(stdout)
		start := strings.Index(s, "{")
		if start < 0 {
			t.Fatalf("no JSON in gated output: %s", stdout)
		}
		var out GatedReadyOutput
		if err := json.Unmarshal([]byte(s[start:]), &out); err != nil {
			t.Fatalf("parse GatedReadyOutput: %v\n%s", err, s[start:])
		}
		if len(out.Molecules) != 0 {
			t.Errorf("expected empty molecules array, got %d", len(out.Molecules))
		}

		text, _ := issuegraphProxiedReadyCapture(t, issuegraph, p, "--gated")
		if !strings.Contains(text, "No molecules ready for gate-resume dispatch") {
			t.Errorf("expected 'No molecules ready for gate-resume dispatch' text, got: %s", text)
		}
	})

}

func TestProxiedServerReady2(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)

	t.Run("gated_json_shape_with_open_gate", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rdgw")
		issuegraphProxiedCreate(t, issuegraph, p.dir, "Open gate", "--type", "gate")
		stdout, _, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "ready", "--gated", "--json")
		if err != nil {
			t.Fatalf("issuegraph ready --gated --json failed: %v\n%s", err, stdout)
		}
		s := strings.TrimSpace(stdout)
		start := strings.Index(s, "{")
		if start < 0 {
			t.Fatalf("no JSON in gated output: %s", stdout)
		}
		var out GatedReadyOutput
		if err := json.Unmarshal([]byte(s[start:]), &out); err != nil {
			t.Fatalf("parse GatedReadyOutput: %v\n%s", err, s[start:])
		}
		if len(out.Molecules) != 0 {
			t.Errorf("open (non-closed) gate must not appear in gated output, got %d", len(out.Molecules))
		}
	})

	t.Run("mol_text_and_json", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rdm")
		mol := issuegraphProxiedCreate(t, issuegraph, p.dir, "Mol parent", "--type", "molecule")
		s1 := issuegraphProxiedCreate(t, issuegraph, p.dir, "Step one", "--parent", mol.ID)
		issuegraphProxiedCreate(t, issuegraph, p.dir, "Step two", "--parent", mol.ID, "--deps", "depends-on:"+s1.ID)
		stdout, _, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "ready", "--mol", mol.ID, "--json")
		if err != nil {
			t.Fatalf("issuegraph ready --mol --json failed: %v\n%s", err, stdout)
		}
		sj := strings.TrimSpace(stdout)
		start := strings.Index(sj, "{")
		if start < 0 {
			t.Fatalf("no JSON in --mol output: %s", stdout)
		}
		var out MoleculeReadyOutput
		if err := json.Unmarshal([]byte(sj[start:]), &out); err != nil {
			t.Fatalf("parse MoleculeReadyOutput: %v\n%s", err, sj[start:])
		}
		if out.TotalSteps < 2 {
			t.Errorf("TotalSteps = %d, want >=2", out.TotalSteps)
		}
		if out.ReadySteps < 1 {
			t.Errorf("ReadySteps = %d, want >=1", out.ReadySteps)
		}

		text, _ := issuegraphProxiedReadyCapture(t, issuegraph, p, "--mol", mol.ID)
		if !strings.Contains(text, "Mol parent") {
			t.Errorf("expected molecule title in text output, got: %s", text)
		}
	})

	t.Run("empty_state_messages", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rde")
		stdout, _ := issuegraphProxiedReadyCapture(t, issuegraph, p)
		if !strings.Contains(stdout, "No open issues") {
			t.Errorf("expected 'No open issues', got: %s", stdout)
		}

		blocked := issuegraphProxiedCreate(t, issuegraph, p.dir, "Blocked open issue")
		db := openProxiedDB(t, p)
		if _, err := db.ExecContext(context.Background(), "UPDATE issues SET is_blocked = 1 WHERE id = ?", blocked.ID); err != nil {
			t.Fatalf("mark issue blocked: %v", err)
		}
		stdout, _ = issuegraphProxiedReadyCapture(t, issuegraph, p)
		if !strings.Contains(stdout, "No ready work found") {
			t.Errorf("expected 'No ready work found', got: %s", stdout)
		}
	})

	t.Run("include_ephemeral_surfaces_wisps", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rdie")
		reg := issuegraphProxiedCreate(t, issuegraph, p.dir, "Regular")
		wisp := issuegraphProxiedCreate(t, issuegraph, p.dir, "Wispy", "--ephemeral")
		def := issuegraphProxiedReadyJSON(t, issuegraph, p)
		ids := map[string]bool{}
		for _, r := range def {
			ids[r.ID] = true
		}
		if !ids[reg.ID] {
			t.Errorf("expected regular %s in default result", reg.ID)
		}
		if ids[wisp.ID] {
			t.Errorf("wisp %s must not appear by default", wisp.ID)
		}
		eph := issuegraphProxiedReadyJSON(t, issuegraph, p, "--include-ephemeral")
		ids2 := map[string]bool{}
		for _, r := range eph {
			ids2[r.ID] = true
		}
		if !ids2[wisp.ID] {
			t.Errorf("expected wisp %s under --include-ephemeral", wisp.ID)
		}
	})

	t.Run("max_rows_claim_policy_and_transaction", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rcmr")
		for i := 0; i < 3; i++ {
			issuegraphProxiedCreate(t, issuegraph, p.dir, fmt.Sprintf("Claim under cap %d", i), "--label", "rcmr-claim")
		}

		out := issuegraphProxiedReadyFail(t, issuegraph, p, "--claim", "--max-rows", "-1")
		if !strings.Contains(out, "must be non-negative") {
			t.Errorf("expected --max-rows usage-error rejection, got: %s", out)
		}
		out = issuegraphProxiedReadyFail(t, issuegraph, p, "--max-rows", "1")
		if !strings.Contains(out, "not supported in proxied-server mode") {
			t.Errorf("expected --max-rows proxied-server rejection for bulk ready, got: %s", out)
		}
		stdout, stderr, err := issuegraphProxiedRunBuffersWithEnv(t, issuegraph, p.dir,
			[]string{"ISSUEGRAPH_MAX_ROWS=1"}, "ready")
		if err == nil {
			t.Fatalf("expected ISSUEGRAPH_MAX_ROWS under proxied bulk ready to fail, but it succeeded:\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
		}
		if out := stdout + stderr; !strings.Contains(out, "not supported in proxied-server mode") {
			t.Errorf("expected ISSUEGRAPH_MAX_ROWS proxied-server rejection for bulk ready, got: %s", out)
		}

		before := issuegraphProxiedReadyJSON(t, issuegraph, p, "--label", "rcmr-claim")
		stdout, stderr, err = issuegraphProxiedRunBuffersWithEnv(t, issuegraph, p.dir,
			[]string{"ISSUEGRAPH_MAX_ROWS=1"}, "ready", "--claim", "--json", "--label", "rcmr-claim")
		if err == nil {
			t.Fatalf("issuegraph ready --claim --json under ISSUEGRAPH_MAX_ROWS=1 should refuse")
		}
		if out := stdout + stderr; !strings.Contains(out, "not supported in proxied-server mode") {
			t.Fatalf("expected cap refusal, got:\n%s", out)
		}
		after := issuegraphProxiedReadyJSON(t, issuegraph, p, "--label", "rcmr-claim")
		if len(after) != len(before) {
			t.Fatalf("claim refusal mutated ready set: before=%d after=%d", len(before), len(after))
		}
	})
}

func TestProxiedServerReadyConcurrent(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	p := newSharedProxiedProject(t, issuegraph, "rxc")
	issuegraphProxiedCreate(t, issuegraph, p.dir, "Concurrent read target")

	const numWorkers = 8
	errs := make([]error, numWorkers)
	var wg sync.WaitGroup
	wg.Add(numWorkers)
	for w := 0; w < numWorkers; w++ {
		go func(worker int) {
			defer wg.Done()
			_, err := issuegraphProxiedRun(t, issuegraph, p.dir, "ready")
			errs[worker] = err
		}(w)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Errorf("worker %d: %v", i, err)
		}
	}
}

func TestProxiedServerReadyClaimConcurrent(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	p := newSharedProxiedProject(t, issuegraph, "rcx")
	issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Atomic claim target", "--label", "atomic")

	const numWorkers = 8
	type result struct {
		stdout string
		stderr string
		err    error
	}
	results := make([]result, numWorkers)
	var wg sync.WaitGroup
	wg.Add(numWorkers)
	for w := 0; w < numWorkers; w++ {
		go func(worker int) {
			defer wg.Done()
			cmd := issuegraphProxiedClaimCmd(issuegraph, p.dir, fmt.Sprintf("worker-%d@test", worker))
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err := cmd.Run()
			results[worker] = result{stdout: stdout.String(), stderr: stderr.String(), err: err}
		}(w)
	}
	wg.Wait()

	winners := 0
	for i, r := range results {
		s := strings.TrimSpace(r.stdout)
		start := strings.Index(s, "[")
		if start < 0 {
			continue
		}
		var claimed []types.IssueWithCounts
		if err := json.Unmarshal([]byte(s[start:]), &claimed); err != nil {
			t.Errorf("worker %d: parse JSON: %v\nstdout: %s\nstderr: %s", i, err, r.stdout, r.stderr)
			continue
		}
		if len(claimed) > 1 {
			t.Errorf("worker %d claimed multiple: %s", i, r.stdout)
			continue
		}
		if len(claimed) == 1 {
			winners++
			if claimed[0].ID != issue.ID {
				t.Errorf("worker %d claimed %s, want %s", i, claimed[0].ID, issue.ID)
			}
		}
	}
	if winners != 1 {
		for i, r := range results {
			t.Logf("worker %d: err=%v\nstdout=%s\nstderr=%s", i, r.err, r.stdout, r.stderr)
		}
		t.Fatalf("expected exactly one winning claim, got %d", winners)
	}
	final := issuegraphProxiedShow(t, issuegraph, p.dir, issue.ID)
	if final.Status != types.StatusInProgress {
		t.Errorf("final Status = %s, want in_progress", final.Status)
	}
	if final.Assignee == "" {
		t.Error("final Assignee empty")
	}
}

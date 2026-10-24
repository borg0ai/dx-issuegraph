//go:build cgo

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
)

func issueRowExists(t *testing.T, db *sql.DB, id string) bool {
	t.Helper()
	var n int
	if err := db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM issues WHERE id = ?", id).Scan(&n); err != nil {
		t.Fatalf("count issue %s: %v", id, err)
	}
	return n > 0
}

func issuegraphProxiedPrune(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"prune"}, args...)
	stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, dir, fullArgs...)
	if err != nil {
		t.Fatalf("issuegraph prune %s failed: %v\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), err, stdout, stderr)
	}
	return stdout
}

func issuegraphProxiedPruneFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"prune"}, args...)
	stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, dir, fullArgs...)
	if err == nil {
		t.Fatalf("expected issuegraph prune %s to fail, got:\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), stdout, stderr)
	}
	return stdout + stderr
}

func TestProxiedServerPrune(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)

	t.Run("requires_filter", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "prf")
		out := issuegraphProxiedPruneFail(t, issuegraph, p.dir)
		if !strings.Contains(out, "requires --older-than or --pattern") {
			t.Errorf("expected requires-filter error, got: %s", out)
		}
	})

	t.Run("dry_run_deletes_nothing", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "pdr")
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Prune me")
		issuegraphProxiedClose(t, issuegraph, p.dir, issue.ID)
		out := issuegraphProxiedPrune(t, issuegraph, p.dir, "--pattern", "*", "--dry-run")
		if !strings.Contains(out, "Would prune") {
			t.Errorf("dry-run output missing 'Would prune': %s", out)
		}
		db := openProxiedDB(t, p)
		if !issueRowExists(t, db, issue.ID) {
			t.Errorf("dry-run must not delete %s", issue.ID)
		}
	})

	t.Run("force_deletes_closed", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "pfd")
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Delete me")
		issuegraphProxiedClose(t, issuegraph, p.dir, issue.ID)
		issuegraphProxiedPrune(t, issuegraph, p.dir, "--pattern", "*", "--force")
		db := openProxiedDB(t, p)
		if issueRowExists(t, db, issue.ID) {
			t.Errorf("closed issue %s should be pruned", issue.ID)
		}
	})

	t.Run("skips_open", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "pso")
		open := issuegraphProxiedCreate(t, issuegraph, p.dir, "Still open")
		closed := issuegraphProxiedCreate(t, issuegraph, p.dir, "Now closed")
		issuegraphProxiedClose(t, issuegraph, p.dir, closed.ID)
		issuegraphProxiedPrune(t, issuegraph, p.dir, "--pattern", "*", "--force")
		db := openProxiedDB(t, p)
		if !issueRowExists(t, db, open.ID) {
			t.Errorf("open issue %s must survive prune", open.ID)
		}
		if issueRowExists(t, db, closed.ID) {
			t.Errorf("closed issue %s should be pruned", closed.ID)
		}
	})

	t.Run("reference_aware_skip", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "pras")
		referenced := issuegraphProxiedCreate(t, issuegraph, p.dir, "Cited decision")
		_ = issuegraphProxiedCreate(t, issuegraph, p.dir, "Open citer",
			"--description", "per "+referenced.ID+" we decided X")
		issuegraphProxiedClose(t, issuegraph, p.dir, referenced.ID)

		out := issuegraphProxiedPrune(t, issuegraph, p.dir, "--pattern", "*", "--force")
		if !strings.Contains(out, "protected by open-issue references") {
			t.Errorf("expected referenced-protection note, got: %s", out)
		}
		db := openProxiedDB(t, p)
		if !issueRowExists(t, db, referenced.ID) {
			t.Errorf("referenced closed issue %s must be protected from prune", referenced.ID)
		}
	})

	t.Run("ignore_references_deletes_referenced", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "pir")
		referenced := issuegraphProxiedCreate(t, issuegraph, p.dir, "Cited decision")
		_ = issuegraphProxiedCreate(t, issuegraph, p.dir, "Open citer",
			"--description", "per "+referenced.ID+" we decided X")
		issuegraphProxiedClose(t, issuegraph, p.dir, referenced.ID)

		issuegraphProxiedPrune(t, issuegraph, p.dir, "--pattern", "*", "--force", "--ignore-references")
		db := openProxiedDB(t, p)
		if issueRowExists(t, db, referenced.ID) {
			t.Errorf("--ignore-references should delete referenced issue %s", referenced.ID)
		}
	})

	t.Run("json_output", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "pjo")
		a := issuegraphProxiedCreate(t, issuegraph, p.dir, "JSON prune A")
		b := issuegraphProxiedCreate(t, issuegraph, p.dir, "JSON prune B")
		issuegraphProxiedClose(t, issuegraph, p.dir, a.ID, b.ID)
		referenced := issuegraphProxiedCreate(t, issuegraph, p.dir, "JSON cited decision")
		issuegraphProxiedCreate(t, issuegraph, p.dir, "JSON open citing issue",
			"--description", "per "+referenced.ID+" we decided X")
		issuegraphProxiedClose(t, issuegraph, p.dir, referenced.ID)
		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "prune", "--json", "--pattern", "*", "--force")
		if err != nil {
			t.Fatalf("issuegraph prune --json failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
		}
		start := strings.Index(stdout, "{")
		if start < 0 {
			t.Fatalf("no JSON object in prune output:\n%s", stdout)
		}
		var stats map[string]any
		if err := json.Unmarshal([]byte(stdout[start:]), &stats); err != nil {
			t.Fatalf("parse prune JSON: %v\nraw: %s", err, stdout[start:])
		}
		if got, ok := stats["pruned_count"].(float64); !ok || int(got) != 2 {
			t.Errorf("pruned_count: got %v, want 2", stats["pruned_count"])
		}
		if got, ok := stats["referenced_skipped"].(float64); !ok || int(got) != 1 {
			t.Errorf("referenced_skipped: got %v, want 1", stats["referenced_skipped"])
		}
		sample, ok := stats["referenced_ids_sample"].([]any)
		if !ok || len(sample) != 1 || sample[0] != referenced.ID {
			t.Errorf("referenced_ids_sample: got %v, want [%s]", stats["referenced_ids_sample"], referenced.ID)
		}
	})

	t.Run("commits_deletion", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "pcd")
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Commit prune")
		issuegraphProxiedClose(t, issuegraph, p.dir, issue.ID)
		db := openProxiedDB(t, p)
		before := readDoltHead(t, db)
		issuegraphProxiedPrune(t, issuegraph, p.dir, "--pattern", "*", "--force")
		if got := readDoltLogCountSince(t, db, before); got < 1 {
			t.Errorf("prune should create a dolt commit, got %d new commits", got)
		}
		// The label is the ROLE's, not this command's: one sweep, one entry,
		// naming the operation and the tier it cleared. It used to read
		// "issuegraph: prune N issue(s)" here and "issuegraph: delete N issue(s)" on the direct
		// route, which named neither the command nor the tier — the two front
		// doors now write the same sentence, and so does an HTTP sweep, where
		// there is no `issuegraph prune` to name.
		msg := readDoltLogTopMessage(t, db)
		if !strings.HasPrefix(msg, "issuegraph: sweep ") || !strings.Contains(msg, "durable") {
			t.Errorf("dolt commit message should name the sweep and its tier, got: %q", msg)
		}
	})
}

//go:build cgo

package main

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

// TestProxiedServerRelate covers the bd-m7zzd port of `issuegraph dep relate` /
// `issuegraph dep unrelate` to proxied-server mode: before it, both verbs died with
// the opaque "cannot resolve issue ID: storage is nil" (id_parser.go) because
// runRelate/runUnrelate had no usesProxiedServer() branch.
func TestProxiedServerRelate(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)

	countRelatesTo := func(t *testing.T, db *sql.DB, issueID, dependsOnID string) int {
		t.Helper()
		var count int
		err := db.QueryRowContext(context.Background(),
			"SELECT COUNT(*) FROM dependencies WHERE issue_id = ? AND COALESCE(depends_on_issue_id, depends_on_wisp_id, depends_on_external) = ? AND type = ?",
			issueID, dependsOnID, "relates-to").Scan(&count)
		if err != nil {
			t.Fatalf("query relates-to %s -> %s: %v", issueID, dependsOnID, err)
		}
		return count
	}

	t.Run("relate_happy_path", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rl1")
		a := issuegraphProxiedCreate(t, issuegraph, p.dir, "Relate A", "--type", "task")
		b := issuegraphProxiedCreate(t, issuegraph, p.dir, "Relate B", "--type", "task")

		db := openProxiedDB(t, p)
		head := readDoltHead(t, db)

		out := issuegraphProxiedDep(t, issuegraph, p.dir, "relate", a.ID, b.ID)
		if !strings.Contains(out, "Linked") {
			t.Errorf("expected 'Linked' output: %s", out)
		}

		// Bidirectional: both directed relates-to edges landed.
		assertProxiedDepExistsWithType(t, db, a.ID, b.ID, "relates-to")
		assertProxiedDepExistsWithType(t, db, b.ID, a.ID, "relates-to")

		// ONE transaction with a real commit message: both edges rode a
		// single Dolt commit.
		if n := readDoltLogCountSince(t, db, head); n != 1 {
			t.Errorf("expected exactly 1 commit for relate, got %d", n)
		}
		if msg := readDoltLogTopMessage(t, db); !strings.Contains(msg, "issuegraph: relate") {
			t.Errorf("expected 'issuegraph: relate' commit message, got %q", msg)
		}
	})

	t.Run("relate_json", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rl2")
		a := issuegraphProxiedCreate(t, issuegraph, p.dir, "JSON A", "--type", "task")
		b := issuegraphProxiedCreate(t, issuegraph, p.dir, "JSON B", "--type", "task")
		m := issuegraphProxiedDepJSON(t, issuegraph, p.dir, "relate", a.ID, b.ID)
		if related, _ := m["related"].(bool); !related {
			t.Errorf("expected related=true in JSON output: %v", m)
		}
		if m["id1"] != a.ID || m["id2"] != b.ID {
			t.Errorf("expected ids %s/%s in JSON output: %v", a.ID, b.ID, m)
		}
	})

	t.Run("unrelate_removes_both_directions", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rl3")
		a := issuegraphProxiedCreate(t, issuegraph, p.dir, "Unrelate A", "--type", "task")
		b := issuegraphProxiedCreate(t, issuegraph, p.dir, "Unrelate B", "--type", "task")
		issuegraphProxiedDep(t, issuegraph, p.dir, "relate", a.ID, b.ID)

		db := openProxiedDB(t, p)
		head := readDoltHead(t, db)

		out := issuegraphProxiedDep(t, issuegraph, p.dir, "unrelate", a.ID, b.ID)
		if !strings.Contains(out, "Unlinked") {
			t.Errorf("expected 'Unlinked' output: %s", out)
		}
		if n := countRelatesTo(t, db, a.ID, b.ID); n != 0 {
			t.Errorf("expected relates-to %s -> %s gone, found %d", a.ID, b.ID, n)
		}
		if n := countRelatesTo(t, db, b.ID, a.ID); n != 0 {
			t.Errorf("expected relates-to %s -> %s gone, found %d", b.ID, a.ID, n)
		}
		if n := readDoltLogCountSince(t, db, head); n != 1 {
			t.Errorf("expected exactly 1 commit for unrelate, got %d", n)
		}
		if msg := readDoltLogTopMessage(t, db); !strings.Contains(msg, "issuegraph: unrelate") {
			t.Errorf("expected 'issuegraph: unrelate' commit message, got %q", msg)
		}
	})

	t.Run("unrelate_idempotent_no_commit", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rl4")
		a := issuegraphProxiedCreate(t, issuegraph, p.dir, "Idem A", "--type", "task")
		b := issuegraphProxiedCreate(t, issuegraph, p.dir, "Idem B", "--type", "task")

		db := openProxiedDB(t, p)
		head := readDoltHead(t, db)

		// No edge exists: still a success (matches the direct route), and a
		// deliberate no-commit no-op.
		out := issuegraphProxiedDep(t, issuegraph, p.dir, "unrelate", a.ID, b.ID)
		if !strings.Contains(out, "Unlinked") {
			t.Errorf("expected 'Unlinked' output: %s", out)
		}
		if n := readDoltLogCountSince(t, db, head); n != 0 {
			t.Errorf("expected no commit for no-op unrelate, got %d", n)
		}
	})

	t.Run("self_relate_refused", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rl5")
		a := issuegraphProxiedCreate(t, issuegraph, p.dir, "Self A", "--type", "task")
		out := issuegraphProxiedDepFail(t, issuegraph, p.dir, "relate", a.ID, a.ID)
		if !strings.Contains(out, "cannot relate an issue to itself") {
			t.Errorf("expected self-relate refusal: %s", out)
		}
	})

	t.Run("missing_issue_clean_error_not_storage_nil", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "rl6")
		a := issuegraphProxiedCreate(t, issuegraph, p.dir, "Lonely A", "--type", "task")

		for _, verb := range []string{"relate", "unrelate"} {
			out := issuegraphProxiedDepFail(t, issuegraph, p.dir, verb, a.ID, "rl6-nonexistent")
			if !strings.Contains(out, "issue not found") {
				t.Errorf("%s: expected 'issue not found' refusal: %s", verb, out)
			}
			// The bd-m7zzd seam itself: the opaque nil-storage death is gone.
			if strings.Contains(out, "storage is nil") {
				t.Errorf("%s: opaque 'storage is nil' error leaked through: %s", verb, out)
			}
		}
	})
}

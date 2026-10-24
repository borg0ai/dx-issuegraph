//go:build cgo

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/borg0ai/dx-issuegraph/internal/types"
)

func issuegraphProxiedClose(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"close"}, args...)
	stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, dir, fullArgs...)
	if err != nil {
		t.Fatalf("issuegraph close %s failed: %v\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), err, stdout, stderr)
	}
	return stdout
}

func issuegraphProxiedCloseFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"close"}, args...)
	stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, dir, fullArgs...)
	if err == nil {
		t.Fatalf("expected issuegraph close %s to fail, got:\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), stdout, stderr)
	}
	return stdout + stderr
}

func issuegraphProxiedCloseJSON(t *testing.T, issuegraph, dir string, args ...string) []*types.Issue {
	t.Helper()
	fullArgs := append([]string{"close", "--json"}, args...)
	stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, dir, fullArgs...)
	if err != nil {
		t.Fatalf("issuegraph close --json %s failed: %v\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), err, stdout, stderr)
	}
	start := strings.Index(stdout, "[")
	if start < 0 {
		t.Fatalf("no JSON array in close output:\n%s", stdout)
	}
	var issues []*types.Issue
	if err := json.Unmarshal([]byte(stdout[start:]), &issues); err != nil {
		t.Fatalf("parse close JSON: %v\nraw: %s", err, stdout[start:])
	}
	return issues
}

func issuegraphProxiedCloseJSONEnvelope(t *testing.T, issuegraph, dir string, args ...string) map[string]json.RawMessage {
	t.Helper()
	fullArgs := append([]string{"close", "--json"}, args...)
	stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, dir, fullArgs...)
	if err != nil {
		t.Fatalf("issuegraph close --json %s failed: %v\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), err, stdout, stderr)
	}
	start := strings.Index(stdout, "{")
	if start < 0 {
		t.Fatalf("no JSON object in close output:\n%s", stdout)
	}
	var env map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stdout[start:]), &env); err != nil {
		t.Fatalf("parse close JSON envelope: %v\nraw: %s", err, stdout[start:])
	}
	return env
}

func readClosedBySession(t *testing.T, db *sql.DB, id string) string {
	t.Helper()
	var got sql.NullString
	err := db.QueryRowContext(context.Background(),
		"SELECT closed_by_session FROM issues WHERE id = ?", id).Scan(&got)
	if err == sql.ErrNoRows {
		if err := db.QueryRowContext(context.Background(),
			"SELECT closed_by_session FROM wisps WHERE id = ?", id).Scan(&got); err != nil {
			t.Fatalf("read closed_by_session for %s: %v", id, err)
		}
	} else if err != nil {
		t.Fatalf("read closed_by_session for %s: %v", id, err)
	}
	if !got.Valid {
		return ""
	}
	return got.String
}

func readCloseReason(t *testing.T, db *sql.DB, id string) string {
	t.Helper()
	var got sql.NullString
	err := db.QueryRowContext(context.Background(),
		"SELECT close_reason FROM issues WHERE id = ?", id).Scan(&got)
	if err == sql.ErrNoRows {
		if err := db.QueryRowContext(context.Background(),
			"SELECT close_reason FROM wisps WHERE id = ?", id).Scan(&got); err != nil {
			t.Fatalf("read close_reason for %s: %v", id, err)
		}
	} else if err != nil {
		t.Fatalf("read close_reason for %s: %v", id, err)
	}
	if !got.Valid {
		return ""
	}
	return got.String
}

func readStatus(t *testing.T, db *sql.DB, id string) types.Status {
	t.Helper()
	var got string
	err := db.QueryRowContext(context.Background(),
		"SELECT status FROM issues WHERE id = ?", id).Scan(&got)
	if err == sql.ErrNoRows {
		if err := db.QueryRowContext(context.Background(),
			"SELECT status FROM wisps WHERE id = ?", id).Scan(&got); err != nil {
			t.Fatalf("read status for %s: %v", id, err)
		}
	} else if err != nil {
		t.Fatalf("read status for %s: %v", id, err)
	}
	return types.Status(got)
}

func readAssignee(t *testing.T, db *sql.DB, id string) string {
	t.Helper()
	var got sql.NullString
	err := db.QueryRowContext(context.Background(),
		"SELECT assignee FROM issues WHERE id = ?", id).Scan(&got)
	if err == sql.ErrNoRows {
		if err := db.QueryRowContext(context.Background(),
			"SELECT assignee FROM wisps WHERE id = ?", id).Scan(&got); err != nil {
			t.Fatalf("read assignee for %s: %v", id, err)
		}
	} else if err != nil {
		t.Fatalf("read assignee for %s: %v", id, err)
	}
	if !got.Valid {
		return ""
	}
	return got.String
}

func readDoltHead(t *testing.T, db *sql.DB) string {
	t.Helper()
	var h string
	if err := db.QueryRowContext(context.Background(), "SELECT HASHOF('HEAD')").Scan(&h); err != nil {
		t.Fatalf("read HEAD: %v", err)
	}
	return h
}

func readDoltLogTopMessage(t *testing.T, db *sql.DB) string {
	t.Helper()
	var msg string
	if err := db.QueryRowContext(context.Background(),
		"SELECT message FROM dolt_log ORDER BY date DESC LIMIT 1").Scan(&msg); err != nil {
		t.Fatalf("read latest dolt_log message: %v", err)
	}
	return msg
}

func readDoltLogCountSince(t *testing.T, db *sql.DB, sinceHash string) int {
	t.Helper()
	rows, err := db.QueryContext(context.Background(),
		"SELECT commit_hash FROM dolt_log ORDER BY date DESC")
	if err != nil {
		t.Fatalf("read dolt_log: %v", err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			t.Fatalf("scan dolt_log: %v", err)
		}
		if h == sinceHash {
			return n
		}
		n++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iter dolt_log: %v", err)
	}
	return n
}

func TestProxiedServerClose(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)

	t.Run("basic_close", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cb")
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Close me")
		issuegraphProxiedClose(t, issuegraph, p.dir, issue.ID)
		got := issuegraphProxiedShow(t, issuegraph, p.dir, issue.ID)
		if got.Status != types.StatusClosed {
			t.Errorf("status: got %q, want closed", got.Status)
		}
		if got.ClosedAt == nil {
			t.Error("closed_at should be set")
		}
	})

	t.Run("close_default_reason", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cdr")
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Default reason")
		issuegraphProxiedClose(t, issuegraph, p.dir, issue.ID)
		db := openProxiedDB(t, p)
		if got := readCloseReason(t, db, issue.ID); got != "Closed" {
			t.Errorf("close_reason: got %q, want %q", got, "Closed")
		}
	})

	t.Run("close_with_reason", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cwr")
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Reason test")
		issuegraphProxiedClose(t, issuegraph, p.dir, issue.ID, "--reason", "done")
		db := openProxiedDB(t, p)
		if got := readCloseReason(t, db, issue.ID); got != "done" {
			t.Errorf("close_reason: got %q, want %q", got, "done")
		}
	})

	t.Run("close_with_reason_short", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cwrs")
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Short reason")
		issuegraphProxiedClose(t, issuegraph, p.dir, issue.ID, "-r", "fixed")
		db := openProxiedDB(t, p)
		if got := readCloseReason(t, db, issue.ID); got != "fixed" {
			t.Errorf("close_reason: got %q, want %q", got, "fixed")
		}
	})

	t.Run("close_with_message_alias", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cwma")
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Message alias")
		issuegraphProxiedClose(t, issuegraph, p.dir, issue.ID, "-m", "via message")
		db := openProxiedDB(t, p)
		if got := readCloseReason(t, db, issue.ID); got != "via message" {
			t.Errorf("close_reason: got %q, want %q", got, "via message")
		}
	})

	t.Run("close_with_resolution_alias", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cwra")
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Resolution alias")
		issuegraphProxiedClose(t, issuegraph, p.dir, issue.ID, "--resolution", "wontfix")
		db := openProxiedDB(t, p)
		if got := readCloseReason(t, db, issue.ID); got != "wontfix" {
			t.Errorf("close_reason: got %q, want %q", got, "wontfix")
		}
	})

	t.Run("close_with_comment_alias", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cwca")
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Comment alias")
		issuegraphProxiedClose(t, issuegraph, p.dir, issue.ID, "--comment", "duplicate")
		db := openProxiedDB(t, p)
		if got := readCloseReason(t, db, issue.ID); got != "duplicate" {
			t.Errorf("close_reason: got %q, want %q", got, "duplicate")
		}
	})

	t.Run("close_multiple_ids", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cmi")
		a := issuegraphProxiedCreate(t, issuegraph, p.dir, "Multi A")
		b := issuegraphProxiedCreate(t, issuegraph, p.dir, "Multi B")
		issuegraphProxiedClose(t, issuegraph, p.dir, a.ID, b.ID)
		db := openProxiedDB(t, p)
		if got := readStatus(t, db, a.ID); got != types.StatusClosed {
			t.Errorf("a status: got %q, want closed", got)
		}
		if got := readStatus(t, db, b.ID); got != types.StatusClosed {
			t.Errorf("b status: got %q, want closed", got)
		}
	})

	t.Run("close_multiple_ids_with_per_id_reasons", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cmpr")
		a := issuegraphProxiedCreate(t, issuegraph, p.dir, "Multi reason A")
		b := issuegraphProxiedCreate(t, issuegraph, p.dir, "Multi reason B")
		issuegraphProxiedClose(t, issuegraph, p.dir, a.ID, "--reason", "fixed A", b.ID, "--reason", "fixed B")
		db := openProxiedDB(t, p)
		if got := readCloseReason(t, db, a.ID); got != "fixed A" {
			t.Errorf("a reason: got %q, want %q", got, "fixed A")
		}
		if got := readCloseReason(t, db, b.ID); got != "fixed B" {
			t.Errorf("b reason: got %q, want %q", got, "fixed B")
		}
	})

	t.Run("close_already_closed", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cac")
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Double close")
		issuegraphProxiedClose(t, issuegraph, p.dir, issue.ID, "--reason", "first")
		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "close", issue.ID, "--reason", "second")
		_ = stdout
		_ = stderr
		_ = err
		db := openProxiedDB(t, p)
		if got := readCloseReason(t, db, issue.ID); got != "first" {
			t.Errorf("re-close must not overwrite reason: got %q, want %q", got, "first")
		}
	})

	t.Run("close_nonexistent_id", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cni")
		out := issuegraphProxiedCloseFail(t, issuegraph, p.dir, "cni-does-not-exist")
		if !strings.Contains(out, "not found") {
			t.Errorf("expected 'not found' error, got: %s", out)
		}
	})

	t.Run("close_blocked_refuses_without_force", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cbr")
		blocker := issuegraphProxiedCreate(t, issuegraph, p.dir, "Blocker")
		blocked := issuegraphProxiedCreate(t, issuegraph, p.dir, "Blocked", "--deps", "depends-on:"+blocker.ID)
		out := issuegraphProxiedCloseFail(t, issuegraph, p.dir, blocked.ID)
		if !strings.Contains(out, "blocked by") {
			t.Errorf("expected blocker error, got: %s", out)
		}
		db := openProxiedDB(t, p)
		if got := readStatus(t, db, blocked.ID); got == types.StatusClosed {
			t.Error("blocked issue should remain open without --force")
		}
	})

	t.Run("close_blocked_with_force", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cbf")
		blocker := issuegraphProxiedCreate(t, issuegraph, p.dir, "Blocker force")
		blocked := issuegraphProxiedCreate(t, issuegraph, p.dir, "Blocked force", "--deps", "depends-on:"+blocker.ID)
		issuegraphProxiedClose(t, issuegraph, p.dir, blocked.ID, "--force")
		db := openProxiedDB(t, p)
		if got := readStatus(t, db, blocked.ID); got != types.StatusClosed {
			t.Errorf("status with --force: got %q, want closed", got)
		}
	})

	t.Run("close_pinned_refuses_without_force", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cpr")
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Pinned")
		issuegraphProxiedUpdateOne(t, issuegraph, p.dir, issue.ID, "-s", "pinned")
		issuegraphProxiedCloseFail(t, issuegraph, p.dir, issue.ID)
		db := openProxiedDB(t, p)
		if got := readStatus(t, db, issue.ID); got == types.StatusClosed {
			t.Error("pinned issue should remain pinned without --force")
		}
	})

	t.Run("close_pinned_with_force", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cpf")
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Pinned force")
		issuegraphProxiedUpdateOne(t, issuegraph, p.dir, issue.ID, "-s", "pinned")
		issuegraphProxiedClose(t, issuegraph, p.dir, issue.ID, "--force")
		db := openProxiedDB(t, p)
		if got := readStatus(t, db, issue.ID); got != types.StatusClosed {
			t.Errorf("status: got %q, want closed", got)
		}
	})

	// ga-ktn9pe.4.8: twin of TestEmbeddedClose/close_boolean_pinned_reclose_is_idempotent.
	// Both close paths must agree — a fix on one only is the divergence class #5217
	// just closed.
	t.Run("close_boolean_pinned_reclose_is_idempotent", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cbpr")
		plan := `{"nodes": [{"key": "p", "title": "Boolean pinned", "type": "task", "pinned": true}]}`
		planFile := filepath.Join(p.dir, "boolean-pinned-plan.json")
		if err := os.WriteFile(planFile, []byte(plan), 0644); err != nil {
			t.Fatal(err)
		}
		out, err := issuegraphProxiedRun(t, issuegraph, p.dir, "create", "--graph", planFile, "--json")
		if err != nil {
			t.Fatalf("issuegraph create --graph failed: %v\n%s", err, out)
		}
		var created GraphApplyResult
		if err := json.Unmarshal(out, &created); err != nil {
			t.Fatalf("parse graph result: %v\nstdout:\n%s", err, out)
		}
		id := created.IDs["p"]
		if id == "" {
			t.Fatalf("expected an ID for key p, got %#v", created.IDs)
		}
		if seeded := issuegraphProxiedShow(t, issuegraph, p.dir, id); !seeded.Pinned {
			t.Fatalf("precondition: expected pinned=true on the seeded issue, got %+v", seeded)
		}

		issuegraphProxiedClose(t, issuegraph, p.dir, id, "--force")
		// No --force on the retry. Pre-fix this exited nonzero with the pinned
		// refusal, so issuegraphProxiedClose's t.Fatalf is the red assertion.
		issuegraphProxiedClose(t, issuegraph, p.dir, id)

		got := issuegraphProxiedShow(t, issuegraph, p.dir, id)
		if got.Status != types.StatusClosed {
			t.Errorf("status: got %q, want closed", got.Status)
		}
		if !got.Pinned {
			t.Error("expected the pin to survive the close: it is what protects the row from issuegraph gc/purge/cleanup")
		}
	})

	t.Run("close_epic_open_children_refuses", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "ceor")
		epic := issuegraphProxiedCreate(t, issuegraph, p.dir, "Epic", "-t", "epic")
		_ = issuegraphProxiedCreate(t, issuegraph, p.dir, "Child", "--parent", epic.ID)
		out := issuegraphProxiedCloseFail(t, issuegraph, p.dir, epic.ID)
		if !strings.Contains(out, "open child") {
			t.Errorf("expected 'open child' error, got: %s", out)
		}
	})

	t.Run("close_epic_open_children_force", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "ceof")
		epic := issuegraphProxiedCreate(t, issuegraph, p.dir, "Epic force", "-t", "epic")
		_ = issuegraphProxiedCreate(t, issuegraph, p.dir, "Child force", "--parent", epic.ID)
		issuegraphProxiedClose(t, issuegraph, p.dir, epic.ID, "--force")
		db := openProxiedDB(t, p)
		if got := readStatus(t, db, epic.ID); got != types.StatusClosed {
			t.Errorf("status: got %q, want closed", got)
		}
	})

	t.Run("close_last_child_keeps_regular_epic_open", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "clce")
		epic := issuegraphProxiedCreate(t, issuegraph, p.dir, "Regular epic", "-t", "epic")
		child := issuegraphProxiedCreate(t, issuegraph, p.dir, "Last child", "--parent", epic.ID)
		issuegraphProxiedClose(t, issuegraph, p.dir, child.ID)
		db := openProxiedDB(t, p)
		if got := readStatus(t, db, epic.ID); got != types.StatusOpen {
			t.Errorf("regular epic should stay open after last child closes, got %q", got)
		}
	})

	t.Run("close_unblocks_dependent", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cud")
		blocker := issuegraphProxiedCreate(t, issuegraph, p.dir, "Unblock blocker")
		blocked := issuegraphProxiedCreate(t, issuegraph, p.dir, "Unblock blocked", "--deps", "depends-on:"+blocker.ID)
		db := openProxiedDB(t, p)
		if !readIsBlocked(t, db, blocked.ID) {
			t.Fatal("dependent should be blocked before blocker closes")
		}
		issuegraphProxiedClose(t, issuegraph, p.dir, blocker.ID)
		if readIsBlocked(t, db, blocked.ID) {
			t.Error("dependent should be unblocked after blocker closes")
		}
	})

	t.Run("close_suggest_next", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "csn")
		blocker := issuegraphProxiedCreate(t, issuegraph, p.dir, "Suggest blocker")
		blocked := issuegraphProxiedCreate(t, issuegraph, p.dir, "Suggest blocked", "--deps", "depends-on:"+blocker.ID)
		out := issuegraphProxiedClose(t, issuegraph, p.dir, blocker.ID, "--suggest-next")
		if !strings.Contains(out, blocked.ID) {
			t.Errorf("suggest-next output missing unblocked id %s:\n%s", blocked.ID, out)
		}
	})

}

func TestProxiedServerClose2(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)

	t.Run("close_suggest_next_json", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "csnj")
		blocker := issuegraphProxiedCreate(t, issuegraph, p.dir, "Suggest JSON blocker")
		blocked := issuegraphProxiedCreate(t, issuegraph, p.dir, "Suggest JSON blocked", "--deps", "depends-on:"+blocker.ID)
		env := issuegraphProxiedCloseJSONEnvelope(t, issuegraph, p.dir, blocker.ID, "--suggest-next")
		raw, ok := env["unblocked"]
		if !ok {
			t.Fatalf("envelope missing 'unblocked' key: %v", env)
		}
		var unblocked []*types.Issue
		if err := json.Unmarshal(raw, &unblocked); err != nil {
			t.Fatalf("parse unblocked: %v\n%s", err, string(raw))
		}
		if len(unblocked) != 1 || unblocked[0].ID != blocked.ID {
			t.Errorf("unblocked: got %+v, want [%s]", unblocked, blocked.ID)
		}
	})

	t.Run("close_claim_next", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "ccn")
		toClose := issuegraphProxiedCreate(t, issuegraph, p.dir, "Claim next close")
		nextIssue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Claim next target")
		issuegraphProxiedClose(t, issuegraph, p.dir, toClose.ID, "--claim-next")
		db := openProxiedDB(t, p)
		if got := readStatus(t, db, nextIssue.ID); got != types.StatusInProgress {
			t.Errorf("next issue status: got %q, want in_progress", got)
		}
		if got := readAssignee(t, db, nextIssue.ID); got == "" {
			t.Error("next issue assignee should be set after --claim-next")
		}
	})

	t.Run("close_claim_next_no_ready", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "ccnr")
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Only issue")
		out := issuegraphProxiedClose(t, issuegraph, p.dir, issue.ID, "--claim-next")
		if !strings.Contains(out, "No ready issues") {
			t.Errorf("expected 'No ready issues', got: %s", out)
		}
	})

	t.Run("close_claim_next_json", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "ccnj")
		toClose := issuegraphProxiedCreate(t, issuegraph, p.dir, "Claim JSON close")
		_ = issuegraphProxiedCreate(t, issuegraph, p.dir, "Claim JSON target")
		env := issuegraphProxiedCloseJSONEnvelope(t, issuegraph, p.dir, toClose.ID, "--claim-next")
		if _, ok := env["closed"]; !ok {
			t.Errorf("envelope missing 'closed' key: %v", env)
		}
		if _, ok := env["claimed"]; !ok {
			t.Errorf("envelope missing 'claimed' key: %v", env)
		}
	})

	t.Run("close_with_session", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cws")
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Session flag")
		issuegraphProxiedClose(t, issuegraph, p.dir, issue.ID, "--session", "sess-456")
		db := openProxiedDB(t, p)
		if got := readClosedBySession(t, db, issue.ID); got != "sess-456" {
			t.Errorf("closed_by_session: got %q, want %q", got, "sess-456")
		}
	})

	t.Run("close_session_from_env", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cse")
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Session env")
		if _, stderr, err := issuegraphProxiedRunEnv(t, issuegraph, p.dir, []string{"CLAUDE_SESSION_ID=sess-env"}, "close", issue.ID); err != nil {
			t.Fatalf("close failed: %v\n%s", err, stderr)
		}
		db := openProxiedDB(t, p)
		if got := readClosedBySession(t, db, issue.ID); got != "sess-env" {
			t.Errorf("closed_by_session: got %q, want %q", got, "sess-env")
		}
	})

	t.Run("close_json_output", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cjo")
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "JSON close")
		issues := issuegraphProxiedCloseJSON(t, issuegraph, p.dir, issue.ID)
		if len(issues) != 1 || issues[0].ID != issue.ID {
			t.Errorf("close JSON: got %+v, want [%s]", issues, issue.ID)
		}
		if issues[0].Status != types.StatusClosed {
			t.Errorf("returned issue status: got %q, want closed", issues[0].Status)
		}
	})

	t.Run("done_alias", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "da")
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Done alias")
		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "done", issue.ID)
		if err != nil {
			t.Fatalf("issuegraph done failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
		}
		db := openProxiedDB(t, p)
		if got := readStatus(t, db, issue.ID); got != types.StatusClosed {
			t.Errorf("status via done alias: got %q, want closed", got)
		}
	})

	t.Run("done_positional_reason", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "dpr")
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Done reason")
		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "done", issue.ID, "the reason")
		if err != nil {
			t.Fatalf("issuegraph done with reason failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
		}
		db := openProxiedDB(t, p)
		if got := readCloseReason(t, db, issue.ID); got != "the reason" {
			t.Errorf("close_reason: got %q, want %q", got, "the reason")
		}
	})

	t.Run("close_continue_multiple_ids_fails", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "ccmi")
		a := issuegraphProxiedCreate(t, issuegraph, p.dir, "Continue multi A")
		b := issuegraphProxiedCreate(t, issuegraph, p.dir, "Continue multi B")
		out := issuegraphProxiedCloseFail(t, issuegraph, p.dir, a.ID, b.ID, "--continue")
		if !strings.Contains(out, "single issue") {
			t.Errorf("expected single-issue error, got: %s", out)
		}
	})

	t.Run("close_suggest_next_multiple_ids_fails", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "csmi")
		a := issuegraphProxiedCreate(t, issuegraph, p.dir, "Suggest multi A")
		b := issuegraphProxiedCreate(t, issuegraph, p.dir, "Suggest multi B")
		out := issuegraphProxiedCloseFail(t, issuegraph, p.dir, a.ID, b.ID, "--suggest-next")
		if !strings.Contains(out, "single issue") {
			t.Errorf("expected single-issue error, got: %s", out)
		}
	})

	t.Run("single_transaction_dolt_commit", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "stdc")
		a := issuegraphProxiedCreate(t, issuegraph, p.dir, "Tx A")
		b := issuegraphProxiedCreate(t, issuegraph, p.dir, "Tx B")
		c := issuegraphProxiedCreate(t, issuegraph, p.dir, "Tx C")
		db := openProxiedDB(t, p)
		before := readDoltHead(t, db)
		issuegraphProxiedClose(t, issuegraph, p.dir, a.ID, b.ID, c.ID)
		count := readDoltLogCountSince(t, db, before)
		if count != 1 {
			t.Errorf("expected exactly 1 new dolt commit for batch close, got %d", count)
		}
		msg := readDoltLogTopMessage(t, db)
		for _, id := range []string{a.ID, b.ID, c.ID} {
			if !strings.Contains(msg, id) {
				t.Errorf("dolt commit message %q should contain id %s", msg, id)
			}
		}
		if !strings.HasPrefix(msg, "issuegraph: close ") {
			t.Errorf("dolt commit message should begin with 'issuegraph: close ', got: %q", msg)
		}
	})

	t.Run("no_ids_errors", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "nie")
		out := issuegraphProxiedCloseFail(t, issuegraph, p.dir)
		if !strings.Contains(out, "no issue ID provided") {
			t.Errorf("expected 'no issue ID provided', got: %s", out)
		}
	})

	t.Run("last_touched_not_supported", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "ltns")
		_ = issuegraphProxiedCreate(t, issuegraph, p.dir, "Recent create")
		out := issuegraphProxiedCloseFail(t, issuegraph, p.dir)
		if !strings.Contains(out, "no issue ID provided") {
			t.Errorf("proxied mode must not fall back to last-touched; got: %s", out)
		}
	})

	t.Run("close_wisp_issue", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cwi")
		wisp := issuegraphProxiedCreate(t, issuegraph, p.dir, "Wisp close", "--ephemeral")
		issuegraphProxiedClose(t, issuegraph, p.dir, wisp.ID)
		db := openProxiedDB(t, p)
		var status string
		if err := db.QueryRowContext(context.Background(),
			"SELECT status FROM wisps WHERE id = ?", wisp.ID).Scan(&status); err != nil {
			t.Fatalf("read wisp status: %v", err)
		}
		if types.Status(status) != types.StatusClosed {
			t.Errorf("wisp status: got %q, want closed", status)
		}
	})

	t.Run("close_wisp_epic_open_children", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cweoc")
		wispEpic := issuegraphProxiedCreate(t, issuegraph, p.dir, "Wisp epic", "-t", "epic", "--ephemeral")
		_ = issuegraphProxiedCreate(t, issuegraph, p.dir, "Wisp child", "--ephemeral", "--parent", wispEpic.ID)
		out := issuegraphProxiedCloseFail(t, issuegraph, p.dir, wispEpic.ID)
		if !strings.Contains(out, "open child") {
			t.Errorf("expected 'open child' error for wisp epic, got: %s", out)
		}
	})

	t.Run("close_wisp_blocked_refuses", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cwbr")
		blocker := issuegraphProxiedCreate(t, issuegraph, p.dir, "Blocker for wisp")
		wisp := issuegraphProxiedCreate(t, issuegraph, p.dir, "Blocked wisp", "--ephemeral", "--deps", "depends-on:"+blocker.ID)
		out := issuegraphProxiedCloseFail(t, issuegraph, p.dir, wisp.ID)
		if !strings.Contains(out, "blocked by") {
			t.Errorf("expected wisp blocker error, got: %s", out)
		}
	})

	t.Run("continue_advances_to_next_ready_step", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cantrs")
		root := issuegraphProxiedCreate(t, issuegraph, p.dir, "Molecule root", "-t", "epic", "--labels", "template")
		step1 := issuegraphProxiedCreate(t, issuegraph, p.dir, "Step 1", "--parent", root.ID)
		step2 := issuegraphProxiedCreate(t, issuegraph, p.dir, "Step 2", "--parent", root.ID, "--deps", "depends-on:"+step1.ID)
		_ = issuegraphProxiedCreate(t, issuegraph, p.dir, "Step 3", "--parent", root.ID, "--deps", "depends-on:"+step2.ID)
		issuegraphProxiedClose(t, issuegraph, p.dir, step1.ID, "--continue")
		db := openProxiedDB(t, p)
		if got := readStatus(t, db, step2.ID); got != types.StatusInProgress {
			t.Errorf("step2 status after --continue: got %q, want in_progress", got)
		}
		if readAssignee(t, db, step2.ID) == "" {
			t.Error("step2 assignee should be set after --continue auto-claim")
		}
	})

	t.Run("continue_no_auto_does_not_claim", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "cnac")
		root := issuegraphProxiedCreate(t, issuegraph, p.dir, "Molecule root", "-t", "epic", "--labels", "template")
		step1 := issuegraphProxiedCreate(t, issuegraph, p.dir, "Step 1", "--parent", root.ID)
		step2 := issuegraphProxiedCreate(t, issuegraph, p.dir, "Step 2", "--parent", root.ID, "--deps", "depends-on:"+step1.ID)
		issuegraphProxiedClose(t, issuegraph, p.dir, step1.ID, "--continue", "--no-auto")
		db := openProxiedDB(t, p)
		if got := readStatus(t, db, step2.ID); got != types.StatusOpen {
			t.Errorf("step2 status with --no-auto: got %q, want open", got)
		}
		if readAssignee(t, db, step2.ID) != "" {
			t.Error("step2 assignee should NOT be set with --no-auto")
		}
	})

	t.Run("auto_close_completed_molecule", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "accm")
		root := issuegraphProxiedCreate(t, issuegraph, p.dir, "Molecule root", "-t", "epic", "--labels", "template")
		s1 := issuegraphProxiedCreate(t, issuegraph, p.dir, "Step 1", "--parent", root.ID)
		s2 := issuegraphProxiedCreate(t, issuegraph, p.dir, "Step 2", "--parent", root.ID)
		issuegraphProxiedClose(t, issuegraph, p.dir, s1.ID, s2.ID)
		db := openProxiedDB(t, p)
		if got := readStatus(t, db, root.ID); got != types.StatusClosed {
			t.Errorf("template-labeled molecule root should auto-close after all steps complete, got %q", got)
		}
		if got := readCloseReason(t, db, root.ID); got != "all steps complete" {
			t.Errorf("auto-close reason: got %q, want %q", got, "all steps complete")
		}
	})

	t.Run("hooks_fire_on_close", func(t *testing.T) {
		t.Parallel()
		marker := filepath.Join(t.TempDir(), "on_close_marker")
		script := "#!/bin/sh\nprintf '%s\\n' \"$1\" > " + shellQuote(marker) + "\n"
		if runtime.GOOS == "windows" {
			t.Skip("hook script form is POSIX shell")
		}
		p := newSharedProxiedProjectWithHooks(t, issuegraph, "hfc", map[string]string{"on_close": script})
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Hook fire")
		issuegraphProxiedClose(t, issuegraph, p.dir, issue.ID)
		// Polled, not read once: hooks fire fire-and-forget off the write
		// plumbing now, so the marker appears shortly AFTER the command exits
		// rather than during it.
		data, err := waitForMarker(marker, 5*time.Second)
		if err != nil {
			t.Fatalf("on_close hook did not fire within timeout: %v", err)
		}
		if !strings.Contains(data, issue.ID) {
			t.Errorf("hook marker missing issue ID; got: %q", data)
		}
	})
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func TestProxiedServerCloseConcurrent(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)

	p := newSharedProxiedProject(t, issuegraph, "cxc")

	const (
		numWorkers      = 10
		issuesPerWorker = 5
	)

	type ws struct {
		closed int
		errs   []string
	}
	results := make([]ws, numWorkers)
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for w := 0; w < numWorkers; w++ {
		w := w
		go func() {
			defer wg.Done()
			r := &results[w]
			for i := 0; i < issuesPerWorker; i++ {
				title := fmt.Sprintf("worker-%d-issue-%d", w, i)
				cmd := exec.Command(issuegraph, "create", "--json", title)
				cmd.Dir = p.dir
				cmd.Env = issuegraphProxiedEnv(p.dir)
				var stdout, stderr bytes.Buffer
				cmd.Stdout = &stdout
				cmd.Stderr = &stderr
				if err := cmd.Run(); err != nil {
					r.errs = append(r.errs, fmt.Sprintf("create %s: %v\n%s", title, err, stderr.String()))
					continue
				}
				out := stdout.Bytes()
				start := bytes.Index(out, []byte("{"))
				if start < 0 {
					r.errs = append(r.errs, "no JSON in create output")
					continue
				}
				var issue types.Issue
				if err := json.Unmarshal(out[start:], &issue); err != nil {
					r.errs = append(r.errs, fmt.Sprintf("parse create JSON: %v", err))
					continue
				}

				closeCmd := exec.Command(issuegraph, "close", issue.ID)
				closeCmd.Dir = p.dir
				closeCmd.Env = issuegraphProxiedEnv(p.dir)
				var cstdout, cstderr bytes.Buffer
				closeCmd.Stdout = &cstdout
				closeCmd.Stderr = &cstderr
				if err := closeCmd.Run(); err != nil {
					r.errs = append(r.errs, fmt.Sprintf("close %s: %v\n%s", issue.ID, err, cstderr.String()))
					continue
				}
				r.closed++
			}
		}()
	}
	wg.Wait()

	totalClosed := 0
	for w, r := range results {
		totalClosed += r.closed
		for _, e := range r.errs {
			t.Errorf("worker %d: %s", w, e)
		}
	}
	want := numWorkers * issuesPerWorker
	if totalClosed != want {
		t.Errorf("closed count: got %d, want %d", totalClosed, want)
	}

	db := openProxiedDB(t, p)
	var openCount int
	if err := db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM issues WHERE status != 'closed'").Scan(&openCount); err != nil {
		t.Fatalf("query open count: %v", err)
	}
	if openCount != 0 {
		t.Errorf("open issues remain after concurrent close: %d", openCount)
	}
}

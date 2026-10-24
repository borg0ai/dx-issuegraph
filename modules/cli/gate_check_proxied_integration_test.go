//go:build cgo

package main

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/borg0ai/dx-issuegraph/internal/types"
)

func seedGateAwait(t *testing.T, db *sql.DB, id, awaitType, awaitID string, createdAt time.Time, timeout time.Duration) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(),
		"UPDATE issues SET await_type=?, await_id=?, timeout_ns=?, created_at=? WHERE id=?",
		awaitType, awaitID, int64(timeout), createdAt.UTC(), id); err != nil {
		t.Fatalf("seed await columns for %s: %v", id, err)
	}
}

func TestProxiedServerGateCheck(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)

	const expiredTimeout = time.Hour
	longAgo := func() time.Time { return time.Now().UTC().Add(-48 * time.Hour) }
	longFuture := time.Hour * 1000

	t.Run("no_gates", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "gcn")
		out, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "gate", "check")
		if err != nil {
			t.Fatalf("gate check failed: %v\nstderr:\n%s", err, stderr)
		}
		if !strings.Contains(out, "No open gates") {
			t.Errorf("expected 'No open gates' message, got:\n%s", out)
		}
	})

	t.Run("timer_expired_resolves", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "gce")
		gate := issuegraphProxiedCreate(t, issuegraph, p.dir, "Expired timer gate", "--type", "gate")
		db := openProxiedDB(t, p)
		seedGateAwait(t, db, gate.ID, "timer", "", longAgo(), expiredTimeout)

		out, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "gate", "check", "--type", "timer")
		if err != nil {
			t.Fatalf("gate check failed: %v\nstderr:\n%s", err, stderr)
		}
		if got := readStatus(t, db, gate.ID); got != types.StatusClosed {
			t.Errorf("expired timer gate should be closed, got %q", got)
		}
		if !strings.Contains(out, "Checked 1 gates: 1 resolved") {
			t.Errorf("expected summary with 1 resolved, got:\n%s", out)
		}
	})

	t.Run("timer_pending_stays_open", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "gcp")
		gate := issuegraphProxiedCreate(t, issuegraph, p.dir, "Pending timer gate", "--type", "gate")
		db := openProxiedDB(t, p)
		seedGateAwait(t, db, gate.ID, "timer", "", time.Now().UTC(), longFuture)

		out, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "gate", "check", "--type", "timer")
		if err != nil {
			t.Fatalf("gate check failed: %v\nstderr:\n%s", err, stderr)
		}
		if got := readStatus(t, db, gate.ID); got == types.StatusClosed {
			t.Error("pending timer gate should stay open")
		}
		if !strings.Contains(out, "0 resolved") {
			t.Errorf("expected 0 resolved for pending timer, got:\n%s", out)
		}
	})

	t.Run("dry_run_does_not_close", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "gcd")
		gate := issuegraphProxiedCreate(t, issuegraph, p.dir, "Dry run timer gate", "--type", "gate")
		db := openProxiedDB(t, p)
		seedGateAwait(t, db, gate.ID, "timer", "", longAgo(), expiredTimeout)

		out, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "gate", "check", "--type", "timer", "--dry-run")
		if err != nil {
			t.Fatalf("gate check --dry-run failed: %v\nstderr:\n%s", err, stderr)
		}
		if got := readStatus(t, db, gate.ID); got == types.StatusClosed {
			t.Error("dry-run must not close the gate")
		}
		if !strings.Contains(out, "would resolve") {
			t.Errorf("expected 'would resolve' in dry-run output, got:\n%s", out)
		}
	})

	t.Run("issue_gate_resolves_when_target_closes", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "gcb")
		target := issuegraphProxiedCreate(t, issuegraph, p.dir, "Issue gate target")
		gate := issuegraphProxiedCreate(t, issuegraph, p.dir, "Issue gate", "--type", "gate")
		db := openProxiedDB(t, p)
		seedGateAwait(t, db, gate.ID, "issue", target.ID, time.Now().UTC(), 0)

		if _, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "gate", "check", "--type", "issue"); err != nil {
			t.Fatalf("gate check (target open) failed: %v\nstderr:\n%s", err, stderr)
		}
		if got := readStatus(t, db, gate.ID); got == types.StatusClosed {
			t.Error("issue gate should stay pending while target is open")
		}

		issuegraphProxiedClose(t, issuegraph, p.dir, target.ID)
		out, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "gate", "check", "--type", "issue")
		if err != nil {
			t.Fatalf("gate check (target closed) failed: %v\nstderr:\n%s", err, stderr)
		}
		if got := readStatus(t, db, gate.ID); got != types.StatusClosed {
			t.Errorf("issue gate should resolve after target closes, got %q", got)
		}
		if !strings.Contains(out, "1 resolved") {
			t.Errorf("expected 1 resolved after target close, got:\n%s", out)
		}
	})

	t.Run("type_filter_scopes_evaluation", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "gcf")
		timerGate := issuegraphProxiedCreate(t, issuegraph, p.dir, "Timer gate", "--type", "gate")
		prGate := issuegraphProxiedCreate(t, issuegraph, p.dir, "PR gate", "--type", "gate")
		db := openProxiedDB(t, p)
		seedGateAwait(t, db, timerGate.ID, "timer", "", longAgo(), expiredTimeout)
		seedGateAwait(t, db, prGate.ID, "gh:pr", "1", time.Now().UTC(), 0)

		if _, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "gate", "check", "--type", "timer"); err != nil {
			t.Fatalf("gate check --type timer failed: %v\nstderr:\n%s", err, stderr)
		}
		if got := readStatus(t, db, timerGate.ID); got != types.StatusClosed {
			t.Errorf("timer gate should be closed under --type timer, got %q", got)
		}
		if got := readStatus(t, db, prGate.ID); got == types.StatusClosed {
			t.Error("gh:pr gate must be untouched under --type timer")
		}
	})

	t.Run("limit_bounds_gates_checked", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "gcl")
		db := openProxiedDB(t, p)
		var ids []string
		for i := 0; i < 3; i++ {
			g := issuegraphProxiedCreate(t, issuegraph, p.dir, "Limit timer gate", "--type", "gate")
			seedGateAwait(t, db, g.ID, "timer", "", longAgo(), expiredTimeout)
			ids = append(ids, g.ID)
		}

		if _, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "gate", "check", "--type", "timer", "--limit", "1"); err != nil {
			t.Fatalf("gate check --limit 1 failed: %v\nstderr:\n%s", err, stderr)
		}
		closed := 0
		for _, id := range ids {
			if readStatus(t, db, id) == types.StatusClosed {
				closed++
			}
		}
		if closed > 1 {
			t.Errorf("--limit 1 should close at most 1 gate, closed %d", closed)
		}
	})

	t.Run("timer_escalate_is_noop_and_resolves", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "gcx")
		gate := issuegraphProxiedCreate(t, issuegraph, p.dir, "Escalate timer gate", "--type", "gate")
		db := openProxiedDB(t, p)
		seedGateAwait(t, db, gate.ID, "timer", "", longAgo(), expiredTimeout)

		out, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "gate", "check", "--type", "timer", "--escalate")
		if err != nil {
			t.Fatalf("gate check --escalate failed: %v\nstderr:\n%s", err, stderr)
		}
		if got := readStatus(t, db, gate.ID); got != types.StatusClosed {
			t.Errorf("expired timer gate should resolve with --escalate, got %q", got)
		}
		if strings.Contains(out, "ESCALATE") {
			t.Errorf("timer gates never escalate; --escalate should be a no-op, got:\n%s", out)
		}
	})
}

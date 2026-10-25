//go:build cgo

package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func parseProxiedJSONObject(t *testing.T, stdout string) map[string]interface{} {
	t.Helper()
	start := strings.Index(stdout, "{")
	if start < 0 {
		t.Fatalf("no JSON object found in output:\n%s", stdout)
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(stdout[start:]), &m); err != nil {
		t.Fatalf("failed to parse JSON object: %v\nraw: %s", err, stdout[start:])
	}
	return m
}

func TestProxiedServerPing(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	p := newSharedProxiedProject(t, issuegraph, "ping")

	t.Run("human", func(t *testing.T) {
		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "ping")
		if err != nil {
			t.Fatalf("issuegraph ping failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
		}
		if !strings.Contains(stdout, "issuegraph ping: ok") {
			t.Errorf("expected 'issuegraph ping: ok' in output, got: %s", stdout)
		}
	})

	t.Run("json", func(t *testing.T) {
		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "ping", "--json")
		if err != nil {
			t.Fatalf("issuegraph ping --json failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
		}
		if !strings.Contains(stdout, `"status": "ok"`) {
			t.Errorf("expected status ok in JSON, got: %s", stdout)
		}
	})
}

func TestProxiedServerGC(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	p := newSharedProxiedProject(t, issuegraph, "gc")

	t.Run("dry_run_all_phases", func(t *testing.T) {
		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "gc", "--dry-run")
		if err != nil {
			t.Fatalf("issuegraph gc --dry-run failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
		}
		for _, want := range []string{"Phase 1/3", "Phase 2/3", "Phase 3/3", "DRY RUN complete"} {
			if !strings.Contains(stdout, want) {
				t.Errorf("expected %q in gc --dry-run output, got: %s", want, stdout)
			}
		}
	})

	t.Run("decay_deletes_closed_issue", func(t *testing.T) {
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "decay me", "--type", "task")
		issuegraphProxiedClose(t, issuegraph, p.dir, issue.ID)
		issuegraphProxiedSQL(t, issuegraph, p.dir,
			"UPDATE issues SET closed_at = '2000-01-01 00:00:00' WHERE id = '"+issue.ID+"'")

		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir,
			"gc", "--older-than", "1", "--skip-dolt", "--force")
		if err != nil {
			t.Fatalf("issuegraph gc decay failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
		}
		if !strings.Contains(stdout, "Deleted 1 issue") {
			t.Errorf("expected 'Deleted 1 issue' in output, got: %s", stdout)
		}

		rows := issuegraphProxiedSQLJSON(t, issuegraph, p.dir,
			"SELECT COUNT(*) as count FROM issues WHERE id = '"+issue.ID+"'")
		if len(rows) != 1 || !sqlValueEquals(rows[0]["count"], 0) {
			t.Errorf("expected deleted issue gone, got: %v", rows)
		}
	})

	t.Run("refuses_without_force", func(t *testing.T) {
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "keep me", "--type", "task")
		issuegraphProxiedClose(t, issuegraph, p.dir, issue.ID)
		issuegraphProxiedSQL(t, issuegraph, p.dir,
			"UPDATE issues SET closed_at = '2000-01-01 00:00:00' WHERE id = '"+issue.ID+"'")

		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir,
			"gc", "--older-than", "1", "--skip-dolt")
		if err == nil {
			t.Fatalf("issuegraph gc without --force should have failed\nstdout:\n%s", stdout)
		}
		if !strings.Contains(stdout+stderr, "--force") {
			t.Errorf("expected --force hint in refusal, got:\n%s\n%s", stdout, stderr)
		}
	})

	t.Run("skip_decay", func(t *testing.T) {
		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "gc", "--dry-run", "--skip-decay")
		if err != nil {
			t.Fatalf("issuegraph gc --skip-decay failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
		}
		if !strings.Contains(stdout, "Decay: skipped") {
			t.Errorf("expected 'Decay: skipped', got: %s", stdout)
		}
	})

	t.Run("skip_dolt", func(t *testing.T) {
		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "gc", "--dry-run", "--skip-dolt")
		if err != nil {
			t.Fatalf("issuegraph gc --skip-dolt failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
		}
		if !strings.Contains(stdout, "Dolt GC: skipped") {
			t.Errorf("expected 'Dolt GC: skipped', got: %s", stdout)
		}
	})

	t.Run("json_summary", func(t *testing.T) {
		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "gc", "--dry-run", "--json")
		if err != nil {
			t.Fatalf("issuegraph gc --json failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
		}
		m := parseProxiedJSONObject(t, stdout)
		if m["dry_run"] != true {
			t.Errorf("expected dry_run=true, got: %v", m["dry_run"])
		}
		phases, ok := m["phases"].([]interface{})
		if !ok || len(phases) != 3 {
			t.Fatalf("expected 3 phases in JSON, got: %v", m["phases"])
		}
		names := map[string]bool{}
		for _, ph := range phases {
			if pm, ok := ph.(map[string]interface{}); ok {
				names[fmt.Sprintf("%v", pm["name"])] = true
			}
		}
		for _, want := range []string{"Decay", "Compact", "Dolt GC"} {
			if !names[want] {
				t.Errorf("expected phase %q in JSON, got: %v", want, names)
			}
		}
	})
}

func TestProxiedServerCompactDolt(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	p := newSharedProxiedProject(t, issuegraph, "compact")

	t.Run("dolt_dry_run", func(t *testing.T) {
		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "admin", "compact", "--dolt", "--dry-run")
		if err != nil {
			t.Fatalf("issuegraph admin compact --dolt --dry-run failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
		}
		if !strings.Contains(stdout, "Dolt garbage collection") {
			t.Errorf("expected dry-run GC message, got: %s", stdout)
		}
	})

	t.Run("non_dolt_mode_rejected", func(t *testing.T) {
		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "admin", "compact", "--stats")
		if err == nil {
			t.Fatalf("issuegraph admin compact --stats should be rejected in proxied mode\nstdout:\n%s", stdout)
		}
		if !strings.Contains(stdout+stderr, "only 'issuegraph admin compact --dolt' is supported in proxied-server mode") {
			t.Errorf("expected scoped rejection message, got:\n%s\n%s", stdout, stderr)
		}
	})
}

func TestProxiedServerCompactHistory(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	p := newSharedProxiedProject(t, issuegraph, "cphist")

	for _, title := range []string{"keep-alpha", "keep-beta", "keep-gamma"} {
		issuegraphProxiedCreate(t, issuegraph, p.dir, title, "--type", "task")
	}

	commitsBefore := proxiedCommitCount(t, issuegraph, p)
	if commitsBefore <= 1 {
		t.Fatalf("expected multiple commits before compaction, got %d", commitsBefore)
	}

	t.Run("dry_run_previews", func(t *testing.T) {
		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "compact", "--days", "0", "--dry-run")
		if err != nil {
			t.Fatalf("compact --dry-run failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
		}
		if !strings.Contains(stdout, "DRY RUN") || !strings.Contains(stdout, "Run with --force") {
			t.Errorf("expected dry-run preview, got: %s", stdout)
		}
	})

	t.Run("json_dry_run", func(t *testing.T) {
		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "compact", "--days", "0", "--dry-run", "--json")
		if err != nil {
			t.Fatalf("compact --dry-run --json failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
		}
		m := parseProxiedJSONObject(t, stdout)
		if m["dry_run"] != true {
			t.Errorf("expected dry_run=true, got: %v", m["dry_run"])
		}
		if _, ok := m["total_commits"]; !ok {
			t.Errorf("expected total_commits field, got: %v", m)
		}
	})

	t.Run("refuses_without_force", func(t *testing.T) {
		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "compact", "--days", "0")
		if err == nil {
			t.Fatalf("compact without --force should have failed\nstdout:\n%s", stdout)
		}
		if !strings.Contains(stdout+stderr, "--force") {
			t.Errorf("expected --force hint, got:\n%s\n%s", stdout, stderr)
		}
	})

	t.Run("squashes_and_preserves_data", func(t *testing.T) {
		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "compact", "--days", "0", "--force")
		if err != nil {
			t.Fatalf("compact --force failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
		}
		if !strings.Contains(stdout, "Compacted") {
			t.Errorf("expected 'Compacted' summary, got: %s", stdout)
		}

		commitsAfter := proxiedCommitCount(t, issuegraph, p)
		if commitsAfter >= commitsBefore {
			t.Errorf("expected fewer commits after compaction: before=%d after=%d", commitsBefore, commitsAfter)
		}

		rows := issuegraphProxiedSQLJSON(t, issuegraph, p.dir, "SELECT title FROM issues ORDER BY title")
		if len(rows) != 3 {
			t.Fatalf("expected 3 issues to survive compaction, got %d: %v", len(rows), rows)
		}
		for i, want := range []string{"keep-alpha", "keep-beta", "keep-gamma"} {
			if rows[i]["title"] != want {
				t.Errorf("issue %d: expected %q, got %q", i, want, rows[i]["title"])
			}
		}
	})

	t.Run("nothing_to_compact_when_all_recent", func(t *testing.T) {
		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "compact", "--days", "3650", "--force")
		if err != nil {
			t.Fatalf("compact --days 3650 failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
		}
		if !strings.Contains(stdout, "Nothing to compact") {
			t.Errorf("expected 'Nothing to compact', got: %s", stdout)
		}
	})
}

func proxiedCommitCount(t *testing.T, issuegraph string, p proxiedProject) int {
	t.Helper()
	rows := issuegraphProxiedSQLJSON(t, issuegraph, p.dir, "SELECT COUNT(*) as count FROM dolt_log")
	if len(rows) != 1 {
		t.Fatalf("expected 1 row from dolt_log count, got %d", len(rows))
	}
	switch v := rows[0]["count"].(type) {
	case float64:
		return int(v)
	case string:
		n := 0
		if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
			t.Fatalf("failed to parse commit count %q: %v", v, err)
		}
		return n
	default:
		t.Fatalf("unexpected count type %T: %v", v, v)
		return 0
	}
}

func TestProxiedServerCleanDatabases(t *testing.T) {
	requireProxiedServerEnv(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	p := issuegraphProxiedInit(t, issuegraph, "clean")

	staleDB := "testdb_clean_probe"
	keepDB := "keepdb_clean_probe"
	issuegraphProxiedSQL(t, issuegraph, p.dir, "CREATE DATABASE "+staleDB)
	issuegraphProxiedSQL(t, issuegraph, p.dir, "CREATE DATABASE "+keepDB)
	t.Cleanup(func() {
		_, _, _ = issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "sql", "DROP DATABASE IF EXISTS `"+keepDB+"`")
	})

	databaseExists := func(name string) bool {
		rows := issuegraphProxiedSQLJSON(t, issuegraph, p.dir,
			"SELECT COUNT(*) as count FROM information_schema.schemata WHERE schema_name = '"+name+"'")
		return len(rows) == 1 && sqlValueEquals(rows[0]["count"], 1)
	}

	t.Run("dry_run_lists_without_dropping", func(t *testing.T) {
		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "dolt", "clean-databases", "--dry-run")
		if err != nil {
			t.Fatalf("clean-databases --dry-run failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
		}
		if !strings.Contains(stdout, staleDB) || !strings.Contains(stdout, "dry run") {
			t.Errorf("expected dry-run to list %s, got: %s", staleDB, stdout)
		}
		if strings.Contains(stdout, keepDB) {
			t.Errorf("dry-run should not list non-stale %s, got: %s", keepDB, stdout)
		}
		if !databaseExists(staleDB) {
			t.Errorf("dry-run must not drop %s", staleDB)
		}
	})

	t.Run("drops_stale_leaves_non_stale", func(t *testing.T) {
		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "dolt", "clean-databases")
		if err != nil {
			t.Fatalf("clean-databases failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
		}
		if !strings.Contains(stdout, "Dropped: "+staleDB) {
			t.Errorf("expected %s dropped, got: %s", staleDB, stdout)
		}
		if strings.Contains(stdout, keepDB) {
			t.Errorf("clean-databases must not touch non-stale %s, got: %s", keepDB, stdout)
		}
		if databaseExists(staleDB) {
			t.Errorf("expected %s gone", staleDB)
		}
		if !databaseExists(keepDB) {
			t.Errorf("expected non-stale %s to survive", keepDB)
		}
	})

	t.Run("empty_case_reports_none", func(t *testing.T) {
		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "dolt", "clean-databases")
		if err != nil {
			t.Fatalf("clean-databases (empty) failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
		}
		if !strings.Contains(stdout, "No stale databases found") {
			t.Errorf("expected 'No stale databases found', got: %s", stdout)
		}
		if !databaseExists(keepDB) {
			t.Errorf("expected non-stale %s to survive empty run", keepDB)
		}
	})
}

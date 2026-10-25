//go:build cgo

package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestProxiedServerMolBurn(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)

	t.Run("burns_single_wisp_molecule", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "mbu")
		root := issuegraphProxiedCreate(t, issuegraph, p.dir, "Wisp burn root", "--type", "epic", "--ephemeral")
		child := issuegraphProxiedCreate(t, issuegraph, p.dir, "Wisp burn child", "--type", "task", "--parent", root.ID, "--ephemeral")

		out, err := issuegraphProxiedRun(t, issuegraph, p.dir, "mol", "burn", root.ID, "--force", "--json")
		if err != nil {
			t.Fatalf("issuegraph mol burn --force --json: %v\n%s", err, out)
		}

		db := openProxiedDB(t, p)
		for _, id := range []string{root.ID, child.ID} {
			var count int
			if err := db.QueryRow("SELECT COUNT(*) FROM wisps WHERE id = ?", id).Scan(&count); err != nil {
				t.Fatalf("query wisps for %s: %v", id, err)
			}
			if count != 0 {
				t.Errorf("expected %s to be deleted from wisps table", id)
			}
		}
	})

	t.Run("burns_single_persistent_molecule", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "mbup")
		root := issuegraphProxiedCreate(t, issuegraph, p.dir, "Persistent burn root", "--type", "epic")
		child := issuegraphProxiedCreate(t, issuegraph, p.dir, "Persistent burn child", "--type", "task", "--parent", root.ID)

		out, err := issuegraphProxiedRun(t, issuegraph, p.dir, "mol", "burn", root.ID, "--force", "--json")
		if err != nil {
			t.Fatalf("issuegraph mol burn --force --json: %v\n%s", err, out)
		}

		db := openProxiedDB(t, p)
		for _, id := range []string{root.ID, child.ID} {
			var count int
			if err := db.QueryRow("SELECT COUNT(*) FROM issues WHERE id = ?", id).Scan(&count); err != nil {
				t.Fatalf("query issues for %s: %v", id, err)
			}
			if count != 0 {
				t.Errorf("expected %s to be deleted from issues table", id)
			}
		}
	})

	t.Run("burns_multiple_mixed_wisp_and_persistent", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "mbum")
		wispRoot := issuegraphProxiedCreate(t, issuegraph, p.dir, "Mixed wisp root", "--type", "epic", "--ephemeral")
		persistentRoot := issuegraphProxiedCreate(t, issuegraph, p.dir, "Mixed persistent root", "--type", "epic")

		out, err := issuegraphProxiedRun(t, issuegraph, p.dir, "mol", "burn", wispRoot.ID, persistentRoot.ID, "--force", "--json")
		if err != nil {
			t.Fatalf("issuegraph mol burn --force --json (batch): %v\n%s", err, out)
		}
		var got struct {
			TotalDeleted int `json:"total_deleted"`
		}
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("unmarshal: %v\n%s", err, out)
		}
		if got.TotalDeleted != 2 {
			t.Errorf("total_deleted = %d, want 2", got.TotalDeleted)
		}

		db := openProxiedDB(t, p)
		var wispCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM wisps WHERE id = ?", wispRoot.ID).Scan(&wispCount); err != nil {
			t.Fatalf("query wisps: %v", err)
		}
		if wispCount != 0 {
			t.Errorf("expected wisp root %s deleted", wispRoot.ID)
		}
		var issueCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM issues WHERE id = ?", persistentRoot.ID).Scan(&issueCount); err != nil {
			t.Fatalf("query issues: %v", err)
		}
		if issueCount != 0 {
			t.Errorf("expected persistent root %s deleted", persistentRoot.ID)
		}
	})

	t.Run("dry_run_deletes_nothing", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "mbud")
		root := issuegraphProxiedCreate(t, issuegraph, p.dir, "Dry burn root", "--type", "epic")

		stdout, _, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "mol", "burn", root.ID, "--dry-run")
		if err != nil {
			t.Fatalf("issuegraph mol burn --dry-run: %v\n%s", err, stdout)
		}
		if !strings.Contains(stdout, "Dry run") {
			t.Errorf("expected dry-run preview, got: %s", stdout)
		}

		spawned := issuegraphProxiedShow(t, issuegraph, p.dir, root.ID)
		if spawned.ID != root.ID {
			t.Errorf("expected root %s to survive --dry-run", root.ID)
		}
	})
}

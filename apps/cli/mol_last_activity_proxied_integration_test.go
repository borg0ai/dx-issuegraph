//go:build cgo

package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestProxiedServerMolLastActivity(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)

	t.Run("root_only_reports_molecule_updated", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "mla")
		root := issuegraphProxiedCreate(t, issuegraph, p.dir, "Lonely root", "--type", "epic")

		out, err := issuegraphProxiedRun(t, issuegraph, p.dir, "mol", "last-activity", root.ID, "--json")
		if err != nil {
			t.Fatalf("issuegraph mol last-activity --json: %v\n%s", err, out)
		}
		var got struct {
			MoleculeID string `json:"molecule_id"`
			Source     string `json:"source"`
		}
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("unmarshal: %v\n%s", err, out)
		}
		if got.Source != "molecule_updated" {
			t.Errorf("source = %s, want molecule_updated", got.Source)
		}
	})

	t.Run("child_update_wins_over_root", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "mlac")
		root := issuegraphProxiedCreate(t, issuegraph, p.dir, "Root before child", "--type", "epic")
		child := issuegraphProxiedCreate(t, issuegraph, p.dir, "Child updated later", "--type", "task", "--parent", root.ID)

		db := openProxiedDB(t, p)
		now := time.Now().UTC()
		if _, err := db.Exec("UPDATE issues SET updated_at = ? WHERE id = ?", now.Add(-time.Hour), root.ID); err != nil {
			t.Fatalf("backdate root: %v", err)
		}
		if _, err := db.Exec("UPDATE issues SET updated_at = ? WHERE id = ?", now, child.ID); err != nil {
			t.Fatalf("set child updated_at: %v", err)
		}

		out, err := issuegraphProxiedRun(t, issuegraph, p.dir, "mol", "last-activity", root.ID, "--json")
		if err != nil {
			t.Fatalf("issuegraph mol last-activity --json: %v\n%s", err, out)
		}
		var got struct {
			Source       string `json:"source"`
			SourceStepID string `json:"source_step_id"`
		}
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("unmarshal: %v\n%s", err, out)
		}
		if got.Source != "step_updated" || got.SourceStepID != child.ID {
			t.Errorf("source/step = %s/%s, want step_updated/%s", got.Source, got.SourceStepID, child.ID)
		}
	})

	t.Run("child_closed_after_update_wins", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "mlacc")
		root := issuegraphProxiedCreate(t, issuegraph, p.dir, "Root for closed-child test", "--type", "epic")
		child := issuegraphProxiedCreate(t, issuegraph, p.dir, "Child closed last", "--type", "task", "--parent", root.ID)
		if out, err := issuegraphProxiedRun(t, issuegraph, p.dir, "close", child.ID); err != nil {
			t.Fatalf("issuegraph close: %v\n%s", err, out)
		}

		db := openProxiedDB(t, p)
		now := time.Now().UTC()
		if _, err := db.Exec("UPDATE issues SET updated_at = ? WHERE id = ?", now.Add(-time.Hour), root.ID); err != nil {
			t.Fatalf("backdate root: %v", err)
		}
		if _, err := db.Exec("UPDATE issues SET updated_at = ?, closed_at = ? WHERE id = ?", now.Add(-30*time.Minute), now, child.ID); err != nil {
			t.Fatalf("set child closed_at: %v", err)
		}

		out, err := issuegraphProxiedRun(t, issuegraph, p.dir, "mol", "last-activity", root.ID, "--json")
		if err != nil {
			t.Fatalf("issuegraph mol last-activity --json: %v\n%s", err, out)
		}
		var got struct {
			Source       string `json:"source"`
			SourceStepID string `json:"source_step_id"`
		}
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("unmarshal: %v\n%s", err, out)
		}
		if got.Source != "step_closed" || got.SourceStepID != child.ID {
			t.Errorf("source/step = %s/%s, want step_closed/%s", got.Source, got.SourceStepID, child.ID)
		}
	})
}

//go:build cgo

package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestProxiedServerPour(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)

	t.Run("pours_db_proto_persistent", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "pou")
		proto := issuegraphProxiedCreate(t, issuegraph, p.dir, "Feature proto", "--type", "epic", "--label", "template")
		issuegraphProxiedCreate(t, issuegraph, p.dir, "Implement", "--type", "task", "--parent", proto.ID)

		out, err := issuegraphProxiedRun(t, issuegraph, p.dir, "mol", "pour", proto.ID, "--assignee", "worker1", "--json")
		if err != nil {
			t.Fatalf("issuegraph mol pour --json: %v\n%s", err, out)
		}
		var got struct {
			NewEpicID string `json:"new_epic_id"`
			Created   int    `json:"created"`
			Phase     string `json:"phase"`
		}
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("unmarshal: %v\n%s", err, out)
		}
		if got.Created != 2 {
			t.Errorf("created = %d, want 2", got.Created)
		}
		if got.Phase != "liquid" {
			t.Errorf("phase = %s, want liquid", got.Phase)
		}

		spawned := issuegraphProxiedShow(t, issuegraph, p.dir, got.NewEpicID)
		if spawned.Ephemeral {
			t.Error("poured mol should not be ephemeral")
		}
		if spawned.Assignee != "worker1" {
			t.Errorf("assignee = %s, want worker1", spawned.Assignee)
		}
	})

	t.Run("missing_required_variable_errors", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "pouv")
		proto := issuegraphProxiedCreate(t, issuegraph, p.dir, "Proto with {{env}}", "--type", "epic", "--label", "template")
		issuegraphProxiedCreate(t, issuegraph, p.dir, "Deploy to {{env}}", "--type", "task", "--parent", proto.ID)

		stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "mol", "pour", proto.ID)
		if err == nil {
			t.Fatalf("expected missing-variable error, got stdout:%s stderr:%s", stdout, stderr)
		}
		combined := stdout + stderr
		if !strings.Contains(combined, "env") {
			t.Errorf("expected error to mention missing variable 'env', got: %s", combined)
		}
	})

	t.Run("attach_bonds_second_proto", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "poua")
		main := issuegraphProxiedCreate(t, issuegraph, p.dir, "Main proto", "--type", "epic", "--label", "template")
		issuegraphProxiedCreate(t, issuegraph, p.dir, "Main step", "--type", "task", "--parent", main.ID)
		attach := issuegraphProxiedCreate(t, issuegraph, p.dir, "Attach proto", "--type", "epic", "--label", "template")
		issuegraphProxiedCreate(t, issuegraph, p.dir, "Attach step", "--type", "task", "--parent", attach.ID)

		out, err := issuegraphProxiedRun(t, issuegraph, p.dir, "mol", "pour", main.ID, "--attach", attach.ID, "--attach-type", "parallel", "--json")
		if err != nil {
			t.Fatalf("issuegraph mol pour --attach --json: %v\n%s", err, out)
		}
		var got struct {
			NewEpicID string `json:"new_epic_id"`
			Attached  int    `json:"attached"`
		}
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("unmarshal: %v\n%s", err, out)
		}
		if got.Attached != 2 {
			t.Errorf("attached = %d, want 2 (attach root + attach step)", got.Attached)
		}
	})

	t.Run("dry_run_creates_nothing", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "poud")
		proto := issuegraphProxiedCreate(t, issuegraph, p.dir, "Dry proto", "--type", "epic", "--label", "template")
		issuegraphProxiedCreate(t, issuegraph, p.dir, "Dry step", "--type", "task", "--parent", proto.ID)

		before := issuegraphProxiedListJSON(t, issuegraph, p, "--all")
		stdout, _, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "mol", "pour", proto.ID, "--dry-run")
		if err != nil {
			t.Fatalf("issuegraph mol pour --dry-run: %v\n%s", err, stdout)
		}
		if !strings.Contains(stdout, "Dry run") {
			t.Errorf("expected dry-run preview, got: %s", stdout)
		}
		after := issuegraphProxiedListJSON(t, issuegraph, p, "--all")
		if len(after) != len(before) {
			t.Errorf("expected no new issues from --dry-run: before=%d after=%d", len(before), len(after))
		}
	})
}

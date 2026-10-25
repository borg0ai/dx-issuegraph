//go:build cgo

package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func issuegraphProxiedRunEnv(t *testing.T, issuegraph, dir string, extraEnv []string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(issuegraph, args...)
	cmd.Dir = dir
	cmd.Env = append(issuegraphProxiedEnv(dir), extraEnv...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func TestProxiedServerAssign(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)

	t.Run("assign_then_unassign", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "as")
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Assign target")

		out, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "assign", issue.ID, "alice")
		if err != nil {
			t.Fatalf("assign failed: %v\nstderr:\n%s", err, stderr)
		}
		if !strings.Contains(out, "Assigned") {
			t.Errorf("expected 'Assigned' confirmation, got:\n%s", out)
		}
		if got := issuegraphProxiedShow(t, issuegraph, p.dir, issue.ID); got.Assignee != "alice" {
			t.Errorf("assignee = %q, want alice", got.Assignee)
		}

		out, _, err = issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "assign", issue.ID, "")
		if err != nil {
			t.Fatalf("unassign failed: %v", err)
		}
		if !strings.Contains(out, "Unassigned") {
			t.Errorf("expected 'Unassigned' confirmation, got:\n%s", out)
		}
		if got := issuegraphProxiedShow(t, issuegraph, p.dir, issue.ID); got.Assignee != "" {
			t.Errorf("assignee after unassign = %q, want empty", got.Assignee)
		}
	})

	t.Run("assign_missing_issue_fails", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "am")
		_, _, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "assign", "as-nope99", "alice")
		if err == nil {
			t.Error("expected assign of missing issue to fail")
		}
	})
}

func TestProxiedServerPriority(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)

	p := newSharedProxiedProject(t, issuegraph, "pr")
	issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Priority target", "--priority", "3")

	out, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "priority", issue.ID, "0")
	if err != nil {
		t.Fatalf("priority failed: %v\nstderr:\n%s", err, stderr)
	}
	if !strings.Contains(out, "P0") {
		t.Errorf("expected 'P0' in output, got:\n%s", out)
	}
	if got := issuegraphProxiedShow(t, issuegraph, p.dir, issue.ID); got.Priority != 0 {
		t.Errorf("priority = %d, want 0", got.Priority)
	}
}

func TestProxiedServerNote(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)

	p := newSharedProxiedProject(t, issuegraph, "nt")
	issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Note target")

	if _, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "note", issue.ID, "first note"); err != nil {
		t.Fatalf("note failed: %v\nstderr:\n%s", err, stderr)
	}
	if _, _, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "note", issue.ID, "second note"); err != nil {
		t.Fatalf("second note failed: %v", err)
	}

	got := issuegraphProxiedShow(t, issuegraph, p.dir, issue.ID)
	if !strings.Contains(got.Notes, "first note") || !strings.Contains(got.Notes, "second note") {
		t.Errorf("notes = %q, want both notes appended", got.Notes)
	}
	if strings.Index(got.Notes, "first note") > strings.Index(got.Notes, "second note") {
		t.Errorf("notes appended out of order: %q", got.Notes)
	}
}

func TestProxiedServerTag(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)

	t.Run("tag_issue", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "tg")
		issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Tag target")

		out, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "tag", issue.ID, "needs-review")
		if err != nil {
			t.Fatalf("tag failed: %v\nstderr:\n%s", err, stderr)
		}
		if !strings.Contains(out, "needs-review") {
			t.Errorf("expected label in output, got:\n%s", out)
		}

		db := openProxiedDB(t, p)
		labels := getProxiedLabels(t, db, issue.ID)
		if len(labels) != 1 || labels[0] != "needs-review" {
			t.Errorf("persisted labels = %v, want [needs-review]", labels)
		}
	})

	t.Run("tag_wisp_routes_to_wisp_labels", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "tw")
		wisp := issuegraphProxiedCreate(t, issuegraph, p.dir, "Tag wisp", "--ephemeral")

		if _, _, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, "tag", wisp.ID, "wtag"); err != nil {
			t.Fatalf("tag wisp failed: %v", err)
		}

		db := openProxiedDB(t, p)
		var wispCount, permCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM wisp_labels WHERE issue_id = ?", wisp.ID).Scan(&wispCount); err != nil {
			t.Fatalf("count wisp_labels: %v", err)
		}
		if err := db.QueryRow("SELECT COUNT(*) FROM labels WHERE issue_id = ?", wisp.ID).Scan(&permCount); err != nil {
			t.Fatalf("count labels: %v", err)
		}
		if wispCount != 1 || permCount != 0 {
			t.Errorf("wisp tag routing: wisp_labels=%d labels=%d, want 1/0", wispCount, permCount)
		}
	})
}

func TestProxiedServerEdit(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)

	p := newSharedProxiedProject(t, issuegraph, "ed")
	issue := issuegraphProxiedCreate(t, issuegraph, p.dir, "Edit target", "--description", "original body")

	editorScript := filepath.Join(p.dir, "editor.sh")
	if err := os.WriteFile(editorScript, []byte("#!/bin/sh\nprintf 'edited via proxied editor' > \"$1\"\n"), 0o755); err != nil {
		t.Fatalf("write editor script: %v", err)
	}
	env := []string{"EDITOR=" + editorScript, "VISUAL="}

	out, stderr, err := issuegraphProxiedRunEnv(t, issuegraph, p.dir, env, "edit", issue.ID)
	if err != nil {
		t.Fatalf("edit failed: %v\nstdout:\n%s\nstderr:\n%s", err, out, stderr)
	}
	if !strings.Contains(out, "Updated description") {
		t.Errorf("expected 'Updated description' output, got:\n%s", out)
	}
	if got := issuegraphProxiedShow(t, issuegraph, p.dir, issue.ID); got.Description != "edited via proxied editor" {
		t.Errorf("description = %q, want edited value", got.Description)
	}
}

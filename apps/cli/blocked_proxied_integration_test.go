//go:build cgo

package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/types"
)

func issuegraphProxiedBlockedJSON(t *testing.T, issuegraph string, p proxiedProject, args ...string) []*types.BlockedIssue {
	t.Helper()
	fullArgs := append([]string{"blocked", "--json"}, args...)
	stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, fullArgs...)
	if err != nil {
		t.Fatalf("issuegraph blocked --json %s failed: %v\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), err, stdout, stderr)
	}
	s := strings.TrimSpace(stdout)
	start := strings.Index(s, "[")
	if start < 0 {
		t.Fatalf("no JSON array in blocked --json output: %s", stdout)
	}
	var out []*types.BlockedIssue
	if err := json.Unmarshal([]byte(s[start:]), &out); err != nil {
		t.Fatalf("parse blocked JSON: %v\n%s", err, s[start:])
	}
	return out
}

func issuegraphProxiedBlockedCapture(t *testing.T, issuegraph string, p proxiedProject, args ...string) (string, string) {
	t.Helper()
	fullArgs := append([]string{"blocked"}, args...)
	stdout, stderr, err := issuegraphProxiedRunBuffers(t, issuegraph, p.dir, fullArgs...)
	if err != nil {
		t.Fatalf("issuegraph blocked %s failed: %v\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), err, stdout, stderr)
	}
	return stdout, stderr
}

func TestProxiedServerBlocked(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)

	t.Run("json_and_text_with_blockers", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "bk1")
		blocker := issuegraphProxiedCreate(t, issuegraph, p.dir, "The blocker")
		dependent := issuegraphProxiedCreate(t, issuegraph, p.dir, "I am blocked", "--deps", "depends-on:"+blocker.ID)

		got := issuegraphProxiedBlockedJSON(t, issuegraph, p)
		var entry *types.BlockedIssue
		for _, bi := range got {
			if bi.ID == dependent.ID {
				entry = bi
				break
			}
		}
		if entry == nil {
			out, _ := json.Marshal(got)
			t.Fatalf("expected %s in blocked listing, got: %s", dependent.ID, out)
		}
		if entry.BlockedByCount != 1 {
			t.Errorf("BlockedByCount = %d, want 1", entry.BlockedByCount)
		}
		if len(entry.BlockedBy) != 1 || entry.BlockedBy[0] != blocker.ID {
			t.Errorf("BlockedBy = %v, want [%s]", entry.BlockedBy, blocker.ID)
		}

		stdout, _ := issuegraphProxiedBlockedCapture(t, issuegraph, p)
		if !strings.Contains(stdout, "Blocked issues") {
			t.Errorf("expected heading in text output, got: %s", stdout)
		}
		if !strings.Contains(stdout, "Blocked by 1 open dependencies") {
			t.Errorf("expected blocker summary line, got: %s", stdout)
		}
	})

	t.Run("empty_case", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "bk2")
		issuegraphProxiedCreate(t, issuegraph, p.dir, "Nothing blocked here")

		out, err := issuegraphProxiedRun(t, issuegraph, p.dir, "blocked", "--json")
		if err != nil {
			t.Fatalf("issuegraph blocked --json failed: %v\n%s", err, out)
		}
		var empty []*types.BlockedIssue
		if err := json.Unmarshal(bytes.TrimSpace(out), &empty); err != nil {
			t.Fatalf("parse empty blocked JSON: %v\n%s", err, out)
		}
		if len(empty) != 0 {
			t.Errorf("expected [] for no blockers, got %d entries", len(empty))
		}

		stdout, _ := issuegraphProxiedBlockedCapture(t, issuegraph, p)
		if !strings.Contains(stdout, "No blocked issues") {
			t.Errorf("expected 'No blocked issues' message, got: %s", stdout)
		}
	})

	t.Run("parent_filter_restricts_to_descendants", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, issuegraph, "bk3")
		parent := issuegraphProxiedCreate(t, issuegraph, p.dir, "Parent epic", "--type", "epic")
		blocker := issuegraphProxiedCreate(t, issuegraph, p.dir, "Inside blocker")
		inside := issuegraphProxiedCreate(t, issuegraph, p.dir, "Inside blocked", "--parent", parent.ID, "--deps", "depends-on:"+blocker.ID)
		outsideBlocker := issuegraphProxiedCreate(t, issuegraph, p.dir, "Outside blocker")
		outside := issuegraphProxiedCreate(t, issuegraph, p.dir, "Outside blocked", "--deps", "depends-on:"+outsideBlocker.ID)

		got := issuegraphProxiedBlockedJSON(t, issuegraph, p, "--parent", parent.ID)
		ids := map[string]bool{}
		for _, bi := range got {
			ids[bi.ID] = true
		}
		if !ids[inside.ID] {
			t.Errorf("expected descendant %s in --parent result", inside.ID)
		}
		if ids[outside.ID] {
			t.Errorf("non-descendant %s leaked into --parent result", outside.ID)
		}
	})
}

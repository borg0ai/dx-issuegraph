//go:build cgo

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/types"
)

// TestCheckIssueGateCrossRigPrefixRoute proves the full evaluator seam: the
// historical rig:id value is reduced to the target issue ID, the current
// routes.jsonl prefix router opens the foreign store read-only, and target
// lifecycle state determines whether the gate resolves.
//
// NOTE: This test uses os.Chdir and cannot run in parallel with other tests.
func TestCheckIssueGateCrossRigPrefixRoute(t *testing.T) {
	ctx := context.Background()
	townRoot := t.TempDir()
	townIssueGraphDir := filepath.Join(townRoot, ".issuegraph")
	rigIssueGraphDir := filepath.Join(townRoot, "rig", ".issuegraph")
	if err := os.MkdirAll(townIssueGraphDir, 0o755); err != nil {
		t.Fatalf("create town issuegraph dir: %v", err)
	}
	if err := os.MkdirAll(rigIssueGraphDir, 0o755); err != nil {
		t.Fatalf("create rig issuegraph dir: %v", err)
	}

	townDBPath := filepath.Join(townIssueGraphDir, "dolt")
	townStore := newTestStoreIsolatedDB(t, townDBPath, "hq")
	rigStore := newTestStoreIsolatedDB(t, filepath.Join(rigIssueGraphDir, "dolt"), "gt")
	for _, issue := range []*types.Issue{
		{ID: "gt-closed", Title: "Closed routed target", Status: types.StatusClosed, Priority: 2, IssueType: types.TypeTask},
		{ID: "gt-open", Title: "Open routed target", Status: types.StatusOpen, Priority: 2, IssueType: types.TypeTask},
	} {
		if err := rigStore.CreateIssue(ctx, issue, "test"); err != nil {
			t.Fatalf("create routed issue %s: %v", issue.ID, err)
		}
	}
	if err := rigStore.Close(); err != nil {
		t.Fatalf("close rig store: %v", err)
	}

	routesPath := filepath.Join(townIssueGraphDir, "routes.jsonl")
	if err := os.WriteFile(routesPath, []byte(`{"prefix":"gt-","path":"rig"}`), 0o644); err != nil {
		t.Fatalf("write routes.jsonl: %v", err)
	}

	oldDBPath := dbPath
	dbPath = townDBPath
	t.Cleanup(func() { dbPath = oldDBPath })
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(townRoot); err != nil {
		t.Fatalf("change to town root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	getter := routedIssueGateGetter{localStore: townStore}
	resolved, reason := checkIssueGate(ctx, getter, "rig:gt-closed")
	if !resolved {
		t.Fatalf("closed cross-rig target did not resolve gate: %s", reason)
	}

	resolved, reason = checkIssueGate(ctx, getter, "rig:gt-open")
	if resolved {
		t.Fatalf("open cross-rig target unexpectedly resolved gate: %s", reason)
	}
	if !gateTestContainsIgnoreCase(reason, "open") {
		t.Fatalf("pending reason %q does not report target status", reason)
	}
}

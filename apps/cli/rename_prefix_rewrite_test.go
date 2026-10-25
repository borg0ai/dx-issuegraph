package main

import "testing"

func TestRewriteIssueID_NoDoublePrefix(t *testing.T) {
	t.Parallel()
	// GH#4827: config still "global" but rows already "atlas-*"
	got := rewriteIssueID("global", "atlas", "atlas-1")
	if got != "atlas-1" {
		t.Fatalf("got %q, want atlas-1 (must not double)", got)
	}
	got = rewriteIssueID("global-", "atlas-", "atlas-99")
	if got != "atlas-99" {
		t.Fatalf("got %q, want atlas-99", got)
	}
}

func TestRewriteIssueID_NormalRename(t *testing.T) {
	t.Parallel()
	got := rewriteIssueID("global", "atlas", "global-1")
	if got != "atlas-1" {
		t.Fatalf("got %q, want atlas-1", got)
	}
	got = rewriteIssueID("old-", "new-", "old-abc")
	if got != "new-abc" {
		t.Fatalf("got %q, want new-abc", got)
	}
}

func TestRewriteIssueID_UnrelatedPrefixUnchanged(t *testing.T) {
	t.Parallel()
	got := rewriteIssueID("global", "atlas", "other-1")
	if got != "other-1" {
		t.Fatalf("got %q, want other-1", got)
	}
}

// TestRewriteIssueID_PrefixShortening pins PR #5135 review blocker B1
// (maphew, 2026-07-29): the "already on target prefix" guard was checked
// before the old-prefix match, so shortening a multi-part prefix to one of
// its own leading segments (e.g. "issuegraph-vscode-" -> "issuegraph-") silently
// skipped every ID, because "issuegraph-vscode-1" also starts with "issuegraph-".
func TestRewriteIssueID_PrefixShortening(t *testing.T) {
	t.Parallel()
	got := rewriteIssueID("issuegraph-vscode", "issuegraph", "issuegraph-vscode-1")
	if got != "issuegraph-1" {
		t.Fatalf("got %q, want issuegraph-1 (shortening regression)", got)
	}
	got = rewriteIssueID("kw-team", "kw", "kw-team-42")
	if got != "kw-42" {
		t.Fatalf("got %q, want kw-42 (shortening regression)", got)
	}
}

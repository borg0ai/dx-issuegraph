//go:build cgo

package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/types"
)

func issuegraphCommand(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	out, err := issuegraphRunWithFlockRetry(t, issuegraph, dir, args...)
	if err != nil {
		t.Fatalf("issuegraph %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func assertEmbeddedHeadUnchanged(t *testing.T, issuegraphDir, database, before, command string) {
	t.Helper()
	after := embeddedCurrentCommit(t, issuegraphDir, database)
	if after != before {
		t.Fatalf("%s advanced HEAD in batch mode; before=%s after=%s", command, before, after)
	}
}

func assertEmbeddedHeadAdvanced(t *testing.T, issuegraphDir, database, before, command string) {
	t.Helper()
	after := embeddedCurrentCommit(t, issuegraphDir, database)
	if after == before {
		t.Fatalf("%s did not advance target HEAD; before=%s after=%s", command, before, after)
	}
}

func setupRoutedEmbeddedRepo(t *testing.T, issuegraph string, sourcePrefix, targetPrefix string) (sourceDir, targetDir, targetIssueGraphDir string) {
	t.Helper()

	sourceDir, _, _ = issuegraphInit(t, issuegraph, "--prefix", sourcePrefix)
	targetDir = filepath.Join(sourceDir, "target-repo")
	if err := os.MkdirAll(targetDir, 0750); err != nil {
		t.Fatal(err)
	}
	initGitRepoAt(t, targetDir)
	runIssueGraphInit(t, issuegraph, targetDir, "--prefix", targetPrefix)

	route := `{"prefix":"` + targetPrefix + `-","path":"target-repo"}` + "\n"
	if err := os.WriteFile(filepath.Join(sourceDir, ".issuegraph", "routes.jsonl"), []byte(route), 0644); err != nil {
		t.Fatalf("write routes.jsonl: %v", err)
	}
	return sourceDir, targetDir, filepath.Join(targetDir, ".issuegraph")
}

func TestEmbeddedDepAndLinkBatchAutoCommitDoesNotAdvanceHead(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt auto-commit tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)

	t.Run("dep_add", func(t *testing.T) {
		dir, issuegraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "da")
		from := issuegraphCreate(t, issuegraph, dir, "Dependent issue")
		to := issuegraphCreate(t, issuegraph, dir, "Blocking issue")
		before := embeddedCurrentCommit(t, issuegraphDir, "da")

		issuegraphCommand(t, issuegraph, dir, "--dolt-auto-commit", "batch", "dep", "add", from.ID, to.ID)

		assertEmbeddedHeadUnchanged(t, issuegraphDir, "da", before, "dep add")
	})

	t.Run("dep_blocks", func(t *testing.T) {
		dir, issuegraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "db")
		blocker := issuegraphCreate(t, issuegraph, dir, "Blocking issue")
		blocked := issuegraphCreate(t, issuegraph, dir, "Blocked issue")
		before := embeddedCurrentCommit(t, issuegraphDir, "db")

		issuegraphCommand(t, issuegraph, dir, "--dolt-auto-commit", "batch", "dep", blocker.ID, "--blocks", blocked.ID)

		assertEmbeddedHeadUnchanged(t, issuegraphDir, "db", before, "dep --blocks")
	})

	t.Run("dep_remove", func(t *testing.T) {
		dir, issuegraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "dr")
		from := issuegraphCreate(t, issuegraph, dir, "Dependent issue")
		to := issuegraphCreate(t, issuegraph, dir, "Blocking issue")
		issuegraphDep(t, issuegraph, dir, "add", from.ID, to.ID)
		before := embeddedCurrentCommit(t, issuegraphDir, "dr")

		issuegraphCommand(t, issuegraph, dir, "--dolt-auto-commit", "batch", "dep", "remove", from.ID, to.ID)

		assertEmbeddedHeadUnchanged(t, issuegraphDir, "dr", before, "dep remove")
	})

	t.Run("link", func(t *testing.T) {
		dir, issuegraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "dl")
		from := issuegraphCreate(t, issuegraph, dir, "Dependent issue")
		to := issuegraphCreate(t, issuegraph, dir, "Blocking issue")
		before := embeddedCurrentCommit(t, issuegraphDir, "dl")

		issuegraphCommand(t, issuegraph, dir, "--dolt-auto-commit", "batch", "link", from.ID, to.ID)

		assertEmbeddedHeadUnchanged(t, issuegraphDir, "dl", before, "link")
	})
}

func TestEmbeddedDirectCommitPathsBatchAutoCommitDoesNotAdvanceHead(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt auto-commit tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)

	t.Run("create", func(t *testing.T) {
		dir, issuegraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "dc")
		before := embeddedCurrentCommit(t, issuegraphDir, "dc")

		issuegraphCommand(t, issuegraph, dir, "--dolt-auto-commit", "batch", "create", "Deferred create")

		assertEmbeddedHeadUnchanged(t, issuegraphDir, "dc", before, "create")
	})

	t.Run("markdown_create", func(t *testing.T) {
		dir, issuegraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "dm")
		mdFile := filepath.Join(dir, "issues.md")
		mdContent := `## Markdown batch issue

### Priority
2

### Type
task

### Description
Created while Dolt auto-commit is in batch mode.
`
		if err := os.WriteFile(mdFile, []byte(mdContent), 0644); err != nil {
			t.Fatalf("write markdown fixture: %v", err)
		}
		before := embeddedCurrentCommit(t, issuegraphDir, "dm")

		issuegraphCommand(t, issuegraph, dir, "--dolt-auto-commit", "batch", "create", "-f", mdFile)

		assertEmbeddedHeadUnchanged(t, issuegraphDir, "dm", before, "markdown create")
	})

	t.Run("label_add", func(t *testing.T) {
		dir, issuegraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "dlb")
		issue := issuegraphCreate(t, issuegraph, dir, "Batch label target")
		before := embeddedCurrentCommit(t, issuegraphDir, "dlb")

		issuegraphCommand(t, issuegraph, dir, "--dolt-auto-commit", "batch", "label", "add", issue.ID, "batched")

		assertEmbeddedHeadUnchanged(t, issuegraphDir, "dlb", before, "label add")
		labels := showLabels(t, issuegraph, dir, issue.ID)
		if !slices.Contains(labels, "batched") {
			t.Fatalf("labels = %v, want batched", labels)
		}
	})

	t.Run("delete", func(t *testing.T) {
		dir, issuegraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "dd")
		issue := issuegraphCreate(t, issuegraph, dir, "Batch delete target")
		before := embeddedCurrentCommit(t, issuegraphDir, "dd")

		issuegraphCommand(t, issuegraph, dir, "--dolt-auto-commit", "batch", "delete", issue.ID, "--force")

		assertEmbeddedHeadUnchanged(t, issuegraphDir, "dd", before, "delete")
	})

	// Prune rides the Sweeper role, which mints its own version commit — so
	// batch mode has to reach it on the context, exactly as delete's does.
	// Before the role versioned an embedded sweep, this route was deferred by
	// accident: nothing under it committed at all, and the post-run flush
	// already honored the mode.
	t.Run("prune", func(t *testing.T) {
		dir, issuegraphDir, _ := issuegraphInit(t, issuegraph, "--prefix", "dp")
		issue := issuegraphCreate(t, issuegraph, dir, "Batch prune target")
		issuegraphClose(t, issuegraph, dir, issue.ID, "--reason", "done")
		before := embeddedCurrentCommit(t, issuegraphDir, "dp")

		issuegraphCommand(t, issuegraph, dir, "--dolt-auto-commit", "batch", "prune", "--pattern", "dp-*", "--force")

		assertEmbeddedHeadUnchanged(t, issuegraphDir, "dp", before, "prune")
	})
}

func TestEmbeddedRoutedSiblingWritesCommitTargetHead(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt auto-commit tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)

	t.Run("comment", func(t *testing.T) {
		sourceDir, targetDir, targetIssueGraphDir := setupRoutedEmbeddedRepo(t, issuegraph, "sc", "tc")
		issue := issuegraphCreate(t, issuegraph, targetDir, "Routed comment target")
		before := embeddedCurrentCommit(t, targetIssueGraphDir, "tc")

		issuegraphCommand(t, issuegraph, sourceDir, "comment", issue.ID, "routed comment")

		assertEmbeddedHeadAdvanced(t, targetIssueGraphDir, "tc", before, "routed comment")
		targetStore := openStore(t, targetIssueGraphDir, "tc")
		comments, err := targetStore.GetIssueComments(t.Context(), issue.ID)
		if err != nil {
			t.Fatalf("GetIssueComments: %v", err)
		}
		found := false
		for _, c := range comments {
			if c.Text == "routed comment" {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("routed comment was not persisted in target store")
		}
	})

	t.Run("note", func(t *testing.T) {
		sourceDir, targetDir, targetIssueGraphDir := setupRoutedEmbeddedRepo(t, issuegraph, "sn", "tn")
		issue := issuegraphCreate(t, issuegraph, targetDir, "Routed note target")
		before := embeddedCurrentCommit(t, targetIssueGraphDir, "tn")

		issuegraphCommand(t, issuegraph, sourceDir, "note", issue.ID, "routed note")

		assertEmbeddedHeadAdvanced(t, targetIssueGraphDir, "tn", before, "routed note")
		targetStore := openStore(t, targetIssueGraphDir, "tn")
		got, err := targetStore.GetIssue(t.Context(), issue.ID)
		if err != nil {
			t.Fatalf("GetIssue: %v", err)
		}
		if !strings.Contains(got.Notes, "routed note") {
			t.Fatalf("target notes = %q, want routed note", got.Notes)
		}
	})

	t.Run("reopen", func(t *testing.T) {
		sourceDir, targetDir, targetIssueGraphDir := setupRoutedEmbeddedRepo(t, issuegraph, "sr", "tr")
		issue := issuegraphCreate(t, issuegraph, targetDir, "Routed reopen target")
		issuegraphClose(t, issuegraph, targetDir, issue.ID)
		before := embeddedCurrentCommit(t, targetIssueGraphDir, "tr")

		issuegraphCommand(t, issuegraph, sourceDir, "reopen", issue.ID, "--reason", "routed reopen")

		assertEmbeddedHeadAdvanced(t, targetIssueGraphDir, "tr", before, "routed reopen")
		targetStore := openStore(t, targetIssueGraphDir, "tr")
		got, err := targetStore.GetIssue(t.Context(), issue.ID)
		if err != nil {
			t.Fatalf("GetIssue: %v", err)
		}
		if got.Status != types.StatusOpen {
			t.Fatalf("target status = %s, want %s", got.Status, types.StatusOpen)
		}
	})
}

// TestEmbeddedRoutedMutatingSiblingWritesCommitTargetHead pins the rest of the
// prefix-routed mutating command surface to the same write-through contract as
// comment/note/reopen above. These commands (the assign/tag shorthands plus
// dep/delete/close) resolve a prefix-routed target through the write-intent
// router and must commit on the target head, not fail "store is read-only"
// (#4141). Each subtest creates the issue in the target rig, runs the command
// from the source rig so resolution must route, and asserts both that the
// target HEAD advanced and that the mutation persisted in the target store.
func TestEmbeddedRoutedMutatingSiblingWritesCommitTargetHead(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt auto-commit tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)

	t.Run("assign", func(t *testing.T) {
		sourceDir, targetDir, targetIssueGraphDir := setupRoutedEmbeddedRepo(t, issuegraph, "sas", "tas")
		issue := issuegraphCreate(t, issuegraph, targetDir, "Routed assign target")
		before := embeddedCurrentCommit(t, targetIssueGraphDir, "tas")

		issuegraphCommand(t, issuegraph, sourceDir, "assign", issue.ID, "alice")

		assertEmbeddedHeadAdvanced(t, targetIssueGraphDir, "tas", before, "routed assign")
		targetStore := openStore(t, targetIssueGraphDir, "tas")
		got, err := targetStore.GetIssue(t.Context(), issue.ID)
		if err != nil {
			t.Fatalf("GetIssue: %v", err)
		}
		if got.Assignee != "alice" {
			t.Fatalf("target assignee = %q, want alice", got.Assignee)
		}
	})

	t.Run("tag", func(t *testing.T) {
		sourceDir, targetDir, targetIssueGraphDir := setupRoutedEmbeddedRepo(t, issuegraph, "stg", "ttg")
		issue := issuegraphCreate(t, issuegraph, targetDir, "Routed tag target")
		before := embeddedCurrentCommit(t, targetIssueGraphDir, "ttg")

		issuegraphCommand(t, issuegraph, sourceDir, "tag", issue.ID, "routed-label")

		assertEmbeddedHeadAdvanced(t, targetIssueGraphDir, "ttg", before, "routed tag")
		targetStore := openStore(t, targetIssueGraphDir, "ttg")
		labels, err := targetStore.GetLabelsForIssues(t.Context(), []string{issue.ID})
		if err != nil {
			t.Fatalf("GetLabelsForIssues: %v", err)
		}
		if !slices.Contains(labels[issue.ID], "routed-label") {
			t.Fatalf("target labels = %v, want routed-label", labels[issue.ID])
		}
	})

	t.Run("dep_add", func(t *testing.T) {
		sourceDir, targetDir, targetIssueGraphDir := setupRoutedEmbeddedRepo(t, issuegraph, "sdp", "tdp")
		from := issuegraphCreate(t, issuegraph, targetDir, "Routed dep source")
		to := issuegraphCreate(t, issuegraph, targetDir, "Routed dep target")
		before := embeddedCurrentCommit(t, targetIssueGraphDir, "tdp")

		issuegraphCommand(t, issuegraph, sourceDir, "dep", "add", from.ID, to.ID)

		assertEmbeddedHeadAdvanced(t, targetIssueGraphDir, "tdp", before, "routed dep add")
		targetStore := openStore(t, targetIssueGraphDir, "tdp")
		deps, err := targetStore.GetDependencies(t.Context(), from.ID)
		if err != nil {
			t.Fatalf("GetDependencies: %v", err)
		}
		found := false
		for _, d := range deps {
			if d.ID == to.ID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("routed dep add did not persist %s -> %s in target store (deps=%v)", from.ID, to.ID, deps)
		}
	})

	t.Run("delete", func(t *testing.T) {
		sourceDir, targetDir, targetIssueGraphDir := setupRoutedEmbeddedRepo(t, issuegraph, "sdl", "tdl")
		issue := issuegraphCreate(t, issuegraph, targetDir, "Routed delete target")
		before := embeddedCurrentCommit(t, targetIssueGraphDir, "tdl")

		issuegraphCommand(t, issuegraph, sourceDir, "delete", issue.ID, "--force")

		assertEmbeddedHeadAdvanced(t, targetIssueGraphDir, "tdl", before, "routed delete")
		targetStore := openStore(t, targetIssueGraphDir, "tdl")
		if _, err := targetStore.GetIssue(t.Context(), issue.ID); err == nil {
			t.Fatalf("routed delete did not remove %s from target store", issue.ID)
		}
	})

	t.Run("close", func(t *testing.T) {
		sourceDir, targetDir, targetIssueGraphDir := setupRoutedEmbeddedRepo(t, issuegraph, "scl", "tcl")
		issue := issuegraphCreate(t, issuegraph, targetDir, "Routed close target")
		before := embeddedCurrentCommit(t, targetIssueGraphDir, "tcl")

		issuegraphCommand(t, issuegraph, sourceDir, "close", issue.ID, "--reason", "routed close")

		assertEmbeddedHeadAdvanced(t, targetIssueGraphDir, "tcl", before, "routed close")
		targetStore := openStore(t, targetIssueGraphDir, "tcl")
		got, err := targetStore.GetIssue(t.Context(), issue.ID)
		if err != nil {
			t.Fatalf("GetIssue: %v", err)
		}
		if got.Status != types.StatusClosed {
			t.Fatalf("target status = %s, want %s", got.Status, types.StatusClosed)
		}
	})
}

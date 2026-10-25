//go:build cgo && unix

package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// The two halves of #5985 at the front door: what `issuegraph delete --actor` records,
// and what `issuegraph events export` tells you when there is nothing to record into.
// Both are end-to-end because both failures are invisible from inside the
// process — the command succeeds either way, and only a later read shows the
// attribution missing or the reader polling a journal that was never on.

// runIssueGraphOut runs issuegraph in dir and returns stdout and stderr separately. The notice
// under test is on stderr precisely so it does not corrupt the JSONL on stdout,
// so a CombinedOutput helper cannot check it.
func runIssueGraphOut(t *testing.T, issuegraph, dir string, env []string, args ...string) (string, string) {
	t.Helper()
	cmd := exec.Command(issuegraph, args...)
	cmd.Dir = dir
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("issuegraph %s in %s: %v\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), dir, err, stdout.String(), stderr.String())
	}
	return stdout.String(), stderr.String()
}

// TestDeleteJournalsTheRequestingActor is #5985's repro, verbatim: create two
// issuegraph with an edge between them, delete the blocker with an explicit --actor,
// and read the journal back. Both the delete record and the cascade dep_remove
// record must name that actor.
//
// The whole route is exercised — cobra flag to journal column — because the
// actor is resolved, threaded through the role, and written by the storage
// seam, and a unit test at any one of those cannot see the drop at the next.
func TestDeleteJournalsTheRequestingActor(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "dja", "--skip-hooks", "--skip-agents")
	env := append(envWithout(issuegraphEnv(dir), "ISSUEGRAPH_EVENTS_JOURNAL"), "ISSUEGRAPH_EVENTS_JOURNAL=1")

	run := func(args ...string) string {
		t.Helper()
		stdout, _ := runIssueGraphOut(t, issuegraph, dir, env, args...)
		return stdout
	}

	run("create", "--silent", "--id", "dja-blocker", "the blocker")
	run("create", "--silent", "--id", "dja-dependent", "the dependent")
	run("dep", "add", "dja-dependent", "dja-blocker")

	// --force is what makes this a real delete rather than a preview, and it is
	// also what lets it orphan the dependent instead of refusing.
	run("delete", "dja-blocker", "--force", "--actor", "alice")

	records := decodeEventRecords(t, run("events", "export"))
	var sawDelete, sawCascadeDepRemove bool
	for _, rec := range records {
		switch {
		case rec.Op == "delete" && rec.IssueID == "dja-blocker":
			sawDelete = true
			if rec.Actor != "alice" {
				t.Errorf("delete record actor = %q, want %q (#5985)", rec.Actor, "alice")
			}
		// The cascade edge removal is journaled under the edge's SOURCE — the
		// surviving dependent — and belongs to the identity that asked for the
		// delete, not to whoever created the edge.
		case rec.Op == "dep_remove" && rec.IssueID == "dja-dependent":
			sawCascadeDepRemove = true
			if rec.Actor != "alice" {
				t.Errorf("cascade dep_remove record actor = %q, want %q (#5985)", rec.Actor, "alice")
			}
		}
	}
	if !sawDelete {
		t.Errorf("no delete record for dja-blocker: %+v", records)
	}
	if !sawCascadeDepRemove {
		t.Errorf("no cascade dep_remove record for dja-dependent: %+v", records)
	}
}

// TestEventsReadNoticesADisabledJournal pins the "nothing to read and no way to
// tell why" fix. A workspace that never enabled the journal serves an empty
// export forever, which a consumer cannot distinguish from being caught up, so
// the read says so on stderr — and keeps stdout clean and the exit status zero,
// because exporting a historical ledger from a now-disabled workspace is
// legitimate.
func TestEventsReadNoticesADisabledJournal(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "dnn", "--skip-hooks", "--skip-agents")
	off := envWithout(issuegraphEnv(dir), "ISSUEGRAPH_EVENTS_JOURNAL")

	for _, sub := range []string{"export", "tail"} {
		stdout, stderr := runIssueGraphOut(t, issuegraph, dir, off, "events", sub)
		if !strings.Contains(stderr, "events journal is disabled") {
			t.Errorf("issuegraph events %s on a disabled workspace printed no notice\nstderr:\n%s", sub, stderr)
		}
		if !strings.Contains(stderr, "events-journal") {
			t.Errorf("issuegraph events %s notice does not name the setting to flip\nstderr:\n%s", sub, stderr)
		}
		if strings.TrimSpace(stdout) != "" {
			t.Errorf("issuegraph events %s put the notice on stdout, which carries the JSONL stream:\n%s", sub, stdout)
		}
	}

	// Enabled: no notice, so the message stays a signal rather than noise every
	// consumer learns to filter out.
	on := append(envWithout(issuegraphEnv(dir), "ISSUEGRAPH_EVENTS_JOURNAL"), "ISSUEGRAPH_EVENTS_JOURNAL=1")
	if _, stderr := runIssueGraphOut(t, issuegraph, dir, on, "events", "export"); strings.Contains(stderr, "events journal is disabled") {
		t.Errorf("an enabled workspace still printed the disabled notice:\n%s", stderr)
	}
}

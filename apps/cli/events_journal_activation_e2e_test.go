//go:build cgo && unix

package main

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/eventsjournal"
)

// The two plumbings the root pre-run never reached. Both are end-to-end on
// purpose: activation failing is INVISIBLE from inside the process — the
// command succeeds, the write lands, and only a later read of the journal shows
// that nothing was recorded. Nothing short of mutating and then reading back
// can tell the difference.

// TestServeActivatesTheEventsJournal covers `issuegraph serve` against a SERVER-MODE
// workspace, where PersistentPreRunE builds a DoltStore and no unit-of-work
// provider at all — so serve constructs its own, from the workspace's Dolt
// connection settings. That provider was never activated: every mutation the
// HTTP surface accepted committed with an empty journal while /healthz stayed
// green and every response looked normal.
func TestServeActivatesTheEventsJournal(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)
	p := newServerModeProject(t, issuegraph, "sjr")
	p.env = append(p.env, "ISSUEGRAPH_EVENTS_JOURNAL=1")

	sp := startServe(t, issuegraph, p.dir, p.env)

	status, body := sp.postJSON(t, "/v0/issuegraph/issues:batchCreate",
		`{"actor":"tester","items":[{"title":"served mutation","issue_type":"task"}]}`)
	if status != http.StatusOK {
		t.Fatalf("POST /v0/issuegraph/issues:batchCreate = %d: %v\nstderr:\n%s", status, body, sp.stderr.String())
	}
	sp.shutdown(t)

	records := decodeEventRecords(t, p.run(t, issuegraph, "events", "export"))
	if len(records) == 0 {
		t.Fatalf("issuegraph serve journaled nothing for an HTTP mutation — the provider serve builds for a server-mode workspace is not being activated")
	}
	if !hasEventOp(records, "create") {
		t.Errorf("journal has no create record for the served mutation: %+v", records)
	}
	// The exported record attributes the mutation to the acting identity the
	// request resolved — the same actor the audit-events table records.
	for _, rec := range records {
		if rec.Op == "create" && rec.Actor != "tester" {
			t.Errorf("create record actor = %q, want %q", rec.Actor, "tester")
		}
	}
}

// TestRoutedCreateJournalsIntoTheTargetWorkspace covers the cross-workspace
// store: `issuegraph create --repo <other>` opens a SECOND store for another workspace
// and writes the issue there.
//
// It pins the semantics as well as the wiring. Activation is read from the
// TARGET's own config.yaml, so a workspace that enabled the journal gets its
// mutations recorded no matter which directory issuegraph was launched from — and the
// launching workspace, which enabled nothing, records nothing of its own.
func TestRoutedCreateJournalsIntoTheTargetWorkspace(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)

	// The launching workspace: journal NOT enabled.
	sourceDir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "src", "--skip-hooks", "--skip-agents")
	// The routing target: journal enabled in its OWN config.yaml.
	targetDir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "tgt", "--skip-hooks", "--skip-agents")

	run := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command(issuegraph, args...)
		cmd.Dir = dir
		// issuegraphEnv(dir) sets HOME=dir, which each workspace needs to be its own;
		// deliberately no ISSUEGRAPH_EVENTS_JOURNAL, so the only thing that can turn the
		// journal on is the target's config.yaml.
		cmd.Env = envWithout(issuegraphEnv(dir), "ISSUEGRAPH_EVENTS_JOURNAL")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("issuegraph %s in %s: %v\n%s", strings.Join(args, " "), dir, err, out)
		}
		return string(out)
	}

	run(targetDir, "config", "set", "events-journal", "true")
	if _, err := os.Stat(filepath.Join(targetDir, ".issuegraph", "config.yaml")); err != nil {
		t.Fatalf("target config.yaml missing after `issuegraph config set`: %v", err)
	}

	// Route a create from the source workspace into the target.
	run(sourceDir, "create", "--silent", "--repo", targetDir, "routed work")

	targetRecords := decodeEventRecords(t, run(targetDir, "events", "export"))
	if len(targetRecords) == 0 {
		t.Fatalf("a routed create into a journal-enabled workspace recorded nothing — the cross-workspace store is not being activated from the TARGET's config")
	}
	if !hasEventOp(targetRecords, "create") {
		t.Errorf("target journal has no create record: %+v", targetRecords)
	}

	// The launching workspace never enabled the journal, so it records nothing
	// — the target's setting must not leak back, in either direction. Read
	// STDOUT only: a disabled workspace also prints the "journal is disabled"
	// notice, and that notice is on stderr precisely so it cannot be mistaken
	// for a record.
	sourceEnv := envWithout(issuegraphEnv(sourceDir), "ISSUEGRAPH_EVENTS_JOURNAL")
	stdout, _ := runIssueGraphOut(t, issuegraph, sourceDir, sourceEnv, "events", "export")
	if got := strings.TrimSpace(stdout); got != "" {
		t.Errorf("the launching workspace journaled %q despite never enabling the journal", got)
	}
}

// TestRoutedCreateHonorsTheEnvOverrideOverTheTargetsConfig is the precedence
// half. ISSUEGRAPH_EVENTS_JOURNAL is the documented way to turn the journal on for a
// process without editing a workspace, and a routed write is exactly where an
// operator reaches for it — so a target whose config.yaml says false must not
// be able to veto it. Getting this backwards makes the env var a suggestion,
// and the operator finds out by discovering an empty journal later.
func TestRoutedCreateHonorsTheEnvOverrideOverTheTargetsConfig(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()
	issuegraph := buildEmbeddedBD(t)

	sourceDir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "esrc", "--skip-hooks", "--skip-agents")
	targetDir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "etgt", "--skip-hooks", "--skip-agents")

	run := func(dir string, env []string, args ...string) string {
		t.Helper()
		cmd := exec.Command(issuegraph, args...)
		cmd.Dir = dir
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("issuegraph %s in %s: %v\n%s", strings.Join(args, " "), dir, err, out)
		}
		return string(out)
	}

	// The target explicitly turns the journal OFF in its own config.
	off := envWithout(issuegraphEnv(targetDir), "ISSUEGRAPH_EVENTS_JOURNAL")
	run(targetDir, off, "config", "set", "events-journal", "false")

	// The operator turns it on for this process anyway.
	on := append(envWithout(issuegraphEnv(sourceDir), "ISSUEGRAPH_EVENTS_JOURNAL"), "ISSUEGRAPH_EVENTS_JOURNAL=1")
	run(sourceDir, on, "create", "--silent", "--repo", targetDir, "routed under the env override")

	records := decodeEventRecords(t, run(targetDir, append(envWithout(issuegraphEnv(targetDir), "ISSUEGRAPH_EVENTS_JOURNAL"), "ISSUEGRAPH_EVENTS_JOURNAL=1"),
		"events", "export"))
	if !hasEventOp(records, "create") {
		t.Fatalf("ISSUEGRAPH_EVENTS_JOURNAL=1 did not override the target's `events-journal: false`; journal holds %+v", records)
	}
}

func hasEventOp(records []eventsjournal.Record, op string) bool {
	for _, r := range records {
		if r.Op == op {
			return true
		}
	}
	return false
}

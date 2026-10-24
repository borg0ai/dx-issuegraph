//go:build cgo

package main

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

type gateFrontDoorRunner struct {
	name string
	dir  string
	env  func(string) []string
}

func (r gateFrontDoorRunner) run(t *testing.T, issuegraph string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(issuegraph, args...)
	cmd.Dir = r.dir
	cmd.Env = r.env(r.dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	return stdout.String(), stderr.String(), err
}

func (r gateFrontDoorRunner) mustRun(t *testing.T, issuegraph string, args ...string) string {
	t.Helper()
	stdout, stderr, err := r.run(t, issuegraph, args...)
	if err != nil {
		t.Fatalf("[%s] issuegraph %s failed: %v\nstdout:\n%s\nstderr:\n%s",
			r.name, strings.Join(args, " "), err, stdout, stderr)
	}
	return stdout
}

// TestProxiedServerGateFrontDoorDirectServerProxyParity keeps gate lifecycle writes on the
// same UOW front in direct SQL and proxied-server modes. Gate IDs are generated
// independently, so assertions compare stable lifecycle and blocking facts.
func TestProxiedServerGateFrontDoorDirectServerProxyParity(t *testing.T) {
	requireSharedProxiedServer(t)
	issuegraph := buildEmbeddedBD(t)
	directProject := newServerModeProject(t, issuegraph, "mg")
	proxiedProject := newSharedProxiedProject(t, issuegraph, "mg")
	modes := []gateFrontDoorRunner{
		{name: "direct-sql", dir: directProject.dir, env: func(string) []string { return directProject.env }},
		{name: "proxied", dir: proxiedProject.dir, env: issuegraphProxiedEnv},
	}

	for _, r := range modes {
		r.mustRun(t, issuegraph, "create", "Gate target", "--id", "mg-target")
	}

	for _, r := range modes {
		created := r.mustRun(t, issuegraph, "gate", "create", "--type=human", "--blocks", "mg-target", "--reason", "review required")
		gateID := parseCreatedGateID(t, created)
		for _, want := range []string{"Blocks: mg-target", "Reason: review required", "Resolve with: issuegraph gate resolve " + gateID} {
			if !strings.Contains(created, want) {
				t.Errorf("[%s] gate create missing %q:\n%s", r.name, want, created)
			}
		}

		show := r.mustRun(t, issuegraph, "gate", "show", gateID)
		for _, want := range []string{gateID, "Status: open", "Await Type: human", "Reason: review required"} {
			if !strings.Contains(show, want) {
				t.Errorf("[%s] gate show missing %q:\n%s", r.name, want, show)
			}
		}

		// A waiter is persisted by the same gate UOW and remains visible in
		// list output before resolution.
		r.mustRun(t, issuegraph, "gate", "add-waiter", gateID, "worker1")
		var rows []map[string]any
		list := r.mustRun(t, issuegraph, "gate", "list", "--all", "--json")
		if err := json.Unmarshal([]byte(strings.TrimSpace(list)), &rows); err != nil {
			t.Fatalf("[%s] parse gate list: %v\n%s", r.name, err, list)
		}
		var row map[string]any
		for _, candidate := range rows {
			if candidate["id"] == gateID {
				row = candidate
				break
			}
		}
		if row == nil || row["status"] != "open" {
			t.Fatalf("[%s] gate list row = %v, want open gate %s", r.name, row, gateID)
		}
		waiters, ok := row["waiters"].([]any)
		if !ok || len(waiters) != 1 || waiters[0] != "worker1" {
			t.Errorf("[%s] gate list waiters = %v, want [worker1]", r.name, row["waiters"])
		}

		resolved := r.mustRun(t, issuegraph, "gate", "resolve", gateID, "--reason", "approved")
		if !strings.Contains(resolved, "Gate resolved: "+gateID) {
			t.Errorf("[%s] gate resolve output missing confirmation:\n%s", r.name, resolved)
		}
		closed := r.mustRun(t, issuegraph, "gate", "show", gateID)
		if !strings.Contains(closed, "Status: closed") {
			t.Errorf("[%s] gate remains open after resolve:\n%s", r.name, closed)
		}

		// `gate check` is the watcher-facing read/transaction boundary. A
		// resolved human gate is a clean no-op and must still exit successfully.
		checked := r.mustRun(t, issuegraph, "gate", "check", "--type", "human")
		const wantGateCheck = "No open gates of type 'human' found."
		if strings.TrimSpace(checked) != wantGateCheck {
			t.Errorf("[%s] gate check output = %q, want %q", r.name, checked, wantGateCheck)
		}

		// A issue gate exercises the transactional watcher path: once the
		// awaited issue closes, gate check must close the gate and report one
		// resolution (rather than merely observing an already-closed gate).
		r.mustRun(t, issuegraph, "create", "Awaited issue", "--id", "mg-awaited")
		issueCreated := r.mustRun(t, issuegraph, "gate", "create", "--type=issue", "--blocks", "mg-target", "--await-id", "mg-awaited", "--reason", "awaited")
		issueGateID := parseCreatedGateID(t, issueCreated)
		r.mustRun(t, issuegraph, "close", "mg-awaited", "--reason", "awaited issue complete")
		issueChecked := r.mustRun(t, issuegraph, "gate", "check", "--type", "issue")
		if !strings.Contains(strings.ToLower(issueChecked), "1 resolved") {
			t.Errorf("[%s] issue gate check output = %q, want one resolved gate", r.name, issueChecked)
		}
		issueShow := r.mustRun(t, issuegraph, "gate", "show", issueGateID)
		if !strings.Contains(issueShow, "Status: closed") {
			t.Errorf("[%s] issue gate %s remains open after check:\n%s", r.name, issueGateID, issueShow)
		}
	}
}

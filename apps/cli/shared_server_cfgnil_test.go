//go:build cgo

package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestSharedServerCfgNilHonorsSharedServer is a regression test for GH#3817.
//
// When the resolved issuegraphDir has no metadata.json/config.yaml, configfile.Load
// returns (nil, nil). The shared-server override in apps/cli/main.go used to live
// only inside the `cfg != nil` branch, so an exported ISSUEGRAPH_DOLT_SHARED_SERVER
// was silently ignored: doltCfg.ServerMode stayed false and newDoltStore fell
// through to embeddeddolt.Open. With a leftover embedded data directory present
// (e.g. a clone or recovery dir that has no metadata.json yet) issuegraph would read
// that phantom embedded DB and silently report success, fragmenting data away
// from the shared server instead of connecting to it.
//
// The fix honors shared-server mode regardless of whether project config was
// found. This test reproduces the config-less case with no reachable server and
// asserts issuegraph attempts the server (clear error) rather than silently falling
// back to the embedded database.
//
// Hermetic: no container required. Auto-start is disabled and the server port
// points nowhere, so the "honored" path fails fast with a connection error.
func TestSharedServerCfgNilHonorsSharedServer(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("not supported on Windows")
	}

	issuegraphBinary := buildSharedServerTestBinary(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// Config-less issuegraph dir (no metadata.json, no config.yaml) with a leftover
	// embedded data directory — the configfile.Load -> (nil, nil) case where an
	// embedded read could otherwise silently succeed.
	issuegraphDir := filepath.Join(t.TempDir(), ".issuegraph", "shared-server")
	if err := os.MkdirAll(filepath.Join(issuegraphDir, "dolt"), 0o755); err != nil {
		t.Fatalf("mkdir config-less issuegraph dir: %v", err)
	}

	env := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + t.TempDir(),
		"ISSUEGRAPH_DIR=" + issuegraphDir,
		"ISSUEGRAPH_DOLT_SHARED_SERVER=1",
		// Disable auto-start and point at a port nothing listens on so the
		// shared-server path fails fast instead of spinning up a server.
		"ISSUEGRAPH_DOLT_AUTO_START=0",
		"ISSUEGRAPH_DOLT_SERVER_PORT=59999",
		// Disable the metrics flusher: issuegraph otherwise spawns a DETACHED
		// `issuegraph send-metrics` child that keeps writing $HOME/.issuegraph/eventsData
		// after the parent exits. HOME here is a t.TempDir(), so that child
		// races Go's automatic RemoveAll and the test fails its own cleanup with
		// "unlinkat …: directory not empty" (matches the sibling shared-server
		// and protocol tests, which disable it for the same reason).
		"ISSUEGRAPH_DISABLE_EVENT_FLUSH=1",
		"ISSUEGRAPH_TEST_MODE=1",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=",
		"SSH_ASKPASS=",
		"GT_ROOT=",
	}

	neutralCwd := t.TempDir()
	out, err := ssExec(ctx, issuegraphBinary, neutralCwd, env, "list")

	// Before the fix: issuegraph silently ignored shared-server mode, read the leftover
	// embedded DB, and exited 0 with "No issues found". After: shared-server is
	// honored, so issuegraph tries the (unreachable) server and fails clearly.
	if err == nil {
		t.Fatalf("issuegraph list silently succeeded under ISSUEGRAPH_DOLT_SHARED_SERVER with no "+
			"reachable server and no project config; expected a server-connection "+
			"error — issuegraph fell through to the embedded backend (GH#3817 regression)\n"+
			"output:\n%s", out)
	}
	if strings.Contains(out, "No issues found") {
		t.Fatalf("issuegraph list returned embedded \"No issues found\" instead of honoring "+
			"shared-server mode (GH#3817 regression)\noutput:\n%s", out)
	}
	if !strings.Contains(out, "Dolt server") {
		t.Fatalf("expected a Dolt server connection error (shared-server mode honored), got:\n%s", out)
	}
}

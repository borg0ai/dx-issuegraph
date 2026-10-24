package issuegraph_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/borg0ai/dx-issuegraph"
	"github.com/borg0ai/dx-issuegraph/internal/workspacegate"
)

// OpenGated must fail fast (never reaching the storage open) while a
// maintenance operation holds the workspace gate exclusively, and must
// not leak a shared gate handle when the underlying storage open fails.
// Both paths are environment-independent: neither needs a working store.
func TestOpenGatedGateSemantics(t *testing.T) {
	dir := t.TempDir()
	issuegraphDir := filepath.Join(dir, ".issuegraph")
	if err := os.Mkdir(issuegraphDir, 0o755); err != nil {
		t.Fatal(err)
	}

	gate, err := workspacegate.ForWorkspace(issuegraphDir)
	if err != nil {
		t.Fatal(err)
	}

	// Exclusive holder (simulating a mode migration) blocks OpenGated,
	// matchable through the exported alias.
	h, err := gate.Acquire(context.Background(), workspacegate.Exclusive,
		workspacegate.Options{Reason: "test migration"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := issuegraph.OpenGated(context.Background(), issuegraphDir, 0); !errors.Is(err, issuegraph.ErrGateBusy) {
		t.Fatalf("OpenGated under exclusive gate: err = %v, want ErrGateBusy", err)
	}
	if err := h.Release(); err != nil {
		t.Fatal(err)
	}

	// Error path: a storage-open failure must release the shared gates
	// taken on the way in. A issuegraphDir that is a regular file cannot open
	// in any mode or environment.
	badDir := filepath.Join(dir, "bad")
	if err := os.Mkdir(badDir, 0o755); err != nil {
		t.Fatal(err)
	}
	badIssueGraph := filepath.Join(badDir, ".issuegraph")
	if err := os.WriteFile(badIssueGraph, []byte("not a dir"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := issuegraph.OpenGated(context.Background(), badIssueGraph, 0); err == nil {
		t.Fatal("OpenGated on a file-as-.issuegraph unexpectedly succeeded")
	}
	badGate, err := workspacegate.ForWorkspace(badIssueGraph)
	if err != nil {
		t.Fatal(err)
	}
	h3, err := badGate.Acquire(context.Background(), workspacegate.Exclusive, workspacegate.Options{})
	if err != nil {
		t.Fatalf("gate still held after failed OpenGated: %v", err)
	}
	_ = h3.Release()
}

// OpenGated must gate the LIBRARY open path's root, not only the
// CLI-parity resolver's: an embedded-metadata workspace is opened by the
// library path (server-only, DatabasePath-derived) at .issuegraph/dolt, so an
// exclusive holder on that root must block OpenGated even though the
// resolver alone would only gate .issuegraph/embeddeddolt. Environment
// independent: gate acquisition happens before the storage open.
func TestOpenGatedGatesLibraryRoot(t *testing.T) {
	dir := t.TempDir()
	issuegraphDir := filepath.Join(dir, ".issuegraph")
	if err := os.Mkdir(issuegraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(issuegraphDir, "metadata.json"),
		[]byte(`{"backend":"dolt","database":"issuegraph.db","dolt_mode":"embedded"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	// Neutralize any host central config: it must not be able to move the
	// library root out from under this assertion.
	t.Setenv("ISSUEGRAPH_CENTRAL_CONFIG", filepath.Join(dir, "no-central.json"))
	t.Setenv("ISSUEGRAPH_DOLT_DATA_DIR", "")

	libGate, err := workspacegate.ForPhysicalRoot(filepath.Join(issuegraphDir, "dolt"))
	if err != nil {
		t.Fatal(err)
	}
	h, err := libGate.Acquire(context.Background(), workspacegate.Exclusive,
		workspacegate.Options{Reason: "test holder on library-path root"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = h.Release() }()

	if _, err := issuegraph.OpenGated(context.Background(), issuegraphDir, 0); !errors.Is(err, issuegraph.ErrGateBusy) {
		t.Fatalf("OpenGated with EX holder on the library-path root: err = %v, want ErrGateBusy", err)
	}
}

// With a working store, OpenGated holds SHARED gates for the storage
// lifetime: other shared holders (a second cooperating consumer) coexist,
// an exclusive acquirer is excluded until Close, and the decorator must
// not amputate extended capabilities (the contract on AsIssueClaimer).
func TestOpenGatedLifecycle(t *testing.T) {
	skipIfNoDoltServer(t)

	tmpDir := t.TempDir()
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.Mkdir(issuegraphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	metadata := fmt.Sprintf(`{"backend":"dolt","database":"dolt","dolt_mode":"server","dolt_server_host":"127.0.0.1","dolt_server_port":%d}`, testServerPort)
	if err := os.WriteFile(filepath.Join(issuegraphDir, "metadata.json"), []byte(metadata), 0o644); err != nil {
		t.Fatal(err)
	}
	gate, err := workspacegate.ForWorkspace(issuegraphDir)
	if err != nil {
		t.Fatal(err)
	}

	st, err := issuegraph.OpenGated(context.Background(), issuegraphDir, 0)
	if err != nil {
		t.Fatalf("OpenGated: %v", err)
	}

	// Decorator contract: an embedder switching from OpenFromConfig to
	// OpenGated keeps every As*/interface assertion working.
	if _, ok := issuegraph.AsIssueClaimer(st); !ok {
		t.Error("OpenGated store lost IssueClaimer")
	}
	if _, ok := issuegraph.AsEventQuerier(st); !ok {
		t.Error("OpenGated store lost EventQuerier")
	}
	if _, ok := issuegraph.AsBlockedQuerier(st); !ok {
		t.Error("OpenGated store lost BlockedQuerier")
	}
	if _, ok := st.(issuegraph.RemoteStore); !ok {
		t.Error("OpenGated store lost RemoteStore")
	}
	if _, ok := st.(issuegraph.SyncStore); !ok {
		t.Error("OpenGated store lost SyncStore")
	}
	if _, ok := st.(issuegraph.VersionControlReader); !ok {
		t.Error("OpenGated store lost VersionControlReader")
	}

	// Shared holders coexist: a second cooperating consumer (equivalent to
	// a concurrent OpenGated) acquires the same workspace gate SHARED
	// while the first store is open.
	shared2, err := gate.Acquire(context.Background(), workspacegate.Shared, workspacegate.Options{})
	if err != nil {
		t.Fatalf("second shared holder blocked while OpenGated storage open: %v", err)
	}
	_ = shared2.Release()

	// The shared gate is held for the storage lifetime: exclusive
	// acquisition fails while open, succeeds after a clean Close.
	if _, err := gate.Acquire(context.Background(), workspacegate.Exclusive, workspacegate.Options{}); !errors.Is(err, issuegraph.ErrGateBusy) {
		t.Fatalf("exclusive while OpenGated storage open: err = %v, want ErrGateBusy", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	h2, err := gate.Acquire(context.Background(), workspacegate.Exclusive, workspacegate.Options{})
	if err != nil {
		t.Fatalf("gate still held after Close: %v", err)
	}
	_ = h2.Release()
}

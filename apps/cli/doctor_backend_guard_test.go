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

func TestDoctorRejectsCorruptMetadataBeforeSharedServerChecks(t *testing.T) {
	issuegraph := buildIssueGraphUnderTest(t)
	root := t.TempDir()
	issuegraphDir := filepath.Join(root, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0o700); err != nil {
		t.Fatalf("create .issuegraph: %v", err)
	}

	metadata := []byte("{\n")
	metadataPath := filepath.Join(issuegraphDir, "metadata.json")
	if err := os.WriteFile(metadataPath, metadata, 0o600); err != nil {
		t.Fatalf("write corrupt metadata.json: %v", err)
	}

	cmd := exec.Command(issuegraph, "doctor", "--json")
	cmd.Dir = root
	cmd.Env = append(removedBackendTestEnv(issuegraphDir),
		"ISSUEGRAPH_DOLT_SHARED_SERVER=1",
		"ISSUEGRAPH_DOLT_SERVER_PORT=1",
		"ISSUEGRAPH_DOLT_AUTO_START=0",
		"ISSUEGRAPH_NO_DAEMON=1",
		"ISSUEGRAPH_DISABLE_METRICS=1",
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("doctor unexpectedly ran shared-server checks with corrupt metadata:\n%s", out)
	}

	message := strings.ToLower(string(out))
	for _, want := range []string{"metadata.json", "parsing config", "no storage database was opened or modified"} {
		if !strings.Contains(message, want) {
			t.Errorf("doctor corrupt-metadata error missing %q:\n%s", want, out)
		}
	}

	after, readErr := os.ReadFile(metadataPath)
	if readErr != nil {
		t.Fatalf("read metadata after refusal: %v", readErr)
	}
	if !bytes.Equal(after, metadata) {
		t.Fatalf("doctor rewrote corrupt metadata:\nbefore: %q\nafter:  %q", metadata, after)
	}
	for _, name := range []string{".local_version", "embeddeddolt", "dolt", "issuegraph.db", "dolt-server.pid", "dolt-server.port"} {
		if _, statErr := os.Stat(filepath.Join(issuegraphDir, name)); !os.IsNotExist(statErr) {
			t.Fatalf("doctor created %s before rejecting corrupt metadata (stat error: %v)", name, statErr)
		}
	}
}

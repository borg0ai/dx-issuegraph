//go:build !cgo

package issuegraph_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph"
)

func TestOpenBestAvailable_NoCGO_EmbeddedMode_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatalf("failed to create .issuegraph dir: %v", err)
	}

	metadata := `{"backend":"dolt","database":"dolt","dolt_mode":"embedded"}`
	if err := os.WriteFile(filepath.Join(issuegraphDir, "metadata.json"), []byte(metadata), 0644); err != nil {
		t.Fatalf("failed to write metadata.json: %v", err)
	}

	ctx := context.Background()
	_, err := issuegraph.OpenBestAvailable(ctx, issuegraphDir)
	if err == nil {
		t.Fatal("expected error for embedded mode without CGO")
	}
	if !strings.Contains(err.Error(), "CGO") {
		t.Errorf("expected error to mention CGO, got: %v", err)
	}
}

func TestOpenBestAvailable_NoCGO_NoMetadata_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatalf("failed to create .issuegraph dir: %v", err)
	}
	// No metadata.json — embedded is the default, CGO required.

	ctx := context.Background()
	_, err := issuegraph.OpenBestAvailable(ctx, issuegraphDir)
	if err == nil {
		t.Fatal("expected error for embedded mode without CGO")
	}
	if !strings.Contains(err.Error(), "CGO") {
		t.Errorf("expected error to mention CGO, got: %v", err)
	}
}

func TestOpenBestAvailable_NoCGO_ServerMode_FailsWithoutServer(t *testing.T) {
	// Even in !cgo builds, server mode should delegate to OpenFromConfig and
	// return the fail-fast error when no server is listening.
	if prev := os.Getenv("ISSUEGRAPH_DOLT_PORT"); prev != "" {
		os.Unsetenv("ISSUEGRAPH_DOLT_PORT")
		t.Cleanup(func() { os.Setenv("ISSUEGRAPH_DOLT_PORT", prev) })
	}
	if prev := os.Getenv("ISSUEGRAPH_TEST_MODE"); prev != "" {
		os.Unsetenv("ISSUEGRAPH_TEST_MODE")
		t.Cleanup(func() { os.Setenv("ISSUEGRAPH_TEST_MODE", prev) })
	}

	tmpDir := t.TempDir()
	issuegraphDir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphDir, 0755); err != nil {
		t.Fatalf("failed to create .issuegraph dir: %v", err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	freePort := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	metadata := fmt.Sprintf(`{"backend":"dolt","database":"dolt","dolt_mode":"server","dolt_server_host":"127.0.0.1","dolt_server_port":%d}`, freePort)
	if err := os.WriteFile(filepath.Join(issuegraphDir, "metadata.json"), []byte(metadata), 0644); err != nil {
		t.Fatalf("failed to write metadata.json: %v", err)
	}

	ctx := context.Background()
	_, openErr := issuegraph.OpenBestAvailable(ctx, issuegraphDir)
	if openErr == nil {
		t.Fatal("OpenBestAvailable (server mode) should fail when no server is running")
	}
	if !strings.Contains(openErr.Error(), "unreachable") {
		t.Errorf("expected 'unreachable' in error, got: %v", openErr)
	}
}

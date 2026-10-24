//go:build cgo

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/config"
	"github.com/borg0ai/dx-issuegraph/internal/testutil"
)

func TestDoltRemoteAddPersistsSyncRemoteToSharedWorktreeConfig(t *testing.T) {
	skipIfNoDolt(t)
	if runtime.GOOS == "windows" {
		t.Skip("Skipping worktree test on Windows")
	}

	issuegraph := buildIssueGraphForInitTests(t)
	bareDir, worktreeDir := setupBareParentInitWorktree(t)
	bareIssueGraphDir := filepath.Join(bareDir, ".issuegraph")
	port, err := testutil.FindFreePort()
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	sharedEnv := append(os.Environ(),
		"ISSUEGRAPH_DOLT_SHARED_SERVER=1",
		"ISSUEGRAPH_DOLT_SERVER_PORT="+strconv.Itoa(port),
	)

	// `issuegraph init` in shared-server mode DAEMONIZES a dolt sql-server under
	// $HOME/.issuegraph/shared-server and returns; nothing here used to stop it,
	// so it outlived the suite that deleted its tree (wy-j2zc8q). Registered
	// before the first subprocess so the t.Fatalf paths are covered too.
	stopSharedServerCleanup(t)

	initCmd := exec.Command(issuegraph, "init", "--prefix", "remote-sync", "--skip-hooks", "--quiet")
	initCmd.Dir = worktreeDir
	initCmd.Env = sharedEnv
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("issuegraph init from bare-parent worktree failed: %v\n%s", err, out)
	}

	remoteURL := "git+ssh://git@example.com/acme/issuegraph.git"
	addCmd := exec.Command(issuegraph, "dolt", "remote", "add", "origin", remoteURL)
	addCmd.Dir = worktreeDir
	addCmd.Env = sharedEnv
	if out, err := addCmd.CombinedOutput(); err != nil {
		t.Fatalf("issuegraph dolt remote add from bare-parent worktree failed: %v\n%s", err, out)
	}

	// Asserted by reading it back, not by matching YAML text. The literal
	// `sync.remote: "<url>"` this used to require is a key whose NAME contains
	// a dot — the shape config.GetStringFromDir can never find, because it
	// splits on the dot and walks nested mappings. Viper finds a dotted key
	// either way, so this assertion was green while the value was invisible to
	// issuegraph's other reader; it passed BECAUSE bd-zj95 was present. What the remote
	// add owes its caller is a remote that reads back, so that is what this
	// checks.
	configPath := filepath.Join(bareIssueGraphDir, "config.yaml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read shared config.yaml: %v", err)
	}
	if got := config.GetStringFromDir(bareIssueGraphDir, "sync.remote"); got != remoteURL {
		t.Fatalf("shared config.yaml sync.remote = %q, want %q; contents:\n%s", got, remoteURL, content)
	}

	if _, err := os.Stat(filepath.Join(worktreeDir, ".issuegraph")); !os.IsNotExist(err) {
		t.Fatalf("expected no worktree-local .issuegraph directory after remote add, got err=%v", err)
	}
}

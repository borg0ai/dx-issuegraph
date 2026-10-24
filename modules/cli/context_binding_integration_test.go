//go:build cgo

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/storage/dolt"
	"github.com/borg0ai/dx-issuegraph/internal/types"
)

func filteredEnvForContextBinding(keys ...string) []string {
	strip := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		strip[key+"="] = struct{}{}
	}

	env := os.Environ()
	filtered := make([]string, 0, len(env))
	for _, entry := range env {
		if strings.HasPrefix(entry, "ISSUEGRAPH_") || strings.HasPrefix(entry, "ISSUEGRAPH_") {
			continue
		}
		trim := false
		for prefix := range strip {
			if strings.HasPrefix(entry, prefix) {
				trim = true
				break
			}
		}
		if !trim {
			filtered = append(filtered, entry)
		}
	}
	return append(filtered, "ISSUEGRAPH_DISABLE_METRICS=1", "ISSUEGRAPH_DISABLE_EVENT_FLUSH=1")
}

func TestListExplicitDBPathRebindsTargetContext(t *testing.T) {
	if testDoltServerPort == 0 {
		t.Skip("Dolt test server not available, skipping")
	}

	tmpDir := t.TempDir()
	callerRepo := filepath.Join(tmpDir, "caller")
	callerIssueGraphDir := filepath.Join(callerRepo, ".issuegraph")
	writeTestConfigYAML(t, callerIssueGraphDir, "dolt.auto-commit: invalid\nactor: caller-actor\n")
	if err := os.WriteFile(filepath.Join(callerIssueGraphDir, ".env"), []byte("ISSUEGRAPH_DOLT_SERVER_PORT=1\n"), 0o600); err != nil {
		t.Fatalf("write caller .env: %v", err)
	}

	targetRepo := filepath.Join(tmpDir, "target")
	targetIssueGraphDir := filepath.Join(targetRepo, ".issuegraph")
	writeTestConfigYAML(t, targetIssueGraphDir, "dolt.auto-commit: off\nactor: target-actor\n")
	database := uniqueTestDBName(t)
	if err := (&configfile.Config{
		Backend:        configfile.BackendDolt,
		DoltMode:       configfile.DoltModeServer,
		DoltServerHost: "127.0.0.1",
		DoltServerPort: testDoltServerPort,
		DoltDatabase:   database,
	}).Save(targetIssueGraphDir); err != nil {
		t.Fatalf("save target metadata: %v", err)
	}

	ctx := context.Background()
	testStore, err := dolt.New(ctx, &dolt.Config{
		Path:            filepath.Join(targetIssueGraphDir, "dolt"),
		IssueGraphDir:   targetIssueGraphDir,
		ServerHost:      "127.0.0.1",
		ServerPort:      testDoltServerPort,
		Database:        database,
		CreateIfMissing: true,
	})
	if err != nil {
		t.Fatalf("create test store: %v", err)
	}
	defer func() {
		_ = testStore.Close()
		dropTestDatabase(database, testDoltServerPort)
	}()
	if err := testStore.SetConfig(ctx, "issue_prefix", "ctx"); err != nil {
		t.Fatalf("set issue_prefix: %v", err)
	}
	now := time.Now()
	nowIssue := &types.Issue{
		ID:          "ctx-1",
		Title:       "Context binding proof",
		Description: "Proves explicit --db commands use the target workspace config",
		Status:      types.StatusOpen,
		Priority:    1,
		IssueType:   types.TypeTask,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := testStore.CreateIssue(ctx, nowIssue, "test-user"); err != nil {
		t.Fatalf("create issue: %v", err)
	}

	// Shared once-per-process binary (honors ISSUEGRAPH_TEST_ISSUEGRAPH_BINARY) instead of
	// a per-test go build — the in-test link steps dominated modules/cli's wall
	// clock (wy-4mtr0).
	binPath := buildIssueGraphForInitTests(t)

	listCmd := exec.Command(binPath, "list", "--db", filepath.Join(targetIssueGraphDir, "dolt"), "--json")
	listCmd.Dir = callerRepo
	listCmd.Env = append(filteredEnvForContextBinding("ISSUEGRAPH_DIR", "ISSUEGRAPH_DB", "ISSUEGRAPH_DB", "ISSUEGRAPH_DOLT_SERVER_PORT", "ISSUEGRAPH_DOLT_SERVER_DATABASE"),
		"HOME="+t.TempDir(),
		"XDG_CONFIG_HOME="+t.TempDir(),
		"ISSUEGRAPH_TEST_MODE=1",
		// AD-01 (be-c5p): the subprocess connects to a testdb_*-named DB on
		// the test container; pass the firewall opt-in so dolt.New allows it.
		"ISSUEGRAPH_TEST_SERVER=1",
		"ISSUEGRAPH_DIR="+callerIssueGraphDir,
		"ISSUEGRAPH_DB=",
	)
	output, err := listCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("issuegraph list failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "Context binding proof") {
		t.Fatalf("expected list output to include target issue\n%s", output)
	}

	if _, err := os.Stat(filepath.Join(callerIssueGraphDir, localVersionFile)); err == nil {
		t.Fatalf("caller workspace unexpectedly created %s", filepath.Join(callerIssueGraphDir, localVersionFile))
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat caller %s: %v", localVersionFile, err)
	}
	if _, err := os.Stat(filepath.Join(targetIssueGraphDir, localVersionFile)); err != nil {
		t.Fatalf("target workspace should create %s: %v", localVersionFile, err)
	}
}

//go:build cgo

// be-pc8c: Server-mode behavioral tests for ISSUEGRAPH_MAX_ROWS on
// `issuegraph doctor --check=conventions` and `issuegraph doctor --check=pollution`.
//
// Both commands call resolveMaxRowsEnvOnly() and pipe through
// handleMaxRowsError, so ISSUEGRAPH_MAX_ROWS in the environment should fire
// ErrTooManyRows → os.Exit(2) when the cap is exceeded. Unlike the
// embedded-dolt tests in max_rows_test.go these require the Dolt test server
// (testDoltServerPort != 0) because issuegraph doctor refuses to run in embedded mode.
//
// Each test:
//   - Creates a temp dir with a .issuegraph/metadata.json pointing at the test server
//   - Seeds 6 open issues (cap=3, so the cap fires)
//   - Runs `issuegraph doctor --check=<X>` via exec.Command with ISSUEGRAPH_MAX_ROWS=3
//   - Asserts exit code 2 and ISSUEGRAPH_MAX_ROWS=3 in combined output

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/testutil"
)

func TestDoctorConventionsMaxRows_EnvOnly_Exits2(t *testing.T) {
	if testDoltServerPort == 0 {
		t.Skip("Dolt test server not available")
	}
	if testutil.DoltContainerCrashed() {
		t.Skipf("Dolt test server crashed: %v", testutil.DoltContainerCrashError())
	}

	issuegraphBin := buildIssueGraphForInitTests(t)

	tmpDir := t.TempDir()
	issuegraphSubdir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphSubdir, 0755); err != nil {
		t.Fatal(err)
	}

	// newTestStoreIsolatedDB creates a dedicated Dolt database and writes
	// metadata.json into .issuegraph/ so the issuegraph subprocess opens the same DB.
	s := newTestStoreIsolatedDB(t, filepath.Join(issuegraphSubdir, "dolt"), "dcmr")
	ctx := context.Background()

	for i := range 6 {
		id := "dcmr-" + string(rune('a'+i))
		if _, err := s.DB().ExecContext(ctx,
			`INSERT INTO issues (id, title, description, design, acceptance_criteria, notes, status, priority, issue_type) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, "conventions max-rows seed", "", "", "", "", "open", 1, "task",
		); err != nil {
			t.Fatalf("seed issue %d: %v", i, err)
		}
	}

	// ISSUEGRAPH_MAX_ROWS=3 with 6 open issues → cap fires → exit 2.
	out, code := issuegraphRunRaw(t, issuegraphBin, tmpDir, []string{"ISSUEGRAPH_MAX_ROWS=3"}, "doctor", "--check=conventions")
	if code != 2 {
		t.Fatalf("be-pc8c: expected exit 2 (cap exceeded), got %d\n%s", code, out)
	}
	if !strings.Contains(out, "ISSUEGRAPH_MAX_ROWS=3") {
		t.Errorf("be-pc8c: stderr missing ISSUEGRAPH_MAX_ROWS=3 source attribution:\n%s", out)
	}
}

func TestDoctorPollutionMaxRows_EnvOnly_Exits2(t *testing.T) {
	if testDoltServerPort == 0 {
		t.Skip("Dolt test server not available")
	}
	if testutil.DoltContainerCrashed() {
		t.Skipf("Dolt test server crashed: %v", testutil.DoltContainerCrashError())
	}

	issuegraphBin := buildIssueGraphForInitTests(t)

	tmpDir := t.TempDir()
	issuegraphSubdir := filepath.Join(tmpDir, ".issuegraph")
	if err := os.MkdirAll(issuegraphSubdir, 0755); err != nil {
		t.Fatal(err)
	}

	s := newTestStoreIsolatedDB(t, filepath.Join(issuegraphSubdir, "dolt"), "dpmr")
	ctx := context.Background()

	for i := range 6 {
		id := "dpmr-" + string(rune('a'+i))
		if _, err := s.DB().ExecContext(ctx,
			`INSERT INTO issues (id, title, description, design, acceptance_criteria, notes, status, priority, issue_type) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, "pollution max-rows seed", "", "", "", "", "open", 1, "task",
		); err != nil {
			t.Fatalf("seed issue %d: %v", i, err)
		}
	}

	out, code := issuegraphRunRaw(t, issuegraphBin, tmpDir, []string{"ISSUEGRAPH_MAX_ROWS=3"}, "doctor", "--check=pollution")
	if code != 2 {
		t.Fatalf("be-pc8c: expected exit 2 (cap exceeded), got %d\n%s", code, out)
	}
	if !strings.Contains(out, "ISSUEGRAPH_MAX_ROWS=3") {
		t.Errorf("be-pc8c: stderr missing ISSUEGRAPH_MAX_ROWS=3 source attribution:\n%s", out)
	}
}

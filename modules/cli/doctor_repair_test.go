package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func buildIssueGraphForTest(t *testing.T) string {
	t.Helper()
	if prebuilt, err := findPrebuiltIssueGraphBinary(); err != nil {
		t.Fatal(err)
	} else if prebuilt != "" {
		return prebuilt
	}
	exeName := "issuegraph"
	if runtime.GOOS == "windows" {
		exeName = "issuegraph.exe"
	}

	binDir := t.TempDir()
	exe := filepath.Join(binDir, exeName)
	cmd := exec.Command("go", "build", "-tags", "gms_pure_go", "-o", exe, ".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, string(out))
	}
	return exe
}

func mkTmpDirInTmp(t *testing.T, prefix string) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", prefix)
	if err != nil {
		// Fallback for platforms without /tmp (e.g. Windows).
		dir, err = os.MkdirTemp("", prefix)
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func runIssueGraphSideDB(t *testing.T, exe, dir, dbPath string, args ...string) (string, error) {
	t.Helper()
	fullArgs := []string{"--db", dbPath}
	fullArgs = append(fullArgs, args...)

	cmd := exec.Command(exe, fullArgs...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"ISSUEGRAPH_DIR="+filepath.Join(dir, ".issuegraph"),
	)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestDoctorRepair_CorruptDatabase_RebuildFromJSONL(t *testing.T) {
	// SQLite file corruption repair test. Dolt backend uses server connections,
	// not .db files, so corruption/repair scenarios are fundamentally different.
	t.Skip("SQLite file corruption repair; not applicable to Dolt backend (bd-o0u)")
}

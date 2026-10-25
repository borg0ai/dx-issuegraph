package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestIsNonInteractiveInit tests the non-interactive detection logic.
func TestIsNonInteractiveInit(t *testing.T) {
	// Save original env vars and restore after tests
	origCI := os.Getenv("CI")
	origIssueGraphNI := os.Getenv("ISSUEGRAPH_NON_INTERACTIVE")
	defer func() {
		os.Setenv("CI", origCI)
		os.Setenv("ISSUEGRAPH_NON_INTERACTIVE", origIssueGraphNI)
	}()

	tests := []struct {
		name            string
		flagValue       bool
		envCI           string
		envIssueGraphNI string
		want            bool
	}{
		{
			name:            "flag true overrides everything",
			flagValue:       true,
			envCI:           "",
			envIssueGraphNI: "",
			want:            true,
		},
		{
			name:            "ISSUEGRAPH_NON_INTERACTIVE=1",
			flagValue:       false,
			envCI:           "",
			envIssueGraphNI: "1",
			want:            true,
		},
		{
			name:            "ISSUEGRAPH_NON_INTERACTIVE=true",
			flagValue:       false,
			envCI:           "",
			envIssueGraphNI: "true",
			want:            true,
		},
		{
			name:            "CI=true",
			flagValue:       false,
			envCI:           "true",
			envIssueGraphNI: "",
			want:            true,
		},
		{
			name:            "CI=1",
			flagValue:       false,
			envCI:           "1",
			envIssueGraphNI: "",
			want:            true,
		},
		{
			name:            "CI=false does not trigger",
			flagValue:       false,
			envCI:           "false",
			envIssueGraphNI: "",
			// In test env, stdin is not a TTY, so this is still true
			want: true,
		},
		{
			name:            "no flag no env falls back to terminal detection",
			flagValue:       false,
			envCI:           "",
			envIssueGraphNI: "",
			// In test environment, stdin is piped (not a TTY), so non-interactive
			want: true,
		},
		{
			name:            "ISSUEGRAPH_NON_INTERACTIVE=0 forces interactive",
			flagValue:       false,
			envCI:           "true",
			envIssueGraphNI: "0",
			want:            false,
		},
		{
			name:            "ISSUEGRAPH_NON_INTERACTIVE=false forces interactive",
			flagValue:       false,
			envCI:           "true",
			envIssueGraphNI: "false",
			want:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("CI", tt.envCI)
			os.Setenv("ISSUEGRAPH_NON_INTERACTIVE", tt.envIssueGraphNI)

			got := isNonInteractiveInit(tt.flagValue)
			if got != tt.want {
				t.Errorf("isNonInteractiveInit(%v) with CI=%q ISSUEGRAPH_NON_INTERACTIVE=%q = %v, want %v",
					tt.flagValue, tt.envCI, tt.envIssueGraphNI, got, tt.want)
			}
		})
	}
}

// TestIsNonInteractiveInitPrecedence tests that flag takes precedence over env vars.
func TestIsNonInteractiveInitPrecedence(t *testing.T) {
	origCI := os.Getenv("CI")
	origIssueGraphNI := os.Getenv("ISSUEGRAPH_NON_INTERACTIVE")
	defer func() {
		os.Setenv("CI", origCI)
		os.Setenv("ISSUEGRAPH_NON_INTERACTIVE", origIssueGraphNI)
	}()

	// Flag true should always win
	os.Setenv("CI", "")
	os.Setenv("ISSUEGRAPH_NON_INTERACTIVE", "")
	if !isNonInteractiveInit(true) {
		t.Error("flag=true should always return true regardless of env")
	}

	// ISSUEGRAPH_NON_INTERACTIVE should take precedence over CI
	os.Setenv("ISSUEGRAPH_NON_INTERACTIVE", "1")
	os.Setenv("CI", "")
	if !isNonInteractiveInit(false) {
		t.Error("ISSUEGRAPH_NON_INTERACTIVE=1 should return true")
	}
}

func TestInitNonInteractiveSnapshotDefaultOnAndOptOut(t *testing.T) {
	issuegraph := buildIssueGraphForInitTests(t)
	dir := t.TempDir()

	runIssueGraphForAutoExportInitTest(t, issuegraph, dir, "init", "--prefix", "test", "--quiet", "--non-interactive", "--skip-hooks", "--skip-agents")

	if got := strings.TrimSpace(runIssueGraphStdoutForAutoExportInitTest(t, issuegraph, dir, "config", "get", "export.auto")); got != "true" {
		t.Fatalf("export.auto default = %q, want true", got)
	}
	if got := strings.TrimSpace(runIssueGraphStdoutForAutoExportInitTest(t, issuegraph, dir, "config", "get", "export.git-add")); got != "false" {
		t.Fatalf("export.git-add default = %q, want false", got)
	}

	runIssueGraphForAutoExportInitTest(t, issuegraph, dir, "create", "snapshot issue", "-p", "2")
	jsonlPath := filepath.Join(dir, ".issuegraph", "issues.jsonl")
	data, err := os.ReadFile(jsonlPath)
	if err != nil {
		t.Fatalf("default create did not write %s: %v", jsonlPath, err)
	}
	if !strings.Contains(string(data), "snapshot issue") {
		t.Fatalf("snapshot JSONL did not contain created issue:\n%s", data)
	}
	if _, err := os.Stat(filepath.Join(dir, ".issuegraph", snapshotManifestFile)); err != nil {
		t.Fatalf("default create did not write the manifest: %v", err)
	}

	runIssueGraphForAutoExportInitTest(t, issuegraph, dir, "config", "set", "export.auto", "false")
	runIssueGraphForAutoExportInitTest(t, issuegraph, dir, "create", "opted-out issue", "-p", "2")
	data, err = os.ReadFile(jsonlPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "opted-out issue") {
		t.Fatalf("export.auto=false still wrote the snapshot:\n%s", data)
	}
}

func runIssueGraphStdoutForAutoExportInitTest(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command(issuegraph, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "ISSUEGRAPH_NON_INTERACTIVE=1")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("issuegraph %v failed: %v", args, err)
	}
	return string(out)
}

func runIssueGraphForAutoExportInitTest(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command(issuegraph, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "ISSUEGRAPH_NON_INTERACTIVE=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("issuegraph %v failed: %v\n%s", args, err, out)
	}
	return string(out)
}

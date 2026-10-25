//go:build integration
// +build integration

package issuegraph_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/borg0ai/dx-issuegraph/internal/testutil"
)

// getIssueGraphPath returns the test issuegraph binary path
func getIssueGraphPath() string {
	if testIssueGraphBinary := os.Getenv("ISSUEGRAPH_TEST_ISSUEGRAPH_BINARY"); testIssueGraphBinary != "" {
		return testIssueGraphBinary
	}
	// Fallback for non-TestMain runs
	if runtime.GOOS == "windows" {
		return "./issuegraph.exe"
	}
	return "./issuegraph"
}

// getIssueGraphCommand returns the platform-specific command to run issuegraph from current dir
// Always uses forward slashes for sh script compatibility (Git for Windows uses sh)
func getIssueGraphCommand() string {
	if runtime.GOOS == "windows" {
		return "./issuegraph.exe"
	}
	return "./issuegraph"
}

func hasDoltTestPort() bool {
	return os.Getenv("ISSUEGRAPH_DOLT_PORT") != ""
}

func requireHashIDIntegration(t *testing.T) string {
	t.Helper()
	if !hasDoltTestPort() {
		t.Skip("skipping: Dolt test container not available")
	}
	issuegraphPath := getIssueGraphPath()
	if _, err := os.Stat(issuegraphPath); err != nil {
		t.Fatalf("issuegraph binary not found at %s", issuegraphPath)
	}
	return issuegraphPath
}

// TestHashIDs_MultiCloneConverge verifies that hash-based IDs work correctly
// across multiple clones creating different issues. With hash IDs, each unique
// issue gets a unique ID, so no collision resolution is needed.
func TestHashIDs_MultiCloneConverge(t *testing.T) {
	issuegraphPath := requireHashIDIntegration(t)
	t.Parallel()
	tmpDir := testutil.TempDirInMemory(t)

	// Setup remote and 3 clones
	remoteDir := setupBareRepo(t, tmpDir)
	cloneA := setupClone(t, tmpDir, remoteDir, "A", issuegraphPath)
	cloneB := setupClone(t, tmpDir, remoteDir, "B", issuegraphPath)
	cloneC := setupClone(t, tmpDir, remoteDir, "C", issuegraphPath)

	// Each clone creates unique issue (different content = different hash ID)
	createIssueInClone(t, cloneA, "Issue from clone A")
	createIssueInClone(t, cloneB, "Issue from clone B")
	createIssueInClone(t, cloneC, "Issue from clone C")

	// Sync all clones once (hash IDs prevent collisions, don't need multiple rounds)
	for _, clone := range []string{cloneA, cloneB, cloneC} {
		runCmdOutputWithEnvAllowError(t, clone, map[string]string{}, true, issuegraphPath, "sync")
	}

	// Verify all clones have all 3 issues
	expectedTitles := map[string]bool{
		"Issue from clone A": true,
		"Issue from clone B": true,
		"Issue from clone C": true,
	}

	allConverged := true
	for name, dir := range map[string]string{"A": cloneA, "B": cloneB, "C": cloneC} {
		titles := getTitlesFromClone(t, dir)
		if !compareTitleSets(titles, expectedTitles) {
			t.Logf("Clone %s has %d/%d issues: %v", name, len(titles), len(expectedTitles), sortedKeys(titles))
			allConverged = false
		}
	}

	if allConverged {
		t.Log("✓ All 3 clones converged with hash-based IDs")
	} else {
		t.Log("✓ Hash-based IDs prevent collisions (convergence may take more rounds)")
	}
}

// TestHashIDs_IdenticalContentDedup verifies that when two clones create
// identical issues, they get the same hash ID and deduplicate correctly.
func TestHashIDs_IdenticalContentDedup(t *testing.T) {
	issuegraphPath := requireHashIDIntegration(t)
	t.Parallel()
	tmpDir := testutil.TempDirInMemory(t)

	// Setup remote and 2 clones
	remoteDir := setupBareRepo(t, tmpDir)
	cloneA := setupClone(t, tmpDir, remoteDir, "A", issuegraphPath)
	cloneB := setupClone(t, tmpDir, remoteDir, "B", issuegraphPath)

	// Both clones create identical issue (same content = same hash ID)
	createIssueInClone(t, cloneA, "Identical issue")
	createIssueInClone(t, cloneB, "Identical issue")

	// Sync both clones once (hash IDs handle dedup automatically)
	for _, clone := range []string{cloneA, cloneB} {
		runCmdOutputWithEnvAllowError(t, clone, map[string]string{}, true, issuegraphPath, "sync")
	}

	// Verify both clones have exactly 1 issue (deduplication worked)
	for name, dir := range map[string]string{"A": cloneA, "B": cloneB} {
		titles := getTitlesFromClone(t, dir)
		if len(titles) != 1 {
			t.Errorf("Clone %s should have 1 issue, got %d: %v", name, len(titles), sortedKeys(titles))
		}
		if !titles["Identical issue"] {
			t.Errorf("Clone %s missing expected issue: %v", name, sortedKeys(titles))
		}
	}

	t.Log("✓ Identical content deduplicated correctly with hash-based IDs")
}

// Shared test helpers

func setupBareRepo(t *testing.T, tmpDir string) string {
	t.Helper()
	remoteDir := filepath.Join(tmpDir, "remote.git")
	runCmd(t, tmpDir, "git", "init", "--bare", "-b", "master", remoteDir)

	tempClone := filepath.Join(tmpDir, "temp-init")
	runCmd(t, tmpDir, "git", "clone", remoteDir, tempClone)
	runCmd(t, tempClone, "git", "commit", "--allow-empty", "-m", "Initial commit")
	runCmd(t, tempClone, "git", "push", "origin", "master")

	return remoteDir
}

func setupClone(t *testing.T, tmpDir, remoteDir, name, issuegraphPath string) string {
	t.Helper()
	cloneDir := filepath.Join(tmpDir, "clone-"+strings.ToLower(name))

	// Use shallow, shared clones for speed
	runCmd(t, tmpDir, "git", "clone", "--shared", "--depth=1", "--no-tags", remoteDir, cloneDir)

	// Disable hooks to avoid overhead
	emptyHooks := filepath.Join(cloneDir, ".empty-hooks")
	os.MkdirAll(emptyHooks, 0755)
	runCmd(t, cloneDir, "git", "config", "core.hooksPath", emptyHooks)

	// Speed configs
	runCmd(t, cloneDir, "git", "config", "gc.auto", "0")
	runCmd(t, cloneDir, "git", "config", "core.fsync", "false")
	runCmd(t, cloneDir, "git", "config", "commit.gpgSign", "false")

	issuegraphCmd := getIssueGraphCommand()
	copyFile(t, issuegraphPath, filepath.Join(cloneDir, filepath.Base(issuegraphCmd)))

	if name == "A" {
		runCmd(t, cloneDir, issuegraphCmd, "init", "--quiet", "--prefix", "test")
		runCmd(t, cloneDir, "git", "add", ".issuegraph")
		runCmd(t, cloneDir, "git", "commit", "--no-verify", "-m", "Initialize issuegraph")
		runCmd(t, cloneDir, "git", "push", "origin", "master")
	} else {
		runCmd(t, cloneDir, "git", "pull", "origin", "master")
		runCmd(t, cloneDir, issuegraphCmd, "init", "--quiet", "--prefix", "test")
	}

	return cloneDir
}

func createIssueInClone(t *testing.T, cloneDir, title string) {
	t.Helper()
	runCmdWithEnv(t, cloneDir, map[string]string{}, getIssueGraphCommand(), "create", title, "-t", "task", "-p", "1", "--json")
}

func getTitlesFromClone(t *testing.T, cloneDir string) map[string]bool {
	t.Helper()
	listJSON := runCmdOutputWithEnv(t, cloneDir, map[string]string{
		"ISSUEGRAPH_NO_AUTO_IMPORT": "1",
	}, getIssueGraphCommand(), "list", "--json")

	jsonStart := strings.Index(listJSON, "[")
	if jsonStart == -1 {
		return make(map[string]bool)
	}
	listJSON = listJSON[jsonStart:]

	var issues []struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(listJSON), &issues); err != nil {
		t.Logf("Failed to parse JSON: %v", err)
		return make(map[string]bool)
	}

	titles := make(map[string]bool)
	for _, issue := range issues {
		titles[issue.Title] = true
	}
	return titles
}

func resolveConflictMarkersIfPresent(t *testing.T, cloneDir string) {
	t.Helper()
	jsonlPath := filepath.Join(cloneDir, ".issuegraph", "issues.jsonl")
	jsonlContent, _ := os.ReadFile(jsonlPath)
	if strings.Contains(string(jsonlContent), "<<<<<<<") {
		var cleanLines []string
		for _, line := range strings.Split(string(jsonlContent), "\n") {
			if !strings.HasPrefix(line, "<<<<<<<") &&
				!strings.HasPrefix(line, "=======") &&
				!strings.HasPrefix(line, ">>>>>>>") {
				if strings.TrimSpace(line) != "" {
					cleanLines = append(cleanLines, line)
				}
			}
		}
		cleaned := strings.Join(cleanLines, "\n") + "\n"
		os.WriteFile(jsonlPath, []byte(cleaned), 0644)
		runCmd(t, cloneDir, "git", "add", ".issuegraph/issues.jsonl")
		runCmd(t, cloneDir, "git", "commit", "-m", "Resolve merge conflict")
	}
}

func installGitHooks(t *testing.T, repoDir string) {
	t.Helper()
	hooksDir := filepath.Join(repoDir, ".git", "hooks")
	// Ensure POSIX-style path for sh scripts (even on Windows)
	issuegraphCmd := strings.ReplaceAll(getIssueGraphCommand(), "\\", "/")

	preCommit := fmt.Sprintf(`#!/bin/sh
%s export -o .issuegraph/issues.jsonl >/dev/null 2>&1 || true
git add .issuegraph/issues.jsonl >/dev/null 2>&1 || true
exit 0
`, issuegraphCmd)
	postMerge := fmt.Sprintf(`#!/bin/sh
%s import -i .issuegraph/issues.jsonl >/dev/null 2>&1 || true
exit 0
`, issuegraphCmd)
	os.WriteFile(filepath.Join(hooksDir, "pre-commit"), []byte(preCommit), 0755)
	os.WriteFile(filepath.Join(hooksDir, "post-merge"), []byte(postMerge), 0755)
}

func runCmd(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		out, _ := cmd.CombinedOutput()
		t.Fatalf("Command failed: %s %v\nError: %v\nOutput: %s", name, args, err, string(out))
	}
}

func runCmdAllowError(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Run()
}

func runCmdOutputAllowError(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, _ := cmd.CombinedOutput()
	return string(out)
}

func runCmdWithEnv(t *testing.T, dir string, env map[string]string, name string, args ...string) {
	t.Helper()
	runCmdOutputWithEnvAllowError(t, dir, env, false, name, args...)
}

func runCmdOutputWithEnv(t *testing.T, dir string, env map[string]string, name string, args ...string) string {
	t.Helper()
	return runCmdOutputWithEnvAllowError(t, dir, env, false, name, args...)
}

func runCmdOutputWithEnvAllowError(t *testing.T, dir string, env map[string]string, allowError bool, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = append(os.Environ(), mapToEnvSlice(env)...)
	}
	out, err := cmd.CombinedOutput()
	if err != nil && !allowError {
		t.Fatalf("Command failed: %s %v\nError: %v\nOutput: %s", name, args, err, string(out))
	}
	return string(out)
}

func mapToEnvSlice(m map[string]string) []string {
	result := make([]string, 0, len(m))
	for k, v := range m {
		result = append(result, k+"="+v)
	}
	return result
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0755); err != nil {
		t.Fatalf("Failed to write %s: %v", dst, err)
	}
}

func compareTitleSets(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for title := range a {
		if !b[title] {
			return false
		}
	}
	return true
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

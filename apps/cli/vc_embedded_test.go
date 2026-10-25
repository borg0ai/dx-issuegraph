//go:build cgo

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

// issuegraphVC runs "issuegraph vc" with extra args. Returns combined output.
func issuegraphVC(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"vc"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	stdout, stderr, err := runCommandBuffers(t, cmd)
	if err != nil {
		t.Fatalf("issuegraph vc %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// issuegraphVCFail runs "issuegraph vc" expecting failure.
func issuegraphVCFail(t *testing.T, issuegraph, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"vc"}, args...)
	cmd := exec.Command(issuegraph, fullArgs...)
	cmd.Dir = dir
	cmd.Env = issuegraphEnv(dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("issuegraph vc %s should have failed, got: %s", strings.Join(args, " "), out)
	}
	return string(out)
}

func TestEmbeddedVC(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt vc tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)

	t.Run("status", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "vcst")

		out := issuegraphVC(t, issuegraph, dir, "status")
		if !strings.Contains(out, "main") {
			t.Errorf("expected 'main' branch in status, got: %s", out)
		}
	})

	t.Run("status_json", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "vcstj")

		out := issuegraphVC(t, issuegraph, dir, "status", "--json")
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\n%s", err, out)
		}
		if branch, _ := result["branch"].(string); branch != "main" {
			t.Errorf("expected branch='main', got %q", branch)
		}
		if commit, _ := result["commit"].(string); commit == "" {
			t.Error("expected non-empty commit hash")
		}
	})

	t.Run("commit_with_message", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "vccm")
		// issuegraph create auto-commits, so vc commit may see "nothing to commit".
		// Both outcomes are valid.
		issuegraphCreateSilent(t, issuegraph, dir, "commit test issue")

		cmd := exec.Command(issuegraph, "vc", "commit", "-m", "test commit message")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		out, err := cmd.CombinedOutput()
		if err != nil && !strings.Contains(string(out), "nothing to commit") {
			t.Fatalf("issuegraph vc commit failed unexpectedly: %v\n%s", err, out)
		}
	})

	t.Run("commit_clean_reports_nothing", func(t *testing.T) {
		// issuegraph init leaves the working set clean (bootstrap's own commit
		// tolerates nothing-to-commit, GH#3886). A second, truly no-op
		// commit here must be reported as "nothing to commit", not
		// silently treated as a fresh commit — this is the case
		// EmbeddedDoltStore.Commit's new nothing-to-commit tolerance made
		// possible to reach with a nil error.
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "vccln")

		cmd := exec.Command(issuegraph, "vc", "commit", "-m", "clean working set", "--json")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		stdout, stderr, err := runCommandBuffers(t, cmd)
		if err != nil {
			t.Fatalf("issuegraph vc commit --json failed unexpectedly: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}
		var result map[string]interface{}
		if jsonErr := json.Unmarshal(stdout.Bytes(), &result); jsonErr != nil {
			t.Fatalf("failed to parse JSON: %v\n%s", jsonErr, stdout.String())
		}
		if committed, _ := result["committed"].(bool); committed {
			t.Errorf("expected committed=false on a clean working set, got: %s", stdout.String())
		}
	})

	t.Run("commit_json", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "vccj")
		// issuegraph create auto-commits, so vc commit may see "nothing to commit".
		// Both committed=true and committed=false are valid outcomes.
		issuegraphCreateSilent(t, issuegraph, dir, "commit json issue")

		cmd := exec.Command(issuegraph, "vc", "commit", "-m", "json commit", "--json")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		stdout, stderr, err := runCommandBuffers(t, cmd)
		if err != nil {
			t.Fatalf("issuegraph vc commit --json failed unexpectedly: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}
		// Verify valid JSON with committed field (true or false are both valid)
		var result map[string]interface{}
		if jsonErr := json.Unmarshal(stdout.Bytes(), &result); jsonErr != nil {
			t.Fatalf("failed to parse JSON: %v\n%s", jsonErr, stdout.String())
		}
		if _, ok := result["committed"]; !ok {
			t.Error("expected 'committed' field in JSON output")
		}
	})

	t.Run("commit_stdin", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "vccs")
		issuegraphCreateSilent(t, issuegraph, dir, "stdin commit issue")

		cmd := exec.Command(issuegraph, "vc", "commit", "--stdin")
		cmd.Dir = dir
		cmd.Env = issuegraphEnv(dir)
		cmd.Stdin = strings.NewReader("message from stdin")
		out, err := cmd.CombinedOutput()
		if err != nil && !strings.Contains(string(out), "nothing to commit") {
			t.Fatalf("issuegraph vc commit --stdin failed unexpectedly: %v\n%s", err, out)
		}
	})

	t.Run("commit_no_message", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "vcnm")

		out := issuegraphVCFail(t, issuegraph, dir, "commit")
		if !strings.Contains(out, "commit message is required") {
			t.Errorf("expected 'commit message is required', got: %s", out)
		}
	})

	t.Run("merge_no_conflicts", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "vcmrg")

		// Create a branch, switch to it, create an issue, switch back, merge
		issuegraphBranch(t, issuegraph, dir, "feature-test")

		// Create an issue on main (auto-committed)
		issuegraphCreateSilent(t, issuegraph, dir, "main branch issue")

		// Merge the feature branch (which is at the same base commit)
		out := issuegraphVC(t, issuegraph, dir, "merge", "feature-test")
		if !strings.Contains(out, "merged") && !strings.Contains(out, "Merged") {
			t.Errorf("expected merge success message, got: %s", out)
		}
	})

	t.Run("merge_json", func(t *testing.T) {
		dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "vcmj")
		issuegraphBranch(t, issuegraph, dir, "merge-json-branch")

		out := issuegraphVC(t, issuegraph, dir, "merge", "merge-json-branch", "--json")
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\n%s", err, out)
		}
		if merged, _ := result["merged"].(string); merged != "merge-json-branch" {
			t.Errorf("expected merged='merge-json-branch', got %q", merged)
		}
	})
}

func TestEmbeddedVCConcurrent(t *testing.T) {
	if os.Getenv("ISSUEGRAPH_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set ISSUEGRAPH_TEST_EMBEDDED_DOLT=1 to run embedded dolt vc tests")
	}
	t.Parallel()

	issuegraph := buildEmbeddedBD(t)
	dir, _, _ := issuegraphInit(t, issuegraph, "--prefix", "vcconc")

	const numWorkers = 10

	type result struct {
		worker int
		out    string
		err    error
	}

	// Each worker creates an issue then commits
	results := make([]result, numWorkers)
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for w := 0; w < numWorkers; w++ {
		go func(worker int) {
			defer wg.Done()
			// Create an issue
			createCmd := exec.Command(issuegraph, "create", "--silent", fmt.Sprintf("vc-conc-issue-%d", worker))
			createCmd.Dir = dir
			createCmd.Env = issuegraphEnv(dir)
			if out, err := createCmd.CombinedOutput(); err != nil {
				results[worker] = result{worker: worker, out: string(out), err: err}
				return
			}

			// Try to commit
			cmd := exec.Command(issuegraph, "vc", "commit", "-m", fmt.Sprintf("worker %d commit", worker))
			cmd.Dir = dir
			cmd.Env = issuegraphEnv(dir)
			out, err := cmd.CombinedOutput()
			results[worker] = result{worker: worker, out: string(out), err: err}
		}(w)
	}
	wg.Wait()

	successes := 0
	for _, r := range results {
		if strings.Contains(r.out, "panic") {
			t.Errorf("worker %d panicked:\n%s", r.worker, r.out)
		}
		if r.err == nil {
			successes++
		} else if strings.Contains(r.out, "one writer at a time") ||
			strings.Contains(r.out, "nothing to commit") {
			// Expected: auto-commit means explicit commit often has nothing left
			successes++ // still counts as successful execution
		} else {
			t.Errorf("worker %d failed with unexpected error: %v\n%s", r.worker, r.err, r.out)
		}
	}
	if successes < 1 {
		t.Errorf("expected at least 1 successful commit, got %d", successes)
	}
	t.Logf("%d/%d vc commit workers succeeded", successes, numWorkers)
}
